package log

import (
	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/common"
)

// EntryGarbageCollector is responsible for cleaning
// up the entries which are already part of some snapshot
// since they are no longer required
type EntryGarbageCollector interface {
	common.ComponentLifecycle
	common.Pausable
	common.Freezable
}

var entryGC = "EGC"
var entryGCNotStartedErr = &common.ComponentHasNotStartedError{ComponentName: entryGC}
var entryGCIsDestroyedErr = &common.ComponentIsDestroyedError{ComponentName: entryGC}
var entryGCisFrozenErr = &common.ComponentIsFrozenError{ComponentName: entryGC}

// RealEntryGarbageCollector is the real-implementation
// of EntryGarbageCollector which is responsible for cleaning
// up the entries that are already part of the snapshot
type RealEntryGarbageCollector struct {
	// SnapshotHandler is required to get the
	// current snapshot index until which entries
	// can be garbage collected
	SnapshotHandler

	// EntryPersistence is used to remove entries
	// with given index.
	EntryPersistence

	// commandChannel is used to receive commands
	// from other components
	commandChannel chan entryGCCommand
}

// NewRealEntryGarbageCollector creates a new instance of
// real-entry garbage collector. It receives updates from
// the given snapshot handler and deletes the entries
// through entry persistence manager given
func NewRealEntryGarbageCollector(
	snapshotHandler SnapshotHandler,
	entryPersistence EntryPersistence,
) *RealEntryGarbageCollector {
	return &RealEntryGarbageCollector{
		SnapshotHandler:  snapshotHandler,
		EntryPersistence: entryPersistence,
		commandChannel:   make(chan entryGCCommand),
	}
}

// Start starts the component operations. This starts the
// Entry GC command server and starts receiving commands
func (egc *RealEntryGarbageCollector) Start() error {
	go egc.loop()
	return egc.sendLifecycleCommand(start)
}

// Destroy destroys the component and makes it irreversibly
// non-operational. Any command sent is not handled
func (egc *RealEntryGarbageCollector) Destroy() error {
	return egc.sendLifecycleCommand(destroy)
}

// Pause pauses the component operations provided it is not
// frozen. If it is frozen then it is an error
func (egc *RealEntryGarbageCollector) Pause() error {
	return egc.sendLifecycleCommand(pause)
}

// Resume resumes the component operations provided it is not
// frozen. If it is frozen then it is an error
func (egc *RealEntryGarbageCollector) Resume() error {
	return egc.sendLifecycleCommand(resume)
}

// Freeze makes the component state read-only
func (egc *RealEntryGarbageCollector) Freeze() error {
	return egc.sendLifecycleCommand(freeze)
}

// Unfreeze unfreezes the component state by one level. That is
// if freeze is called two times then unfreeze must be called
// twice as well.
func (egc *RealEntryGarbageCollector) Unfreeze() error {
	return egc.sendLifecycleCommand(unfreeze)
}

// sendLifecycleCommand sends the command related to the lifecycle of the component
// If there is any error then it is returned
func (egc *RealEntryGarbageCollector) sendLifecycleCommand(cmd commandType) error {
	errorChan := make(chan error)
	egc.commandChannel <- entryGCCommand{cmd: cmd, errChan: errorChan}
	return <-errorChan
}

type entryGCState struct {
	isStarted   bool
	isDestroyed bool
	freezeLevel uint64
}

// loop is the command server goroutine which receives commands
// and takes appropriate actions based on command and state
func (egc *RealEntryGarbageCollector) loop() {
	gcState := &entryGCState{
		isStarted:   false,
		isDestroyed: false,
		freezeLevel: 0,
	}
	for {
		c := <-egc.commandChannel
		switch c.cmd {
		case start:
			c.errChan <- egc.handleStart(gcState)
		case destroy:
			c.errChan <- egc.handleDestroy(gcState)
		case pause:
			c.errChan <- egc.handlePause(gcState)
		case resume:
			c.errChan <- egc.handleResume(gcState)
		case freeze:
			c.errChan <- egc.handleFreeze(gcState)
		case unfreeze:
			c.errChan <- egc.handleUnfreeze(gcState)
		}
	}
}

func (egc *RealEntryGarbageCollector) handleStart(gcState *entryGCState) error {
	if gcState.isDestroyed {
		return entryGCIsDestroyedErr
	}
	gcState.isStarted = true
	return nil
}

func (egc *RealEntryGarbageCollector) handleDestroy(gcState *entryGCState) error {
	gcState.isDestroyed = true
	return nil
}

func (egc *RealEntryGarbageCollector) handlePause(gcState *entryGCState) error {
	return nil
}

func (egc *RealEntryGarbageCollector) handleResume(gcState *entryGCState) error {
	if err := egc.checkOperationalStatus(gcState); err != nil {
		return nil
	}
	snapshotMetadata := egc.SnapshotHandler.GetSnapshotMetadata()
	go egc.triggerGarbageCollection(snapshotMetadata)
	return nil
}

func (egc *RealEntryGarbageCollector) triggerGarbageCollection(sm SnapshotMetadata) {
	if sm.Index == 0 {
		return
	}
	for i := sm.Index - 1; i > 0; i-- {
		err := egc.EntryPersistence.DeleteEntry(i)
		if err != nil {
			return
		}
		logrus.WithFields(logrus.Fields{
			logfield.Component: entryGC,
			logfield.Event:     "BG-CLEANUP",
		}).Debugf("cleaned up entry #%d", i)
	}
}

func (egc *RealEntryGarbageCollector) handleFreeze(gcState *entryGCState) error {
	if err := egc.checkWithoutFrozenness(gcState); err != nil {
		return err
	}
	gcState.freezeLevel++
	return nil
}

func (egc *RealEntryGarbageCollector) handleUnfreeze(gcState *entryGCState) error {
	if err := egc.checkWithoutFrozenness(gcState); err != nil {
		return nil
	}
	if gcState.freezeLevel > 0 {
		gcState.freezeLevel--
	}
	return nil
}

func (egc *RealEntryGarbageCollector) checkOperationalStatus(gcState *entryGCState) error {
	if err := egc.checkWithoutFrozenness(gcState); err != nil {
		return err
	}
	if egc.isFrozen(gcState) {
		return entryGCisFrozenErr
	}
	return nil
}

func (egc *RealEntryGarbageCollector) checkWithoutFrozenness(gcState *entryGCState) error {
	if !gcState.isStarted {
		return entryGCNotStartedErr
	}
	if gcState.isDestroyed {
		return entryGCIsDestroyedErr
	}
	return nil
}

func (egc *RealEntryGarbageCollector) isFrozen(gcState *entryGCState) bool {
	return gcState.freezeLevel > 0
}
