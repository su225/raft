package log

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/data"
)

// SnapshotHandler is responsible for building snapshot
// out of the committed log entries. It is also responsible
// for building snapshot out of transferred key-value pairs
// if the node lags far behind
type SnapshotHandler interface {
	// ComponentLifecycle indicates that the snapshot handler
	// is a component with Start() and Destroy() lifecycle methods
	common.ComponentLifecycle

	// Recoverable indicates that the component has some state
	// that must be persisted across restarts and crashes
	common.Recoverable

	// Freezable indicates that the component can be stopped by
	// any other component and in order to continue all the
	// calls to freeze must agree to unfreeze.
	common.Freezable

	// RunSnapshotBuilder runs the snapshot building
	// process in the background. If the component is
	// frozen then it returns with an error
	RunSnapshotBuilder() error

	// StopSnapshotBuilder stops the snapshot builder if
	// it is running. If it already stopped then it is no-op
	// In other words this is idempotent.
	StopSnapshotBuilder() error

	// AddKeyValuePair adds a key-value pair to the snapshot for
	// the given epoch. If there is an error either due to epoch
	// bound violation or the persistence then it is returned. This
	// operation needs component not to be in frozen state and the
	// epoch must be greater than or equal to the current epoch
	AddKeyValuePair(epoch uint64, key string, value string) error

	// RemoveKeyValuePair removes the key-value pair for a given key.
	// It requires the epoch to be greater than or equal to the
	// current epoch. If there are any errors due to component state,
	// epoch bound violation or persistence issue it is reported.
	RemoveKeyValuePair(epoch uint64, key string) error

	// GetKeyValuePair gets the key-value pair from the snapshot
	// for the given epoch for the given key. This will work even
	// if the component is in frozen state. If there are any errors
	// related to epoch bound violation, component state and the
	// persistence errors then it is returned
	GetKeyValuePair(epoch uint64, key string) (value string, err error)

	// CreateEpoch creates a new snapshot epoch. The epochID specified
	// must be STRICTLY GREATER THAN the current epoch. Otherwise it
	// is results in epoch bound violation error. This operation
	// requires component to be in unfrozen state.
	CreateEpoch(epochID uint64) error

	// DeleteEpoch deletes the epoch. The epochID must be STRICTLY
	// LESS THAN the current epoch. Otherwise it is epoch bound error.
	// This operation needs component to be unfrozen.
	DeleteEpoch(epochID uint64) error

	// SetCurrentEpoch sets the current epoch. The epoch ID must be
	// be monotonically increasing. The given epochID must be STRICTLY
	// GREATER THAN the current epoch. Any error during the process is
	// returned. This needs component to be unfrozen
	SetCurrentEpoch(epochID uint64) (err error)

	// SetCurrentSnapshotIndex sets the current snapshot and persists
	// it to disk. If there is any error during the operation then
	// it is returned
	SetCurrentSnapshotIndex(index uint64) error

	// GetSnapshotMetadata returns the snapshot metadata containing
	// current snapshot index and current epoch
	GetSnapshotMetadata() SnapshotMetadata
}

const snapshotHandler = "SNAPSHOT"

var errSnapshotHandlerNotStarted = &common.ComponentHasNotStartedError{ComponentName: snapshotHandler}
var errSnapshotHandlerDestroyed = &common.ComponentIsDestroyedError{ComponentName: snapshotHandler}
var errSnapshotHandlerFrozen = &common.ComponentIsFrozenError{ComponentName: snapshotHandler}

// RealSnapshotHandler is the implementation of SnapshotHandler
// which derives its snapshot from the given write-ahead log manager
type RealSnapshotHandler struct {
	// parentWALog represents the parent write-ahead log from
	// which the snapshot is derived
	parentWALog WriteAheadLogManager

	// Persistence for snapshot and its metadata
	SnapshotPersistence
	SnapshotMetadataPersistence

	// Garbage collectors to clean things up once
	// snapshot is taken
	EntryGarbageCollector

	// Channel to send command to command handler
	commandChan chan snapshotHandlerCommand
}

// NewRealSnapshotHandler creates a new instance of real snapshot
// handler whose snapshot is derived from the committed entries of
// the log managed by the given write-ahead log manager.
func NewRealSnapshotHandler(
	parentWALog WriteAheadLogManager,
	snapPersistence SnapshotPersistence,
	snapMetaPersistence SnapshotMetadataPersistence,
	entryGC EntryGarbageCollector,
) *RealSnapshotHandler {
	return &RealSnapshotHandler{
		parentWALog:                 parentWALog,
		SnapshotPersistence:         snapPersistence,
		SnapshotMetadataPersistence: snapMetaPersistence,
		EntryGarbageCollector:       entryGC,
		commandChan:                 make(chan snapshotHandlerCommand),
	}
}

// Start makes the component operational and initializes necessary
// state required for the functioning of SnapshotHandler
func (sh *RealSnapshotHandler) Start() error {
	go sh.loop()
	errorChan := make(chan error)
	sh.commandChan <- &snapshotHandlerStart{errorChan: errorChan}
	if err := <-errorChan; err != nil {
		return err
	}
	return sh.Recover()
}

// Destroy makes the component non-functional irreversibly and cleans
// up the resources used if necessary
func (sh *RealSnapshotHandler) Destroy() error {
	errorChan := make(chan error)
	sh.commandChan <- &snapshotHandlerDestroy{errorChan: errorChan}
	return <-errorChan
}

// Recover recovers the state of the SnapshotHandler from the crash like
// the snapshot index and epoch metadata. If this fails then the node
// should not start since it could return incorrect results if snapshot
// handler is used for querying.
func (sh *RealSnapshotHandler) Recover() error {
	errorChan := make(chan error)
	sh.commandChan <- &snapshotHandlerRecover{errorChan: errorChan}
	return <-errorChan
}

// Freeze temporarily makes the component non-functional. Unlike pause this
// is not idempotent. If freeze is called twice, then unfreeze must be called
// twice to make the component operational again. When the component is in
// frozen state it cannot be destroyed as well.
func (sh *RealSnapshotHandler) Freeze() error {
	errorChan := make(chan error)
	sh.commandChan <- &snapshotHandlerFreeze{errorChan: errorChan}
	return <-errorChan
}

// Unfreeze removes one level of freezing. If the number of levels of freezing
// goes to zero then the component would be operational again. Note that calling
// unfreeze on an unfrozen component is a no-op
func (sh *RealSnapshotHandler) Unfreeze() error {
	errorChan := make(chan error)
	sh.commandChan <- &snapshotHandlerUnfreeze{errorChan: errorChan}
	return <-errorChan
}

// RunSnapshotBuilder runs the snapshot builder process in the background. If
// the component is frozen, destroyed or not started then it returns appropriate error
func (sh *RealSnapshotHandler) RunSnapshotBuilder() error {
	errorChan := make(chan error)
	sh.commandChan <- &runSnapshotBuilder{errorChan: errorChan}
	return <-errorChan
}

// StopSnapshotBuilder stops the snapshot builder process in the background. If
// the component is frozen, destroyed or not started then it returns appropriate error.
// If the snapshot building process has already stopped then it is a no-op
func (sh *RealSnapshotHandler) StopSnapshotBuilder() error {
	errorChan := make(chan error)
	sh.commandChan <- &stopSnapshotBuilder{errorChan: errorChan}
	return <-errorChan
}

// AddKeyValuePair adds a new key-value pair to the snapshot with the given epoch
// If there is any error in the operation then it is returned. The epoch must be
// greater than or equal to the current epoch
func (sh *RealSnapshotHandler) AddKeyValuePair(epoch uint64, key string, value string) error {
	errorChan := make(chan error)
	sh.commandChan <- &addKVPair{
		epoch:     epoch,
		key:       key,
		value:     value,
		errorChan: errorChan,
	}
	return <-errorChan
}

// RemoveKeyValuePair removes the key-value pair from the snapshot with the given
// epoch. If there is any error in the operation then it is returned. The epoch must
// greater than or equal to the current epoch.
func (sh *RealSnapshotHandler) RemoveKeyValuePair(epoch uint64, key string) error {
	errorChan := make(chan error)
	sh.commandChan <- &removeKVPair{
		epoch:     epoch,
		key:       key,
		errorChan: errorChan,
	}
	return <-errorChan
}

// GetKeyValuePair returns the key-value pair from the snapshot with given epoch if the
// epoch is valid and the key exists. The epoch must be greater than or equal to current
// epoch as previous epochs are considered garbage
func (sh *RealSnapshotHandler) GetKeyValuePair(epoch uint64, key string) (string, error) {
	replyChan := make(chan *getKVPairReply)
	sh.commandChan <- &getKVPair{
		epoch:     epoch,
		key:       key,
		replyChan: replyChan,
	}
	reply := <-replyChan
	return reply.value, reply.err
}

// CreateEpoch creates a new epoch with the given epoch ID. If the new epoch is less than
// or equal to the current epoch or if there is already an epoch then it is an error
func (sh *RealSnapshotHandler) CreateEpoch(epochID uint64) error {
	errorChan := make(chan error)
	sh.commandChan <- &createEpoch{
		epoch:     epochID,
		errorChan: errorChan,
	}
	return <-errorChan
}

// DeleteEpoch deletes the given epoch. The epoch must be strictly less than the current
// epoch. This is used mostly for garbage collection purposes
func (sh *RealSnapshotHandler) DeleteEpoch(epochID uint64) error {
	errorChan := make(chan error)
	sh.commandChan <- &deleteEpoch{
		epoch:     epochID,
		errorChan: errorChan,
	}
	return <-errorChan
}

// SetCurrentEpoch switches the current epoch. The epoch must be monotonically increasing.
// So epochID must be greater than or equal to current epoch.
func (sh *RealSnapshotHandler) SetCurrentEpoch(epochID uint64) error {
	errorChan := make(chan error)
	sh.commandChan <- &setCurrentEpoch{
		epoch:     epochID,
		errorChan: errorChan,
	}
	return <-errorChan
}

// GetSnapshotMetadata returns current snapshot metadata.
func (sh *RealSnapshotHandler) GetSnapshotMetadata() SnapshotMetadata {
	replyChan := make(chan SnapshotMetadata)
	sh.commandChan <- &getSnapshotMetadata{replyChan: replyChan}
	return <-replyChan
}

// SetCurrentSnapshotIndex sets the current snapshot index to the given one
func (sh *RealSnapshotHandler) SetCurrentSnapshotIndex(index uint64) error {
	errorChan := make(chan error)
	sh.commandChan <- &setCurrentSnapshotIndex{
		index:     index,
		errorChan: errorChan,
	}
	return <-errorChan
}

func (sh *RealSnapshotHandler) terminateSnapshotBuilder() error {
	errorChan := make(chan error)
	sh.commandChan <- &terminateSnapshotBuilder{errorChan: errorChan}
	return <-errorChan
}

type snapshotHandlerState struct {
	isStarted               bool
	isDestroyed             bool
	isRunning               bool
	snapshotBuilderStopChan chan struct{}
	freezeLevel             uint64
	SnapshotMetadata
}

func (sh *RealSnapshotHandler) loop() {
	state := &snapshotHandlerState{
		isStarted:               false,
		isDestroyed:             false,
		isRunning:               false,
		freezeLevel:             0,
		snapshotBuilderStopChan: make(chan struct{}),
		SnapshotMetadata: SnapshotMetadata{
			Index: 0,
			Epoch: 1,
		},
	}
	for {
		cmd := <-sh.commandChan
		switch c := cmd.(type) {
		case *snapshotHandlerStart:
			c.errorChan <- sh.handleSnapshotHandlerStart(state)
		case *snapshotHandlerDestroy:
			c.errorChan <- sh.handleSnapshotHandlerDestroy(state)
		case *snapshotHandlerRecover:
			c.errorChan <- sh.handleSnapshotHandlerRecover(state)
		case *snapshotHandlerFreeze:
			c.errorChan <- sh.handleSnapshotHandlerFreeze(state)
		case *snapshotHandlerUnfreeze:
			c.errorChan <- sh.handleSnapshotHandlerUnfreeze(state)

		case *runSnapshotBuilder:
			c.errorChan <- sh.handleRunSnapshotBuilder(state)
		case *stopSnapshotBuilder:
			c.errorChan <- sh.handleStopSnapshotBuilder(state)
		case *terminateSnapshotBuilder:
			c.errorChan <- sh.handleTerminateSnapshotBuilder(state)

		case *addKVPair:
			c.errorChan <- sh.handleAddKVPair(state, c)
		case *removeKVPair:
			c.errorChan <- sh.handleRemoveKVPair(state, c)
		case *getKVPair:
			c.replyChan <- sh.handleGetKVPair(state, c)

		case *createEpoch:
			c.errorChan <- sh.handleCreateEpoch(state, c)
		case *deleteEpoch:
			c.errorChan <- sh.handleDeleteEpoch(state, c)

		case *getSnapshotMetadata:
			c.replyChan <- sh.handleGetSnapshotMetadata(state)
		case *setCurrentEpoch:
			c.errorChan <- sh.handleSetCurrentEpoch(state, c)
		case *setCurrentSnapshotIndex:
			c.errorChan <- sh.handleSetCurrentSnapshotIndex(state, c)
		}
	}
}

// handleSnapshotHandlerStart handles the start of the snapshot handler.
func (sh *RealSnapshotHandler) handleSnapshotHandlerStart(state *snapshotHandlerState) error {
	if state.isDestroyed {
		return errSnapshotHandlerDestroyed
	}
	go func() {
		if err := sh.EntryGarbageCollector.Start(); err != nil {
			logrus.WithFields(logrus.Fields{
				logfield.ErrorReason: err.Error(),
				logfield.Component:   snapshotHandler,
				logfield.Event:       "START-EGC",
			}).Errorf("error while starting entry garbage collector")
			return
		}
		logrus.WithFields(logrus.Fields{
			logfield.Component: snapshotHandler,
			logfield.Event:     "START-EGC",
		}).Infof("started entry garbage collector")
	}()
	state.isStarted = true
	return nil
}

// handleSnapshotHandlerDestroy handles destruction of snapshot handler.
// It requires the component not to be frozen
func (sh *RealSnapshotHandler) handleSnapshotHandlerDestroy(state *snapshotHandlerState) error {
	if state.isDestroyed {
		return nil
	}
	if sh.isFrozen(state) {
		return errSnapshotHandlerFrozen
	}
	go func() {
		if err := sh.EntryGarbageCollector.Destroy(); err != nil {
			logrus.WithFields(logrus.Fields{
				logfield.ErrorReason: err.Error(),
				logfield.Component:   snapshotHandler,
				logfield.Event:       "DESTROY-EGC",
			}).Errorf("error while destroying entry garbage collector")
		}
		logrus.WithFields(logrus.Fields{
			logfield.Component: snapshotHandler,
			logfield.Event:     "DESTROY-EGC",
		}).Infof("destroyed entry garbage collector")
	}()
	sh.handleStopSnapshotBuilder(state)
	state.isDestroyed = true
	return nil
}

// handleSnapshotHandlerRecover handles recovery process of the snapshot handler. If there
// is any error during recovery then it is reported.
func (sh *RealSnapshotHandler) handleSnapshotHandlerRecover(state *snapshotHandlerState) error {
	if state.isDestroyed {
		return errSnapshotHandlerDestroyed
	}
	if sh.isFrozen(state) {
		return errSnapshotHandlerFrozen
	}
	metadata, retrieveErr := sh.SnapshotMetadataPersistence.RetrieveMetadata()
	if retrieveErr != nil {
		if os.IsNotExist(retrieveErr) {
			state.SnapshotMetadata = SnapshotMetadata{Epoch: 1, Index: 0}
			sh.SnapshotPersistence.StartEpoch(state.Epoch)
			return nil
		}
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: retrieveErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "RECOVER",
		}).Errorf("error while retrieving snapshot metadata")
		return retrieveErr
	}
	state.SnapshotMetadata = *metadata
	sh.SnapshotPersistence.StartEpoch(state.Epoch)
	return nil
}

// handleSnapshotHandlerFreeze freezes the SnapshotHandler by one level. This makes
// the component state read-only.
func (sh *RealSnapshotHandler) handleSnapshotHandlerFreeze(state *snapshotHandlerState) error {
	if statusErr := sh.checkWithoutFrozenness(state); statusErr != nil {
		return statusErr
	}
	state.freezeLevel++
	return nil
}

// handleSnapshotHandlerUnfreeze unfreezes the SnapshotHandler by one level. But this
// does not start the snapshot building process though
func (sh *RealSnapshotHandler) handleSnapshotHandlerUnfreeze(state *snapshotHandlerState) error {
	if statusErr := sh.checkWithoutFrozenness(state); statusErr != nil {
		return statusErr
	}
	if state.freezeLevel > 0 {
		state.freezeLevel--
	}
	return nil
}

// handleRunSnapshotBuilder runs the snapshot builder which picks each entry from the write-ahead
// log and applies it to the snapshot in the current epoch. If it runs out of entries to apply
// then it just stops. This is a background process that can be started or stopped
func (sh *RealSnapshotHandler) handleRunSnapshotBuilder(state *snapshotHandlerState) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	if state.isRunning {
		return nil
	}
	go sh.runSnapshotBuilder(state.snapshotBuilderStopChan)
	state.isRunning = true
	return nil
}

func (sh *RealSnapshotHandler) handleStopSnapshotBuilder(state *snapshotHandlerState) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	if !state.isRunning {
		return nil
	}
	// It asynchronously sends stop signal to the snapshot builder
	// which in turn notifies the command server to flip the running
	// switch to false to complete the stop process
	go func(stopChan chan<- struct{}) {
		stopChan <- struct{}{}
	}(state.snapshotBuilderStopChan)
	return nil
}

// handleTerminateSnapshotBuilder is an unfortunate creation because of the way these things are
// designed. That is state is not shared outside command server goroutine. So all of this had to be done
func (sh *RealSnapshotHandler) handleTerminateSnapshotBuilder(state *snapshotHandlerState) error {
	if !state.isRunning {
		return nil
	}
	state.isRunning = false
	return nil
}

// runSnapshotBuilder runs the snapshot builder for the given epoch. It iterates through each
// entry in the write-ahead log and takes appropriate actions based on entry.
func (sh *RealSnapshotHandler) runSnapshotBuilder(stopSignal <-chan struct{}) {
	stopSnapshotBuilder := false
	for !stopSnapshotBuilder {
		select {
		case <-stopSignal:
			stopSnapshotBuilder = true
		default:
			snapshotMetadata := sh.GetSnapshotMetadata()
			epoch := snapshotMetadata.Epoch
			curSnapshotIndex := snapshotMetadata.Index
			metadata, _ := sh.parentWALog.GetMetadata()
			curCommittedIndex := metadata.MaxCommittedIndex
			if curSnapshotIndex == curCommittedIndex {
				stopSnapshotBuilder = true
				continue
			}
			nextIndex := curSnapshotIndex + 1
			if err := sh.applyEntryToSnapshot(epoch, nextIndex); err != nil {
				logrus.WithFields(logrus.Fields{
					logfield.ErrorReason: err.Error(),
					logfield.Component:   snapshotHandler,
					logfield.Event:       "APPLY",
				}).Errorf("unable to apply entry")
				stopSnapshotBuilder = true
				continue
			}
			if err := sh.SetCurrentSnapshotIndex(nextIndex); err != nil {
				logrus.WithFields(logrus.Fields{
					logfield.ErrorReason: err.Error(),
					logfield.Component:   snapshotHandler,
					logfield.Event:       "SET-CURRENT-SNAPIDX",
				}).Errorf("error while setting snapshot index to %d", nextIndex)
				stopSnapshotBuilder = true
			}
			go sh.EntryGarbageCollector.Resume()
		}
	}
	sh.terminateSnapshotBuilder()
}

// applyEntryToSnapshot fetches the snapshot entry corresponding to the given
// index and applies it to the snapshot with given epoch
func (sh *RealSnapshotHandler) applyEntryToSnapshot(epoch, index uint64) error {
	entry, err := sh.parentWALog.GetEntry(index)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: err.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "FETCH-ENTRY",
		}).Errorf("error while fetching entry #%d", index)
		return err
	}
	if err := sh.doApplyEntry(epoch, index, entry); err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: err.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "APPLY-ENTRY",
		}).Errorf("failed to apply entry #%d to snapshot(epoch:%d)",
			index, epoch)
		return err
	}
	return nil
}

// doApplyEntry applies the given entry at the given index to the snapshot of
// a particular epoch. If there is an error in the process then it is returned
func (sh *RealSnapshotHandler) doApplyEntry(epoch, index uint64, entry Entry) error {
	var execEntryErr error
	switch e := entry.(type) {
	case *UpsertEntry:
		execEntryErr = sh.doApplyUpsertEntry(epoch, index, e)
	case *DeleteEntry:
		execEntryErr = sh.doApplyDeleteEntry(epoch, index, e)
	case *SentinelEntry:
		logrus.WithFields(logrus.Fields{
			logfield.Component: snapshotHandler,
			logfield.Event:     "EXEC-LOG-OP",
		}).Warnf("sentinel entry found for entry #%d", index)
	}
	if execEntryErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: execEntryErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "EXEC-LOG-OP",
		}).Errorf("error while applying entry #%d to snapshot(epoch:%d)", index, epoch)
	}
	return execEntryErr
}

// doApplyUpsertEntry applies the upsert entry to the snapshot with given epoch.
// If there is an error then it is returned
func (sh *RealSnapshotHandler) doApplyUpsertEntry(epoch, index uint64, upsert *UpsertEntry) error {
	key, value := upsert.Key, upsert.Value
	if err := sh.AddKeyValuePair(epoch, key, value); err != nil {
		return err
	}
	return nil
}

// doApplyDeleteEntry deletes the key-value pair with the given key in the snapshot
// with the given epoch. If it does not exist or already deleted then it is a no-op
func (sh *RealSnapshotHandler) doApplyDeleteEntry(epoch, index uint64, delEntry *DeleteEntry) error {
	key := delEntry.Key
	if err := sh.RemoveKeyValuePair(epoch, key); err != nil {
		return err
	}
	return nil
}

// handleAddKVPair adds the key-value pair to the given epoch of the snapshot. If the key already
// exists then the value is updated. If there is any error during the process then it is returned
// (Possible errors - epoch bound violation, persistence related errors)
func (sh *RealSnapshotHandler) handleAddKVPair(state *snapshotHandlerState, cmd *addKVPair) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	if epochBoundErr := sh.checkEpochLowerBound(state.Epoch, cmd.epoch, true); epochBoundErr != nil {
		return epochBoundErr
	}
	kvPair := data.KVPair{Key: cmd.key, Value: cmd.value}
	if persistErr := sh.SnapshotPersistence.PersistKeyValuePair(cmd.epoch, kvPair); persistErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: persistErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "PERSIST-KV",
		}).Errorf("error while persisting key %s in snapshot(epoch:%d)",
			cmd.key, cmd.epoch)
		return persistErr
	}
	return nil
}

// handleRemoveKVPair removes the given key-value pair from the given epoch of the snapshot.
// If there is an issue with epoch bound or removal operation then it is reported.
func (sh *RealSnapshotHandler) handleRemoveKVPair(state *snapshotHandlerState, cmd *removeKVPair) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	if epochBoundErr := sh.checkEpochLowerBound(state.Epoch, cmd.epoch, true); epochBoundErr != nil {
		return epochBoundErr
	}
	if deleteErr := sh.SnapshotPersistence.DeleteKeyValuePair(cmd.epoch, cmd.key); deleteErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: deleteErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "DELETE-KV",
		}).Errorf("error while deleting key %s in snapshot(epoch:%d)",
			cmd.key, cmd.epoch)
		return deleteErr
	}
	return nil
}

// handleGetKVPair returns the key-value pair for the given key from the snapshot
// for the given epoch. If the given epoch is less than the current epoch then it is invalid.
// If there is any error while retrieving the key-value pair then it is returned
func (sh *RealSnapshotHandler) handleGetKVPair(state *snapshotHandlerState, cmd *getKVPair) *getKVPairReply {
	reply := &getKVPairReply{
		key:   cmd.key,
		value: "",
	}
	if statusErr := sh.checkWithoutFrozenness(state); statusErr != nil {
		reply.err = statusErr
		return reply
	}
	currentEpoch := state.SnapshotMetadata.Epoch
	if epochBoundErr := sh.checkEpochLowerBound(currentEpoch, cmd.epoch, true); epochBoundErr != nil {
		reply.err = epochBoundErr
		return reply
	}
	kvPair, retrieveErr := sh.SnapshotPersistence.RetrieveKeyValuePair(cmd.epoch, cmd.key)
	if retrieveErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: retrieveErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "GET-KV",
		}).Errorf("error while retrieving value for key %s in snapshot(epoch:%d)",
			cmd.key, cmd.epoch)
		reply.err = retrieveErr
		return reply
	}
	reply.value = kvPair.Value
	reply.err = nil
	return reply
}

// handleCreateEpoch creates a new epoch. The new epoch should be greater than the current
// epoch. Otherwise epoch bound error is returned. If there is any error then it is returned
func (sh *RealSnapshotHandler) handleCreateEpoch(state *snapshotHandlerState, cmd *createEpoch) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	currentEpoch := state.SnapshotMetadata.Epoch
	if epochBoundErr := sh.checkEpochLowerBound(currentEpoch, cmd.epoch, false); epochBoundErr != nil {
		return epochBoundErr
	}
	if epochCreateErr := sh.SnapshotPersistence.StartEpoch(cmd.epoch); epochCreateErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: epochCreateErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "CREATE-EPOCH",
		}).Errorf("error while creating epoch %d", cmd.epoch)
		return epochCreateErr
	}
	return nil
}

// handleDeleteEpoch deletes the snapshot corresponding to the given epoch. If the epoch is greater
// than or equal to the current epoch then error is returned
func (sh *RealSnapshotHandler) handleDeleteEpoch(state *snapshotHandlerState, cmd *deleteEpoch) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	currentEpoch := state.SnapshotMetadata.Epoch
	if epochBoundErr := sh.checkEpochUpperBound(currentEpoch, cmd.epoch, false); epochBoundErr != nil {
		return epochBoundErr
	}
	if epochDeleteErr := sh.SnapshotPersistence.DeleteEpoch(cmd.epoch); epochDeleteErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: epochDeleteErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "DELETE-EPOCH",
		}).Errorf("error while deleting epoch %d", cmd.epoch)
		return epochDeleteErr
	}
	return nil
}

// handleSetCurrentEpoch sets the current epoch provided the component is operational.
// If the new epoch is less than or equal to given epoch then error is returned
func (sh *RealSnapshotHandler) handleSetCurrentEpoch(state *snapshotHandlerState, cmd *setCurrentEpoch) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	epochBeforeUpdate := state.SnapshotMetadata.Epoch
	if epochBoundErr := sh.checkEpochLowerBound(epochBeforeUpdate, cmd.epoch, false); epochBoundErr != nil {
		return epochBoundErr
	}
	state.SnapshotMetadata.Epoch = cmd.epoch
	persistErr := sh.SnapshotMetadataPersistence.PersistMetadata(&state.SnapshotMetadata)
	if persistErr != nil {
		state.SnapshotMetadata.Epoch = epochBeforeUpdate
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: persistErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "SET-EPOCH",
		}).Errorf("error while persisting snapshot metadata for switch-epoch (%d -> %d)",
			epochBeforeUpdate, cmd.epoch)
		return persistErr
	}
	logrus.WithFields(logrus.Fields{
		logfield.Component: snapshotHandler,
		logfield.Event:     "SET-EPOCH",
	}).Debugf("switched snapshot epoch: %d -> %d", epochBeforeUpdate, cmd.epoch)
	return nil
}

// handleSetCurrentSnapshotIndex sets the current snapshot index. If the component is not operational
// then it returns an error complaining the same. The component must be in unfrozen state for this to
// work otherwise it is an error
func (sh *RealSnapshotHandler) handleSetCurrentSnapshotIndex(state *snapshotHandlerState, cmd *setCurrentSnapshotIndex) error {
	if statusErr := sh.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	beforeUpdateIndex := state.SnapshotMetadata.Index
	state.SnapshotMetadata.Index = cmd.index
	if persistErr := sh.SnapshotMetadataPersistence.PersistMetadata(&state.SnapshotMetadata); persistErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: persistErr.Error(),
			logfield.Component:   snapshotHandler,
			logfield.Event:       "PERSIST",
		}).Errorf("error while persisting snapshot metadata")
		state.SnapshotMetadata.Index = beforeUpdateIndex
		return persistErr
	}
	return nil
}

func (sh *RealSnapshotHandler) handleGetSnapshotMetadata(state *snapshotHandlerState) SnapshotMetadata {
	if statusErr := sh.checkWithoutFrozenness(state); statusErr != nil {
		return SnapshotMetadata{Index: 0, Epoch: 0}
	}
	return state.SnapshotMetadata
}

// checkEpoch checks if the given epoch satisfies the upper and lower bounds. If one of the
// bounds is not satisfied then an error is returned. (specifically InvalidEpochError)
func (sh *RealSnapshotHandler) checkEpoch(lowerBound, upperBound, given uint64, includeLower bool, includeUpper bool) error {
	lowerErr := sh.checkEpochLowerBound(lowerBound, given, includeLower)
	upperErr := sh.checkEpochUpperBound(upperBound, given, includeUpper)
	if lowerErr != nil && upperErr != nil {
		return &InvalidEpochError{
			StrictLowerBound: lowerErr.StrictLowerBound,
			StrictUpperBound: upperErr.StrictUpperBound,
		}
	}
	if lowerErr != nil {
		return lowerErr
	}
	if upperErr != nil {
		return upperErr
	}
	return nil
}

// checkEpochLowerBound checks if the given epoch satisfies the lower bound
// If it does not then error is returned
func (sh *RealSnapshotHandler) checkEpochLowerBound(bound, given uint64, includeEnd bool) *InvalidEpochError {
	if (includeEnd && given < bound) || (!includeEnd && given <= bound) {
		return &InvalidEpochError{StrictLowerBound: bound}
	}
	return nil
}

// checkEpochUpperBound checks if the given epoch satisfies the upper bound.
// If it does not error is returned
func (sh *RealSnapshotHandler) checkEpochUpperBound(bound, given uint64, includeEnd bool) *InvalidEpochError {
	if (includeEnd && given > bound) || (!includeEnd && given >= bound) {
		return &InvalidEpochError{StrictUpperBound: bound}
	}
	return nil
}

// checkOperationalStatus checks if the component is operational. The component is not operational
// when it is not started or it is frozen/destroyed. In those cases appropriate error is returned
func (sh *RealSnapshotHandler) checkOperationalStatus(state *snapshotHandlerState) error {
	if err := sh.checkWithoutFrozenness(state); err != nil {
		return err
	}
	if sh.isFrozen(state) {
		return errSnapshotHandlerFrozen
	}
	return nil
}

// checkWithoutFrozenness is similar to checkOperationalStatus but a bit more relaxed. That is,
// it deos not return an error if the component is non-operational because of freezing
func (sh *RealSnapshotHandler) checkWithoutFrozenness(state *snapshotHandlerState) error {
	if state.isDestroyed {
		return errSnapshotHandlerDestroyed
	}
	if !state.isStarted {
		return errSnapshotHandlerNotStarted
	}
	return nil
}

// isFrozen returns true if it is the freeze level is greater than zero indicating
// that there are some freeze calls without matching unfreeze calls.
func (sh *RealSnapshotHandler) isFrozen(state *snapshotHandlerState) bool {
	return state.freezeLevel > 0
}
