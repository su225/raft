package log

import (
	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/common"
)

// EntryID represents the identifier for the
// entry in the write-ahead log.
type EntryID struct {
	// TermID is the term in which the
	// entry was created.
	TermID uint64 `json:"term_id"`
	// Index represents the position of
	// the entry in the write-ahead log
	Index uint64 `json:"index"`
}

// WriteAheadLogMetadata contains information about
// the tail entry ID of the write-ahead log and also
// the commit index of the log
type WriteAheadLogMetadata struct {
	// TailEntryID represents the EntryID of the last
	// valid entry in the write-ahead log
	TailEntryID EntryID `json:"tail_entry_id"`
	// MaxCommittedIndex represents the maximum index
	// that has been committed - that is, the entries
	// upto the index cannot be changed.
	MaxCommittedIndex uint64 `json:"max_committed_index"`
}

const writeAheadLog = "WRITE-AHEAD-LOG"

var writeAheadLogManagerNotStartedError = &common.ComponentHasNotStartedError{ComponentName: writeAheadLog}
var writeAheadLogManagerIsDestroyedError = &common.ComponentIsDestroyedError{ComponentName: writeAheadLog}

// WriteAheadLogManager represents the component which
// is responsible for managing the write-ahead log - storing
// and retrieving metadata and entry to/from durable storage,
// maintaining transactionality when needed etc
type WriteAheadLogManager interface {
	// UpdateMaxComimittedIndex tries to update the maximum committed index
	// in the log. If the given index is greater than the current one then
	// it is updated and persisted. If persistence fails, then update is
	// rolled back. In other words, the state on disk and memory is kept
	// consistent and transactionality must be maintained. This operation
	// must be idempotent.
	UpdateMaxCommittedIndex(index uint64) (updatedIndex uint64, err error)

	// AppendEntry appends the given entry to the tail of the log. The entry
	// must be persisted to durable storage. If persistence fails then this
	// operation is considered to be failed and the tailLogEntryID must not
	// be updated. This operation is not idempotent, but transactional
	AppendEntry(entry Entry) (tailLogEntryID EntryID, err error)

	// WriteEntry writes the given entry at the given index subject to the
	// condition that the index is not committed and that it is not more than
	// (index of last log entry + 1). If persistence fails then it is an error
	// This operation is transactional as well as idempotent.
	WriteEntry(index uint64, entry Entry) (tailLogEntryID EntryID, err error)

	// GetEntry returns the entry at the given index if it is in the range from
	// 0 to the index of the tail entry. If there is an error during the process
	// either because the index is invalid or the persistence failure then it
	// is returned. This function should not change any entry.
	GetEntry(index uint64) (entry Entry, err error)

	// GetMaxCommittedIndex returns the maximum committed index at the
	// time of calling. In case of an error it is returned, in which case
	// the index returned must not be considered
	GetMetadata() (metadata WriteAheadLogMetadata, err error)

	// ComponentLifecycle must be implemented so that starting and destroying
	// the component can be done gracefully. For instance, if there is a write
	// going on when destruction command is issued then it can wait till the
	// completion/rollback of write operation before destroying.
	common.ComponentLifecycle

	// Recoverable must be implemented so that the metadata of the write-ahead
	// log like tailLogEntryID, MaxCommittedIndex can be recovered.
	common.Recoverable
}

// WriteAheadLogManagerImpl is the actual implementation of WriteAheadLogManager.
// Persistence is pluggable - it can be a file or something like RocksDB, LevelDB
// (or B*-Tree file, Log Sort Merge Tree and so on)
type WriteAheadLogManagerImpl struct {
	EntryPersistence
	MetadataPersistence
	commandChannel chan writeAheadLogManagerCommand
}

// NewWriteAheadLogManagerImpl creates a new instance of write-ahead log manager
// with given entry and log persistence mechanisms. The component is not yet
// operational - that is, operations invoked will fail
func NewWriteAheadLogManagerImpl(
	entryPersistence EntryPersistence,
	metadataPersistence MetadataPersistence,
) *WriteAheadLogManagerImpl {
	return &WriteAheadLogManagerImpl{
		EntryPersistence:    entryPersistence,
		MetadataPersistence: metadataPersistence,
		commandChannel:      make(chan writeAheadLogManagerCommand),
	}
}

// Start starts component operations if it is not already destroyed. If the component
// is already started then this is a no-op provided it is not destroyed. At the start,
// it recovers the metadata of the write-ahead log from the disk.
func (wal *WriteAheadLogManagerImpl) Start() error {
	go wal.loop()
	if recoveryErr := wal.Recover(); recoveryErr != nil {
		return recoveryErr
	}
	errorChannel := make(chan error)
	wal.commandChannel <- &startWriteAheadLogManager{errChan: errorChannel}
	return <-errorChannel
}

// Destroy makes the component non-operational. In other words, no operation can be
// invoked on this component after its destruction.
func (wal *WriteAheadLogManagerImpl) Destroy() error {
	errorChannel := make(chan error)
	wal.commandChannel <- &destroyWriteAheadLogManager{errChan: errorChannel}
	return <-errorChannel
}

// Recover recovers the metadata of the write-ahead log before the crash/start. If there
// is an issue while recovering then it is returned. It should not proceed further
func (wal *WriteAheadLogManagerImpl) Recover() error {
	errorChannel := make(chan error)
	wal.commandChannel <- &recoverWriteAheadLogMetadata{errChan: errorChannel}
	return <-errorChannel
}

// UpdateMaxCommittedIndex updates the maximum committed index if possible and
// returns the updated index. It also persists this information to durable storage
// as this is an important part of metadata
func (wal *WriteAheadLogManagerImpl) UpdateMaxCommittedIndex(index uint64) (uint64, error) {
	replyChannel := make(chan *updateMaxCommittedIndexReply)
	wal.commandChannel <- &updateMaxCommittedIndex{
		index:     index,
		replyChan: replyChannel,
	}
	reply := <-replyChannel
	return reply.updatedIndex, reply.updateErr
}

// GetMetadata returns the metadata of the write-ahead log manager
func (wal *WriteAheadLogManagerImpl) GetMetadata() (WriteAheadLogMetadata, error) {
	replyChannel := make(chan *getMetadataReply)
	wal.commandChannel <- &getMetadata{replyChan: replyChannel}
	reply := <-replyChannel
	return reply.metadata, reply.retrievalErr
}

// AppendEntry appends the entry to the log, persists the entry and metadata after the
// update and returns the updated tail entry ID. In case there is an error during the
// operation, then it is returned and metadata is untouched. This is not idempotent
func (wal *WriteAheadLogManagerImpl) AppendEntry(entry Entry) (EntryID, error) {
	replyChannel := make(chan *writeEntryReply)
	wal.commandChannel <- &appendEntry{
		entry:     entry,
		replyChan: replyChannel,
	}
	reply := <-replyChannel
	return reply.tailLogEntryID, reply.appendErr
}

// WriteEntry writes the entry to the given index in the log provided the index is not committed
// and at max, just one more than the tail index. It persists the entry and metadata to disk and
// returns the updated tail entry ID. If there is any error during operation, then metadata is
// not updated and the error is returned.
func (wal *WriteAheadLogManagerImpl) WriteEntry(index uint64, entry Entry) (EntryID, error) {
	replyChannel := make(chan *writeEntryReply)
	wal.commandChannel <- &writeEntry{
		entry:     entry,
		index:     index,
		replyChan: replyChannel,
	}
	reply := <-replyChannel
	return reply.tailLogEntryID, reply.appendErr
}

// GetEntry returns the entry at the corresponding index if it is valid. If the entry is retrieved
// successfully then it is returned. In case of trouble error is returned with nil entry.
func (wal *WriteAheadLogManagerImpl) GetEntry(index uint64) (Entry, error) {
	replyChannel := make(chan *getEntryReply)
	wal.commandChannel <- &getEntry{
		index:     index,
		replyChan: replyChannel,
	}
	reply := <-replyChannel
	return reply.entry, reply.retrievalErr
}

// writeAheadLogManagerState represents the state of the
// write-ahead log manager
type writeAheadLogManagerState struct {
	isStarted, isDestroyed bool
	WriteAheadLogMetadata
}

// loop accepts commands and handles them appropriately. It is the command server. It also maintains
// the state of the Write-ahead log manager locally. This is designed to be safe in concurrent environment
func (wal *WriteAheadLogManagerImpl) loop() {
	state := &writeAheadLogManagerState{
		isStarted:   false,
		isDestroyed: false,
	}
	for {
		cmd := <-wal.commandChannel
		switch c := cmd.(type) {
		case *startWriteAheadLogManager:
			c.errChan <- wal.handleStartWriteAheadLogManager(state, c)
		case *destroyWriteAheadLogManager:
			c.errChan <- wal.handleDestroyWriteAheadLogManager(state, c)
		case *recoverWriteAheadLogMetadata:
			c.errChan <- wal.handleRecoverWriteAheadLogManager(state, c)
		case *updateMaxCommittedIndex:
			c.replyChan <- wal.handleUpdateMaxCommittedIndex(state, c)
		case *getMetadata:
			c.replyChan <- wal.handleGetMetadata(state, c)
		case *appendEntry:
			c.replyChan <- wal.handleAppendEntry(state, c)
		case *writeEntry:
			c.replyChan <- wal.handleWriteEntry(state, c)
		case *getEntry:
			c.replyChan <- wal.handleGetEntry(state, c)
		}
	}
}

func (wal *WriteAheadLogManagerImpl) handleStartWriteAheadLogManager(state *writeAheadLogManagerState, cmd *startWriteAheadLogManager) error {
	if state.isDestroyed {
		return writeAheadLogManagerIsDestroyedError
	}
	if state.isStarted {
		return nil
	}
	state.isStarted = true
	logrus.WithFields(logrus.Fields{
		logfield.Component: writeAheadLog,
		logfield.Event:     "DESTROY",
	}).Infoln("started write-ahead log manager")
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleDestroyWriteAheadLogManager(state *writeAheadLogManagerState, cmd *destroyWriteAheadLogManager) error {
	if state.isDestroyed {
		return nil
	}
	state.isDestroyed = true
	logrus.WithFields(logrus.Fields{
		logfield.Component: writeAheadLog,
		logfield.Event:     "DESTROY",
	}).Infoln("destroyed write-ahead log manager")
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleRecoverWriteAheadLogManager(state *writeAheadLogManagerState, cmd *recoverWriteAheadLogMetadata) error {
	metadata, retrievalErr := wal.RetrieveMetadata()
	if retrievalErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: retrievalErr.Error(),
			logfield.Component:   writeAheadLog,
			logfield.Event:       "RECOVERY",
		}).Errorln("error while recovering state. Setting to zeros and moving on..")

		// Don't crash just because recovery is not successful.
		// Reset everything to default state and move on
		state.WriteAheadLogMetadata = WriteAheadLogMetadata{
			TailEntryID: EntryID{
				TermID: 0,
				Index:  0,
			},
			MaxCommittedIndex: 0,
		}
		return nil
	}
	state.WriteAheadLogMetadata = *metadata
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleUpdateMaxCommittedIndex(state *writeAheadLogManagerState, cmd *updateMaxCommittedIndex) *updateMaxCommittedIndexReply {
	beforeMaxCommittedIndex := state.WriteAheadLogMetadata.MaxCommittedIndex
	if statusErr := wal.checkOperationalStatus(state); statusErr != nil {
		return &updateMaxCommittedIndexReply{
			updatedIndex: beforeMaxCommittedIndex,
			updateErr:    statusErr,
		}
	}
	if beforeMaxCommittedIndex < cmd.index {
		tailIndex := state.WriteAheadLogMetadata.TailEntryID.Index
		if cmd.index > tailIndex {
			return &updateMaxCommittedIndexReply{
				updatedIndex: beforeMaxCommittedIndex,
				updateErr: &CommittedIndexTooFarAheadError{
					UpperLimit:       tailIndex,
					GivenCommitIndex: cmd.index,
				},
			}
		}
		state.WriteAheadLogMetadata.MaxCommittedIndex = cmd.index
		if persistErr := wal.MetadataPersistence.PersistMetadata(&state.WriteAheadLogMetadata); persistErr != nil {
			logrus.WithFields(logrus.Fields{
				logfield.ErrorReason: persistErr.Error(),
				logfield.Component:   writeAheadLog,
				logfield.Event:       "UPDATE-COMMIT-IDX",
			}).Errorln("error while persisting metadata")
			state.WriteAheadLogMetadata.MaxCommittedIndex = beforeMaxCommittedIndex
			return &updateMaxCommittedIndexReply{
				updatedIndex: beforeMaxCommittedIndex,
				updateErr:    persistErr,
			}
		}
	}
	return &updateMaxCommittedIndexReply{updatedIndex: state.WriteAheadLogMetadata.MaxCommittedIndex}
}

func (wal *WriteAheadLogManagerImpl) handleGetMetadata(state *writeAheadLogManagerState, cmd *getMetadata) *getMetadataReply {
	statusErr := wal.checkOperationalStatus(state)
	return &getMetadataReply{
		metadata:     state.WriteAheadLogMetadata,
		retrievalErr: statusErr,
	}
}

func (wal *WriteAheadLogManagerImpl) handleAppendEntry(state *writeAheadLogManagerState, cmd *appendEntry) *writeEntryReply {
	if statusErr := wal.checkOperationalStatus(state); statusErr != nil {
		return &writeEntryReply{
			tailLogEntryID: state.WriteAheadLogMetadata.TailEntryID,
			appendErr:      statusErr,
		}
	}
	return wal.handleWriteEntry(state, &writeEntry{
		index: state.TailEntryID.Index + 1,
		entry: cmd.entry,
	})
}

func (wal *WriteAheadLogManagerImpl) handleWriteEntry(state *writeAheadLogManagerState, cmd *writeEntry) *writeEntryReply {
	beforeMetadata := state.WriteAheadLogMetadata
	if statusErr := wal.checkOperationalStatus(state); statusErr != nil {
		return &writeEntryReply{
			tailLogEntryID: beforeMetadata.TailEntryID,
			appendErr:      statusErr,
		}
	}
	currentEntryIndex := cmd.index
	if currentEntryIndex <= state.MaxCommittedIndex || currentEntryIndex > state.TailEntryID.Index+1 {
		return &writeEntryReply{
			tailLogEntryID: beforeMetadata.TailEntryID,
			appendErr: &CannotWriteEntryError{
				IndexAttempted: cmd.index,
				CommittedIndex: beforeMetadata.MaxCommittedIndex,
				TailIndex:      beforeMetadata.TailEntryID.Index,
			},
		}
	}
	if persistErr := wal.PersistEntry(currentEntryIndex, cmd.entry); persistErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: persistErr.Error(),
			logfield.Component:   writeAheadLog,
			logfield.Event:       "APPEND-ENTRY",
		}).Errorln("error while persisting entry")
		state.WriteAheadLogMetadata = beforeMetadata
		return &writeEntryReply{
			tailLogEntryID: beforeMetadata.TailEntryID,
			appendErr:      persistErr,
		}
	}
	nextTailEntryID := EntryID{
		TermID: cmd.entry.GetTermID(),
		Index:  currentEntryIndex,
	}
	state.WriteAheadLogMetadata.TailEntryID = nextTailEntryID
	if metaPersistErr := wal.PersistMetadata(&state.WriteAheadLogMetadata); metaPersistErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: metaPersistErr.Error(),
			logfield.Component:   writeAheadLog,
			logfield.Event:       "APPEND-ENTRY",
		}).Errorln("error while persisting metadata")
		state.WriteAheadLogMetadata = beforeMetadata
		return &writeEntryReply{
			tailLogEntryID: beforeMetadata.TailEntryID,
			appendErr:      metaPersistErr,
		}
	}
	return &writeEntryReply{
		tailLogEntryID: state.WriteAheadLogMetadata.TailEntryID,
		appendErr:      nil,
	}
}

func (wal *WriteAheadLogManagerImpl) handleGetEntry(state *writeAheadLogManagerState, cmd *getEntry) *getEntryReply {
	if statusErr := wal.checkOperationalStatus(state); statusErr != nil {
		return &getEntryReply{
			entry:        nil,
			retrievalErr: statusErr,
		}
	}
	entry, retrievalErr := wal.EntryPersistence.RetrieveEntry(cmd.index)
	if retrievalErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: retrievalErr.Error(),
			logfield.Component:   writeAheadLog,
			logfield.Event:       "GET-ENTRY",
		}).Errorf("failed to retrieve entry at index %d", cmd.index)
	}
	return &getEntryReply{
		entry:        entry,
		retrievalErr: retrievalErr,
	}
}

func (wal *WriteAheadLogManagerImpl) checkOperationalStatus(state *writeAheadLogManagerState) error {
	if state.isDestroyed {
		return writeAheadLogManagerIsDestroyedError
	}
	if !state.isStarted {
		return writeAheadLogManagerNotStartedError
	}
	return nil
}
