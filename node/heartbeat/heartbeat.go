package heartbeat

import (
	"time"

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
	SendHeartbeat(termID uint64) (acceptedAsLeader bool, heartbeatErr error)

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

// NewRealLeaderHeartbeatController creates a new instance of real
// leader heartbeat controller with given parameters and returns the same
// "heartbeatInterval" is the period between two heartbeats in milliseconds,
// "raftClient" is the client used to send heartbeat to remote nodes,
// "stateMgr" and "writeAheadLogMgr" are used to fetch the data to be
// included in the heartbeat. "membershipManager" is the list of all
// nodes to which heartbeat must be sent.
func NewRealLeaderHeartbeatController(
	heartbeatInterval int64,
	raftClient rpc.RaftProtobufClient,
	stateMgr state.RaftStateManager,
	writeAheadLogMgr log.WriteAheadLogManager,
	membershipMgr cluster.MembershipManager,
	currentNodeID string,
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
	return nil
}

// Destroy destroys the leader election controller. It makes the component
// non-operational - that is, any operation invoked further returns error
func (h *RealLeaderHeartbeatController) Destroy() error {
	return nil
}

// Pause stops sending heartbeat to remote nodes. This happens when the node
// is not longer the leader of the cluster.
func (h *RealLeaderHeartbeatController) Pause() error {
	return nil
}

// Resume resumes sending heartbeat to remote nodes. This happens when the
// node is elected as leader and must establish authority over other nodes
func (h *RealLeaderHeartbeatController) Resume() error {
	return nil
}

// SendHeartbeat is responsible for sending heartbeat to all remote nodes
// It returns false if one of the nodes did not accept the authority of this
// node as the leader or true otherwise.
func (h *RealLeaderHeartbeatController) SendHeartbeat(termID uint64) (bool, error) {
	return false, nil
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
	return nil
}

func (h *RealLeaderHeartbeatController) handleHeartbeatControllerDestroy(state *heartbeatControllerState, cmd *heartbeatControllerDestroy) error {
	return nil
}

func (h *RealLeaderHeartbeatController) handleHeartbeatControllerPause(state *heartbeatControllerState, cmd *heartbeatControllerPause) error {
	return nil
}

func (h *RealLeaderHeartbeatController) handleHeartbeatControllerResume(state *heartbeatControllerState, cmd *heartbeatControllerResume) error {
	return nil
}

func (h *RealLeaderHeartbeatController) handleTimeout(state *heartbeatControllerState) error {
	return nil
}

// listener listens to events from RaftStateManager and takes actions accordingly
// If the node becomes a leader then it starts sending heartbeats. If it is not
// then it pauses heartbeat controller.
func (h *RealLeaderHeartbeatController) listener() {
	listenerState := &heartbeatControllerState{
		isStarted:   false,
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
	return nil
}

func (h *RealLeaderHeartbeatController) onBecomeCandidate(listenerState *heartbeatControllerState, ev *state.BecomeCandidateEvent) error {
	return nil
}

func (h *RealLeaderHeartbeatController) onDowngradeToFollower(listenerState *heartbeatControllerState, ev *state.DowngradeToFollowerEvent) error {
	return nil
}
