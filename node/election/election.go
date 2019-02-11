// Copyright (c) 2019 Suchith J N

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package election

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
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
	return nil
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

type electionTimerCommand int8

const (
	stopTimeout electionTimerCommand = iota
	resetTimeout
)

type leaderElectionManagerState struct {
	isDestroyed bool
	isPaused    bool
	cmdChan     chan electionTimerCommand
}

func (e *RealLeaderElectionManager) commandServer() {
	state := &leaderElectionManagerState{
		isDestroyed: false,
		isPaused:    true,
		cmdChan:     make(chan electionTimerCommand),
	}
	for {
		cmd := <-e.commandChannel
		switch c := cmd.(type) {
		case *leaderElectionManagerDestroy:
			c.errorChan <- e.handleLeaderElectionManagerDestroy(state, c)
		case *leaderElectionManagerPause:
			c.errorChan <- e.handleLeaderElectionManagerPause(state, c)
		case *leaderElectionManagerResume:
			c.errorChan <- e.handleLeaderElectionManagerResume(state, c)
		case *leaderElectionManagerReset:
			c.errorChan <- e.handleLeaderElectionManagerReset(state, c)
		}
	}
}

func (e *RealLeaderElectionManager) startElectionTimer(cmdChan <-chan electionTimerCommand) {
	go func() {
		stopTimer := false
		timeoutMultiplier := uint64(rand.Int63n(2)) + 1
		timeout := time.Duration(e.ElectionTimeoutInMillis*timeoutMultiplier) * time.Millisecond
		for !stopTimer {
			select {
			case c := <-cmdChan:
				switch c {
				case stopTimeout:
					stopTimer = true
				case resetTimeout:
					// Do-nothing.
				}
			case <-time.After(timeout):
				go e.LeaderElectionAlgorithm.ConductElection()
			}
		}
	}()
}

func (e *RealLeaderElectionManager) stopElectionTimer(cmdChan chan<- electionTimerCommand) {
	cmdChan <- stopTimeout
}

func (e *RealLeaderElectionManager) resetElectionTimer(cmdChan chan<- electionTimerCommand) {
	cmdChan <- resetTimeout
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerDestroy(state *leaderElectionManagerState, cmd *leaderElectionManagerDestroy) error {
	if state.isDestroyed {
		return nil
	}
	if !state.isPaused {
		e.stopElectionTimer(state.cmdChan)
		state.isPaused = true
	}
	state.isDestroyed = true
	logrus.WithFields(logrus.Fields{
		logfield.Component: leaderElectionMgr,
		logfield.Event:     "DESTROY",
	}).Infof("destroyed election manager")
	return nil
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerPause(state *leaderElectionManagerState, cmd *leaderElectionManagerPause) error {
	if state.isDestroyed {
		return electionMgrIsDestroyedErr
	}
	if !state.isPaused {
		e.stopElectionTimer(state.cmdChan)
		state.isPaused = true
	}
	return nil
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerResume(state *leaderElectionManagerState, cmd *leaderElectionManagerResume) error {
	if state.isDestroyed {
		return electionMgrIsDestroyedErr
	}
	if state.isPaused {
		e.startElectionTimer(state.cmdChan)
		state.isPaused = false
	}
	return nil
}

func (e *RealLeaderElectionManager) handleLeaderElectionManagerReset(state *leaderElectionManagerState, cmd *leaderElectionManagerReset) error {
	if state.isDestroyed {
		return electionMgrIsDestroyedErr
	}
	if !state.isPaused {
		e.resetElectionTimer(state.cmdChan)
	}
	return nil
}

func (e *RealLeaderElectionManager) roleChangeListener() {
	listenerState := &leaderElectionManagerState{
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
	logrus.WithFields(logrus.Fields{
		logfield.Component: leaderElectionMgr,
		logfield.Event:     "UPGRADE-TO-LEADER",
	}).Debugf("Leader now - stop election timer")
	return e.Pause()
}

func (e *RealLeaderElectionManager) onBecomeCandidate(listenerState *leaderElectionManagerState, ev *state.BecomeCandidateEvent) error {
	return e.Resume()
}

func (e *RealLeaderElectionManager) onDowngradeToFollower(listenerState *leaderElectionManagerState, ev *state.DowngradeToFollowerEvent) error {
	return e.Resume()
}
