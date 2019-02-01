package heartbeat

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/rpc"
	"github.com/su225/raft/node/state"
)

// LeaderHeartbeatController is responsible for controlling
// sending heartbeats to the remote nodes in the cluster
type LeaderHeartbeatController interface {
	// SendHeartbeat sends heartbeats to all remote nodes and
	// returns false if one of the nodes do not accept the
	// authority of this node as the leader
	SendHeartbeat() (acceptedAsLeader bool, heartbeatErr error)

	// ComponentLifecycle indicates that the LeaderHeartbeatController
	// must be a component which can be started and destroyed
	common.ComponentLifecycle

	// Pausable indicates that the operation of the component
	// can be paused and restarted at will
	common.Pausable
}

// RealLeaderHeartbeatController is responsible for sending
// heartbeats to remote nodes and determining if it should
// continue as the leader.
type RealLeaderHeartbeatController struct {
	// HeartbeatInterval is the time between two heartbeats
	HeartbeatInterval int64

	// RaftProtobufClient is required to send heartbeat
	// to the remote nodes in the cluster
	rpc.RaftProtobufClient

	// RaftStateManager is required to get the current term ID
	// to be included in heartbeat message header
	state.RaftStateManager

	// WriteAheadLogManager is required to get the current max
	// committed index to the remote node
	log.WriteAheadLogManager

	// MembershipManager is required to get the list of all nodes
	// known to this node in the cluster
	cluster.MembershipManager
	currentNodeID string

	commandChannel  chan heartbeatControllerCommand
	listenerChannel chan state.RaftStateManagerEvent
}

var heartbeatController = "HEARTBEAT"
var heartbeatControllerNotStartedError = &common.ComponentHasNotStartedError{ComponentName: heartbeatController}
var heartbeatControllerIsDestroyedError = &common.ComponentIsDestroyedError{ComponentName: heartbeatController}

// NewRealLeaderHeartbeatController creates a new instance of real
// leader heartbeat controller with given parameters and returns the same
// "heartbeatInterval" is the period between two heartbeats in milliseconds,
// "raftClient" is the client used to send heartbeat to remote nodes,
// "stateMgr" and "writeAheadLogMgr" are used to fetch the data to be
// included in the heartbeat. "membershipManager" is the list of all
// nodes to which heartbeat must be sent.
func NewRealLeaderHeartbeatController(
	currentNodeID string,
	heartbeatInterval int64,
	raftClient rpc.RaftProtobufClient,
	stateMgr state.RaftStateManager,
	writeAheadLogMgr log.WriteAheadLogManager,
	membershipMgr cluster.MembershipManager,
) *RealLeaderHeartbeatController {
	return &RealLeaderHeartbeatController{
		HeartbeatInterval:    heartbeatInterval,
		RaftProtobufClient:   raftClient,
		RaftStateManager:     stateMgr,
		WriteAheadLogManager: writeAheadLogMgr,
		MembershipManager:    membershipMgr,
		currentNodeID:        currentNodeID,
		commandChannel:       make(chan heartbeatControllerCommand),
		listenerChannel:      make(chan state.RaftStateManagerEvent),
	}
}

// Start starts the leader election controller component which then starts
// command server and raft state manager event notification listeners
func (h *RealLeaderHeartbeatController) Start() error {
	go h.commandServer()
	go h.listener()
	errorChannel := make(chan error)
	h.commandChannel <- &heartbeatControllerStart{errorChannel: errorChannel}
	return <-errorChannel
}

// Destroy destroys the leader election controller. It makes the component
// non-operational - that is, any operation invoked further returns error
func (h *RealLeaderHeartbeatController) Destroy() error {
	errorChannel := make(chan error)
	h.commandChannel <- &heartbeatControllerDestroy{errorChannel: errorChannel}
	return <-errorChannel
}

// Pause stops sending heartbeat to remote nodes. This happens when the node
// is not longer the leader of the cluster.
func (h *RealLeaderHeartbeatController) Pause() error {
	errorChannel := make(chan error)
	h.commandChannel <- &heartbeatControllerPause{errorChannel: errorChannel}
	return <-errorChannel
}

// Resume resumes sending heartbeat to remote nodes. This happens when the
// node is elected as leader and must establish authority over other nodes
func (h *RealLeaderHeartbeatController) Resume() error {
	errorChannel := make(chan error)
	h.commandChannel <- &heartbeatControllerResume{errorChannel: errorChannel}
	return <-errorChannel
}

// SendHeartbeat is responsible for sending heartbeat to all remote nodes
// It returns false if one of the nodes did not accept the authority of this
// node as the leader or true otherwise. It fails even if the node fails to
// reach the majority of nodes in the cluster.
func (h *RealLeaderHeartbeatController) SendHeartbeat() (bool, error) {
	currentTermID := h.RaftStateManager.GetCurrentTermID()
	currentLogMetadata, retrieveErr := h.WriteAheadLogManager.GetMetadata()
	if retrieveErr != nil {
		return false, h.stepDownFromTerm(currentTermID + 1)
	}
	nodesReached := int32(0)
	nodes := h.MembershipManager.GetAllNodes()

	hbWaitGroup := sync.WaitGroup{}
	hbWaitGroup.Add(len(nodes) - 1)

	acceptedAsLeaderIndicator := true
	authorityNotAccepted := sync.Once{}

	// Now send heartbeat to each node and wait for its response. If one of
	// them does not accept authority then record the fact. If the node is
	// reached and response arrived successfully then the node is "reached"
	for _, nodeInfo := range nodes {
		if nodeInfo.ID == h.currentNodeID {
			atomic.AddInt32(&nodesReached, 1)
			continue
		}
		go func(ni cluster.NodeInfo) {
			defer hbWaitGroup.Done()
			acceptedAsLeader, rpcErr := h.RaftProtobufClient.Heartbeat(currentTermID, ni.ID, currentLogMetadata.MaxCommittedIndex)
			if rpcErr != nil {
				logrus.WithFields(logrus.Fields{
					logfield.ErrorReason: rpcErr.Error(),
					logfield.Component:   heartbeatController,
					logfield.Event:       "SEND-HEARTBEAT",
				}).Errorf("cannot send heartbeat to %s", ni.ID)
				return
			}
			atomic.AddInt32(&nodesReached, 1)
			if !acceptedAsLeader {
				logrus.WithFields(logrus.Fields{
					logfield.Component: heartbeatController,
					logfield.Event:     "NO-AUTHORITY",
				}).Debugf("node %s does not accept authority", ni.ID)
				authorityNotAccepted.Do(func() { acceptedAsLeaderIndicator = false })
			}
		}(nodeInfo)
	}
	hbWaitGroup.Wait()

	// If one of the nodes do not accept the authority of this node
	// as leader then this node must step-down from leadership.
	if !acceptedAsLeaderIndicator {
		logrus.WithFields(logrus.Fields{
			logfield.Component: heartbeatController,
			logfield.Event:     "NO-AUTHORITY",
		}).Debugf("one of the nodes did not accept authority - step down")
		return false, h.stepDownFromTerm(currentTermID + 1)
	}

	// If this node cannot send heartbeat to majority of the nodes in
	// the cluster, then it must step-down
	majorityCount := int32(len(nodes)/2 + 1)
	if nodesReached < majorityCount {
		logrus.WithFields(logrus.Fields{
			logfield.Component: heartbeatController,
			logfield.Event:     "MAJORITY-NOT-REACHED",
		}).Debugf("reached %d, but need to reach %d",
			nodesReached, majorityCount)
		return false, h.stepDownFromTerm(currentTermID + 1)
	}
	return true, nil
}

// stepDownFromTerm steps down from leadership and switches to the next term
func (h *RealLeaderHeartbeatController) stepDownFromTerm(termID uint64) error {
	if opErr := h.RaftStateManager.DowngradeToFollower("", termID+1); opErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: opErr.Error(),
			logfield.Component:   heartbeatController,
			logfield.Event:       "STEP-DOWN",
		}).Errorf("error while stepping down")
		return opErr
	}
	return nil
}

type heartbeatControllerState struct {
	isStarted, isDestroyed, isPaused bool
}

// commandServer is the actual place where operations are executed. It is
// designed in alignment with Go's share by communicating principle. It removes
// lot of synchronization overhead since state is local and this one is single-threaded
func (h *RealLeaderHeartbeatController) commandServer() {
	state := &heartbeatControllerState{
		isStarted:   false,
		isDestroyed: false,
		isPaused:    true,
	}
	for {
		var timeoutChan <-chan time.Time
		if state.isStarted && !state.isDestroyed && !state.isPaused {
			timeoutChan = time.After(time.Duration(h.HeartbeatInterval) * time.Millisecond)
		}
		select {
		case cmd := <-h.commandChannel:
			switch c := cmd.(type) {
			case *heartbeatControllerStart:
				c.errorChannel <- h.handleHeartbeatControllerStart(state, c)
			case *heartbeatControllerDestroy:
				c.errorChannel <- h.handleHeartbeatControllerDestroy(state, c)
			case *heartbeatControllerPause:
				c.errorChannel <- h.handleHeartbeatControllerPause(state, c)
			case *heartbeatControllerResume:
				c.errorChannel <- h.handleHeartbeatControllerResume(state, c)
			}
		case <-timeoutChan:
		}
	}
}

func (h *RealLeaderHeartbeatController) handleHeartbeatControllerStart(state *heartbeatControllerState, cmd *heartbeatControllerStart) error {
	if state.isDestroyed {
		return heartbeatControllerIsDestroyedError
	}
	if state.isStarted {
		return nil
	}
	state.isStarted = true
	return nil
}

func (h *RealLeaderHeartbeatController) handleHeartbeatControllerDestroy(state *heartbeatControllerState, cmd *heartbeatControllerDestroy) error {
	if state.isDestroyed {
		return nil
	}
	state.isDestroyed = true
	return nil
}

func (h *RealLeaderHeartbeatController) handleHeartbeatControllerPause(state *heartbeatControllerState, cmd *heartbeatControllerPause) error {
	if statusErr := h.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	state.isPaused = true
	return nil
}

func (h *RealLeaderHeartbeatController) handleHeartbeatControllerResume(state *heartbeatControllerState, cmd *heartbeatControllerResume) error {
	if statusErr := h.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	state.isPaused = false
	return nil
}

func (h *RealLeaderHeartbeatController) handleTimeout(state *heartbeatControllerState) error {
	if statusErr := h.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	go h.SendHeartbeat()
	return nil
}

// listener listens to events from RaftStateManager and takes actions accordingly
// If the node becomes a leader then it starts sending heartbeats. If it is not
// then it pauses heartbeat controller.
func (h *RealLeaderHeartbeatController) listener() {
	listenerState := &heartbeatControllerState{
		isStarted:   true,
		isDestroyed: false,
		isPaused:    false,
	}
	for {
		event := <-h.listenerChannel
		switch ev := event.(type) {
		case *state.UpgradeToLeaderEvent:
			h.onUpgradeToLeader(listenerState, ev)
		case *state.BecomeCandidateEvent:
			h.onBecomeCandidate(listenerState, ev)
		case *state.DowngradeToFollowerEvent:
			h.onDowngradeToFollower(listenerState, ev)
		}
	}
}

// NotifyChannel returns the channel through which raft state manager event
// notification stream arrives
func (h *RealLeaderHeartbeatController) NotifyChannel() chan<- state.RaftStateManagerEvent {
	return h.listenerChannel
}

func (h *RealLeaderHeartbeatController) onUpgradeToLeader(listenerState *heartbeatControllerState, ev *state.UpgradeToLeaderEvent) error {
	if statusErr := h.checkOperationalStatus(listenerState); statusErr != nil {
		return statusErr
	}
	if opErr := h.Resume(); opErr != nil {
		return opErr
	}
	logrus.WithFields(logrus.Fields{
		logfield.Component: heartbeatController,
		logfield.Event:     "RESUME-HEARTBEAT",
	}).Debugf("now leader - start heartbeating for term %d", ev.TermID)
	return nil
}

func (h *RealLeaderHeartbeatController) onBecomeCandidate(listenerState *heartbeatControllerState, ev *state.BecomeCandidateEvent) error {
	if err := h.tryPauseHeartbeat(listenerState); err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		logfield.Component: heartbeatController,
		logfield.Event:     "PAUSE-HEARTBEAT",
	}).Debugf("now candidate - stop heartbeating for term %d", ev.TermID)
	return nil
}

func (h *RealLeaderHeartbeatController) onDowngradeToFollower(listenerState *heartbeatControllerState, ev *state.DowngradeToFollowerEvent) error {
	if err := h.tryPauseHeartbeat(listenerState); err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		logfield.Component: heartbeatController,
		logfield.Event:     "PAUSE-HEARTBEAT",
	}).Debugf("now follower - stop heartbeating for term %d", ev.TermID)
	return nil
}

func (h *RealLeaderHeartbeatController) tryPauseHeartbeat(listenerState *heartbeatControllerState) error {
	if statusErr := h.checkOperationalStatus(listenerState); statusErr != nil {
		return statusErr
	}
	if opErr := h.Pause(); opErr != nil {
		return opErr
	}
	return nil
}

func (h *RealLeaderHeartbeatController) checkOperationalStatus(state *heartbeatControllerState) error {
	if state.isDestroyed {
		return heartbeatControllerIsDestroyedError
	}
	if !state.isStarted {
		return heartbeatControllerNotStartedError
	}
	return nil
}
