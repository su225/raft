package election

import (
	"time"

	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/state"
)

// LeaderElectionManager defines the operations that must
// be supported by any implementation of leader election manager.
// The implementation must be a pausable component.
type LeaderElectionManager interface {
	// ResetTimeout resets the election timeout.
	ResetTimeout() (resetErr error)

	// ComponentLifecylce indicates that the implementation
	// must be a component with Start() and Destroy() methods
	common.ComponentLifecycle

	// Pausable indicates that the component operation can be
	// paused and restarted when needed.
	common.Pausable
}

var leaderElectionMgr = "ELECTION-MGR"
var electionMgrNotStartedErr = &common.ComponentHasNotStartedError{ComponentName: leaderElectionMgr}
var electionMgrIsDestroyedErr = &common.ComponentIsDestroyedError{ComponentName: leaderElectionMgr}

// RealLeaderElectionManager is the implementation of leader election
// manager. This component is responsible for leader election process
// in the cluster. It waits for a certain time called electiom timeout.
// On timeout, it becomes candidate for the next term and requests other
// nodes for votes. If it gets votes from the majority of the nodes in
// the cluster then it is elected as the leader and it can start sending
// heartbeats to other cluster nodes to establish its authority.
type RealLeaderElectionManager struct {
	ElectionTimeoutInMillis uint64
	LeaderElectionAlgorithm

	commandChannel  chan leaderElectionManagerCommand
	listenerChannel chan state.RaftStateManagerEvent
}

// NewRealLeaderElectionManager creates a new instance of leader election
// manager with given state manager and write-ahead log manager instances.
func NewRealLeaderElectionManager(
	electionTimeout uint64,
	algo LeaderElectionAlgorithm,
) *RealLeaderElectionManager {
	return &RealLeaderElectionManager{
		ElectionTimeoutInMillis: electionTimeout,
		LeaderElectionAlgorithm: algo,
		commandChannel:          make(chan leaderElectionManagerCommand),
		listenerChannel:         make(chan state.RaftStateManagerEvent),
	}
}

// Start starts the command server as well as the listener which is
// responsible for listening to events from RaftStateManager and take
// appropriate actions
func (e *RealLeaderElectionManager) Start() error {
	go e.commandServer()
	go e.roleChangeListener()
	errorChan := make(chan error)
	e.commandChannel <- &leaderElectionManagerStart{errorChan: errorChan}
	return <-errorChan
}

// Destroy makes the component non-functional. It destroys both the
// listener as well as the command server
func (e *RealLeaderElectionManager) Destroy() error {
	errorChan := make(chan error)
	e.commandChannel <- &leaderElectionManagerDestroy{errorChan: errorChan}
	return <-errorChan
}

// Pause stops the command server operation temporarily. This is
// reversible unlike Destroy() which makes the component completely
// non-operational.
func (e *RealLeaderElectionManager) Pause() error {
	errorChan := make(chan error)
	e.commandChannel <- &leaderElectionManagerPause{errorChan: errorChan}
	return <-errorChan
}

// Resume resumes the command server operation. If the component is
// already destroyed then error is returned
func (e *RealLeaderElectionManager) Resume() error {
	errorChan := make(chan error)
	e.commandChannel <- &leaderElectionManagerResume{errorChan: errorChan}
	return <-errorChan
}

// ResetTimeout resets the leader election timeout
func (e *RealLeaderElectionManager) ResetTimeout() error {
	errorChan := make(chan error)
	e.commandChannel <- &leaderElectionManagerReset{errorChan: errorChan}
	return <-errorChan
}

type leaderElectionManagerState struct {
	isStarted, isDestroyed, isPaused bool
}

func (e *RealLeaderElectionManager) commandServer() {
	state := &leaderElectionManagerState{
		isStarted:   true,
		isDestroyed: false,
		isPaused:    false,
	}
	timeout := time.Duration(e.ElectionTimeoutInMillis) * time.Millisecond
	for {
		select {
		case cmd := <-e.commandChannel:
			switch c := cmd.(type) {
			case *leaderElectionManagerStart:
				c.errorChan <- e.handleLeaderElectionManagerStart(state, c)
			case *leaderElectionManagerDestroy:
				c.errorChan <- e.handleLeaderElectionManagerDestroy(state, c)
			case *leaderElectionManagerPause:
				c.errorChan <- e.handleLeaderElectionManagerPause(state, c)
			case *leaderElectionManagerResume:
				c.errorChan <- e.handleLeaderElectionManagerResume(state, c)
			case *leaderElectionManagerReset:
				c.errorChan <- e.handleLeaderElectionManagerReset(state, c)
			}
		case <-time.After(timeout):
			if !state.isPaused {
				e.handleTimeout(state)
			}
		}
	}
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerStart(state *leaderElectionManagerState, cmd *leaderElectionManagerStart) error {
	return nil
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerDestroy(state *leaderElectionManagerState, cmd *leaderElectionManagerDestroy) error {
	return nil
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerPause(state *leaderElectionManagerState, cmd *leaderElectionManagerPause) error {
	return nil
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerResume(state *leaderElectionManagerState, cmd *leaderElectionManagerResume) error {
	return nil
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerReset(state *leaderElectionManagerState, cmd *leaderElectionManagerReset) error {
	return nil
}

func (e *RealLeaderElectionManager) handleTimeout(state *leaderElectionManagerState) error {
	return nil
}

func (e *RealLeaderElectionManager) roleChangeListener() {
	listenerState := &leaderElectionManagerState{
		isStarted:   true,
		isDestroyed: false,
		isPaused:    false,
	}
	for {
		event := <-e.listenerChannel
		switch ev := event.(type) {
		case *state.UpgradeToLeaderEvent:
			e.onUpgradeToLeader(listenerState, ev)
		case *state.BecomeCandidateEvent:
			e.onBecomeCandidate(listenerState, ev)
		case *state.DowngradeToFollowerEvent:
			e.onDowngradeToFollower(listenerState, ev)
		}
	}
}

// NotifyChannel returns the channel from which the events arrive
func (e *RealLeaderElectionManager) NotifyChannel() chan<- state.RaftStateManagerEvent {
	return e.listenerChannel
}

func (e *RealLeaderElectionManager) onUpgradeToLeader(listenerState *leaderElectionManagerState, ev *state.UpgradeToLeaderEvent) error {
	return nil
}

func (e *RealLeaderElectionManager) onBecomeCandidate(listenerState *leaderElectionManagerState, ev *state.BecomeCandidateEvent) error {
	return nil
}

func (e *RealLeaderElectionManager) onDowngradeToFollower(listenerState *leaderElectionManagerState, ev *state.DowngradeToFollowerEvent) error {
	return nil
}
