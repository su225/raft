package log

import (
	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/common"
)

var gcComp = "GC"
var gcNotStartedErr = &common.ComponentHasNotStartedError{ComponentName: gcComp}
var gcIsDestroyedErr = &common.ComponentIsDestroyedError{ComponentName: gcComp}
var gcisFrozenErr = &common.ComponentIsFrozenError{ComponentName: gcComp}

// RealGarbageCollector is the real-implementation
// of EntryGarbageCollector which is responsible for cleaning
// up the entries that are already part of the snapshot
type RealGarbageCollector struct {
	// SnapshotHandler is required to get the
	// current snapshot index until which entries
	// can be garbage collected
	SnapshotHandler

	// EntryPersistence is used to remove entries
	// with given index.
	EntryPersistence

	// commandChannel is used to receive commands
	// from other components
	commandChannel chan gcCommand
}

// NewRealGarbageCollector creates a new instance of
// real-entry garbage collector. It receives updates from
// the given snapshot handler and deletes the entries
// through entry persistence manager given
func NewRealGarbageCollector(
	snapshotHandler SnapshotHandler,
	entryPersistence EntryPersistence,
) *RealGarbageCollector {
	return &RealGarbageCollector{
		SnapshotHandler:  snapshotHandler,
		EntryPersistence: entryPersistence,
		commandChannel:   make(chan gcCommand),
	}
}

// Start starts the component operations. This starts the
// Entry GC command server and starts receiving commands
func (gc *RealGarbageCollector) Start() error {
	go gc.loop()
	return gc.sendLifecycleCommand(start)
}

// Destroy destroys the component and makes it irreversibly
// non-operational. Any command sent is not handled
func (gc *RealGarbageCollector) Destroy() error {
	return gc.sendLifecycleCommand(destroy)
}

// Pause pauses the component operations provided it is not
// frozen. If it is frozen then it is an error
func (gc *RealGarbageCollector) Pause() error {
	return gc.sendLifecycleCommand(pause)
}

// Resume resumes the component operations provided it is not
// frozen. If it is frozen then it is an error
func (gc *RealGarbageCollector) Resume() error {
	return gc.sendLifecycleCommand(resume)
}

// Freeze makes the component state read-only
func (gc *RealGarbageCollector) Freeze() error {
	return gc.sendLifecycleCommand(freeze)
}

// Unfreeze unfreezes the component state by one level. That is
// if freeze is called two times then unfreeze must be called
// twice as well.
func (gc *RealGarbageCollector) Unfreeze() error {
	return gc.sendLifecycleCommand(unfreeze)
}

// sendLifecycleCommand sends the command related to the lifecycle of the component
// If there is any error then it is returned
func (gc *RealGarbageCollector) sendLifecycleCommand(cmd commandType) error {
	errorChan := make(chan error)
	gc.commandChannel <- gcCommand{cmd: cmd, errChan: errorChan}
	return <-errorChan
}

type entryGCState struct {
	isStarted   bool
	isDestroyed bool
	freezeLevel uint64
}

// loop is the command server goroutine which receives commands
// and takes appropriate actions based on command and state
func (gc *RealGarbageCollector) loop() {
	gcState := &entryGCState{
		isStarted:   false,
		isDestroyed: false,
		freezeLevel: 0,
	}
	for {
		c := <-gc.commandChannel
		switch c.cmd {
		case start:
			c.errChan <- gc.handleStart(gcState)
		case destroy:
			c.errChan <- gc.handleDestroy(gcState)
		case pause:
			c.errChan <- gc.handlePause(gcState)
		case resume:
			c.errChan <- gc.handleResume(gcState)
		case freeze:
			c.errChan <- gc.handleFreeze(gcState)
		case unfreeze:
			c.errChan <- gc.handleUnfreeze(gcState)
		}
	}
}

func (gc *RealGarbageCollector) handleStart(gcState *entryGCState) error {
	if gcState.isDestroyed {
		return gcIsDestroyedErr
	}
	gcState.isStarted = true
	return nil
}

func (gc *RealGarbageCollector) handleDestroy(gcState *entryGCState) error {
	gcState.isDestroyed = true
	return nil
}

func (gc *RealGarbageCollector) handlePause(gcState *entryGCState) error {
	return nil
}

func (gc *RealGarbageCollector) handleResume(gcState *entryGCState) error {
	if err := gc.checkOperationalStatus(gcState); err != nil {
		return nil
	}
	snapshotMetadata := gc.SnapshotHandler.GetSnapshotMetadata()
	go gc.triggerEntryGarbageCollection(snapshotMetadata)
	go gc.triggerSnapshotGarbageCollection(snapshotMetadata)
	return nil
}

func (gc *RealGarbageCollector) triggerEntryGarbageCollection(sm SnapshotMetadata) {
	if sm.Index == 0 {
		return
	}
	for i := sm.Index - 1; i > 0; i-- {
		err := gc.EntryPersistence.DeleteEntry(i)
		if err != nil {
			return
		}
		logrus.WithFields(logrus.Fields{
			logfield.Component: gcComp,
			logfield.Event:     "BG-CLEANUP",
		}).Debugf("cleaned up entry #%d", i)
	}
}

func (gc *RealGarbageCollector) triggerSnapshotGarbageCollection(sm SnapshotMetadata) {
	if sm.Epoch <= 1 {
		return
	}
	for i := sm.Epoch - 1; i >= 0; i-- {
		err := gc.SnapshotHandler.DeleteEpoch(i)
		if err != nil {
			return
		}
		logrus.WithFields(logrus.Fields{
			logfield.Component: gcComp,
			logfield.Event:     "BG-SNAP-CLEANUP",
		}).Debugf("cleaned up snapshot(epoch:%d)", i)
	}
}

func (gc *RealGarbageCollector) handleFreeze(gcState *entryGCState) error {
	if err := gc.checkWithoutFrozenness(gcState); err != nil {
		return err
	}
	gcState.freezeLevel++
	return nil
}

func (gc *RealGarbageCollector) handleUnfreeze(gcState *entryGCState) error {
	if err := gc.checkWithoutFrozenness(gcState); err != nil {
		return nil
	}
	if gcState.freezeLevel > 0 {
		gcState.freezeLevel--
	}
	return nil
}

func (gc *RealGarbageCollector) checkOperationalStatus(gcState *entryGCState) error {
	if err := gc.checkWithoutFrozenness(gcState); err != nil {
		return err
	}
	if gc.isFrozen(gcState) {
		return gcisFrozenErr
	}
	return nil
}

func (gc *RealGarbageCollector) checkWithoutFrozenness(gcState *entryGCState) error {
	if !gcState.isStarted {
		return gcNotStartedErr
	}
	if gcState.isDestroyed {
		return gcIsDestroyedErr
	}
	return nil
}

func (gc *RealGarbageCollector) isFrozen(gcState *entryGCState) bool {
	return gcState.freezeLevel > 0
}
