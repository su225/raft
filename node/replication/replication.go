package replication

import (
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/state"
)

// EntryReplicationController is responsible for controlling and
// coordinating the replication of log entries to the majority of
// nodes in the Raft cluster.
type EntryReplicationController interface {
	// ReplicateEntry replicates the given entry at the given entryID
	// to at least majority of the nodes in the cluster. If there is
	// an error during replication then it is returned. Replication
	// is still successful if there are errors in minority of them
	// since all it needs is the entry to be present in majority.
	ReplicateEntry(entryID log.EntryID, entry log.Entry) error

	// ComponentLifecycle makes the replication controller a component
	// with Start() and Destroy() lifecycle methods.
	common.ComponentLifecycle

	// Pausable makes the component pausable. That is - it can start
	// and stop operations without becoming non-functional
	common.Pausable
}

const replicationCtrl = "REPLICATION"

var replicationCtrlIsDestroyedErr = common.ComponentIsDestroyedError{ComponentName: replicationCtrl}

// RealEntryReplicationController is the implementation of replication
// controller which talks to other nodes in the effort of replicating
// a given entry to their logs. This must be active only when the node
// is the leader in the cluster
type RealEntryReplicationController struct {
	commandChannel  chan replicationControllerCommand
	listenerChannel chan state.RaftStateManagerEvent
}

// NewRealEntryReplicationController creates a new instance of real entry
// replication controller and returns the same
func NewRealEntryReplicationController() *RealEntryReplicationController {
	return &RealEntryReplicationController{
		commandChannel:  make(chan replicationControllerCommand),
		listenerChannel: make(chan state.RaftStateManagerEvent),
	}
}

// Start starts the component. It makes the component operational
func (r *RealEntryReplicationController) Start() error {
	return nil
}

// Destroy makes the component non-operational
func (r *RealEntryReplicationController) Destroy() error {
	return nil
}

// Pause pauses the component's operation, but unlike destroy it
// does not make the component non-operational
func (r *RealEntryReplicationController) Pause() error {
	return nil
}

// Resume resumes component's operation.
func (r *RealEntryReplicationController) Resume() error {
	return nil
}

// ReplicateEntry tries to replicate the given entry at the given entryID
// and returns nil if replication is successful or false otherwise.
func (r *RealEntryReplicationController) ReplicateEntry(entryID log.EntryID, entry log.Entry) error {
	return nil
}

type replicationControllerState struct {
	isStarted   bool
	isPaused    bool
	isDestroyed bool

	remoteNodeInfo cluster.NodeInfo
	nextIndex      uint64
	matchIndex     uint64
}

func (r *RealEntryReplicationController) loop(nodeInfo cluster.NodeInfo) {
	state := &replicationControllerState{
		isStarted:   true,
		isPaused:    true,
		isDestroyed: false,

		remoteNodeInfo: nodeInfo,
		nextIndex:      0,
		matchIndex:     0,
	}
	for {
		cmd := <-r.commandChannel
		switch c := cmd.(type) {
		case *replicationControllerDestroy:
			c.errorChan <- r.handleReplicationControllerDestroy(state, c)
		case *replicationControllerPause:
			c.errorChan <- r.handleReplicationControllerPause(state, c)
		case *replicationControllerResume:
			c.errorChan <- r.handleReplicationControllerResume(state, c)
		case *replicateEntry:
			c.errorChan <- r.handleReplicateEntry(state, c)
		}
	}
}

func (r *RealEntryReplicationController) handleReplicationControllerDestroy(state *replicationControllerState, cmd *replicationControllerDestroy) error {
	return nil
}

func (r *RealEntryReplicationController) handleReplicationControllerPause(state *replicationControllerState, cmd *replicationControllerPause) error {
	return nil
}

func (r *RealEntryReplicationController) handleReplicationControllerResume(state *replicationControllerState, cmd *replicationControllerResume) error {
	return nil
}

func (r *RealEntryReplicationController) handleReplicateEntry(state *replicationControllerState, cmd *replicateEntry) error {
	return nil
}

type replicationControllerListenerState struct {
	isDestroyed bool
}

func (r *RealEntryReplicationController) listener() {
	lisState := &replicationControllerListenerState{
		isDestroyed: false,
	}
	for {
		event := <-r.listenerChannel
		switch ev := event.(type) {
		case *state.UpgradeToLeaderEvent:
			r.onUpgradeToLeader(lisState, ev)
		case *state.BecomeCandidateEvent:
			r.onBecomeCandidate(lisState, ev)
		case *state.DowngradeToFollowerEvent:
			r.onDowngradeToFollower(lisState, ev)
		}
	}
}

// NotifyChannel is needed to subscribe to events from RaftStateManager
func (r *RealEntryReplicationController) NotifyChannel() chan<- state.RaftStateManagerEvent {
	return r.listenerChannel
}

func (r *RealEntryReplicationController) onUpgradeToLeader(state *replicationControllerListenerState, ev *state.UpgradeToLeaderEvent) error {
	return nil
}

func (r *RealEntryReplicationController) onBecomeCandidate(state *replicationControllerListenerState, ev *state.BecomeCandidateEvent) error {
	return nil
}

func (r *RealEntryReplicationController) onDowngradeToFollower(state *replicationControllerListenerState, ev *state.DowngradeToFollowerEvent) error {
	return nil
}
