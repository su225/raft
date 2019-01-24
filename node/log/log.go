package log

import (
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
	return nil
}

// Destroy makes the component non-operational. In other words, no operation can be
// invoked on this component after its destruction.
func (wal *WriteAheadLogManagerImpl) Destroy() error {
	return nil
}

// Recover recovers the metadata of the write-ahead log before the crash/start. If there
// is an issue while recovering then it is returned. It should not proceed further
func (wal *WriteAheadLogManagerImpl) Recover() error {
	return nil
}

// UpdateMaxCommittedIndex updates the maximum committed index if possible and
// returns the updated index. It also persists this information to durable storage
// as this is an important part of metadata
func (wal *WriteAheadLogManagerImpl) UpdateMaxCommittedIndex(index uint64) (uint64, error) {
	return 0, nil
}

// AppendEntry appends the entry to the log, persists the entry and metadata after the
// update and returns the updated tail entry ID. In case there is an error during the
// operation, then it is returned and metadata is untouched. This is not idempotent
func (wal *WriteAheadLogManagerImpl) AppendEntry(entry Entry) (EntryID, error) {
	return EntryID{}, nil
}

// WriteEntry writes the entry to the given index in the log provided the index is not committed
// and at max, just one more than the tail index. It persists the entry and metadata to disk and
// returns the updated tail entry ID. If there is any error during operation, then metadata is
// not updated and the error is returned.
func (wal *WriteAheadLogManagerImpl) WriteEntry(index uint64, entry Entry) (EntryID, error) {
	return EntryID{}, nil
}

// GetEntry returns the entry at the corresponding index if it is valid. If the entry is retrieved
// successfully then it is returned. In case of trouble error is returned with nil entry.
func (wal *WriteAheadLogManagerImpl) GetEntry(index uint64) (Entry, error) {
	return nil, nil
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
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleDestroyWriteAheadLogManager(state *writeAheadLogManagerState, cmd *destroyWriteAheadLogManager) error {
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleRecoverWriteAheadLogManager(state *writeAheadLogManagerState, cmd *recoverWriteAheadLogMetadata) error {
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleUpdateMaxCommittedIndex(state *writeAheadLogManagerState, cmd *updateMaxCommittedIndex) *updateMaxCommittedIndexReply {
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleAppendEntry(state *writeAheadLogManagerState, cmd *appendEntry) *writeEntryReply {
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleWriteEntry(state *writeAheadLogManagerState, cmd *writeEntry) *writeEntryReply {
	return nil
}

func (wal *WriteAheadLogManagerImpl) handleGetEntry(state *writeAheadLogManagerState, cmd *getEntry) *getEntryReply {
	return nil
}
