package log

import (
	"errors"
)

var errWriteAheadLog = errors.New("write-ahead log error")

// MockWriteAheadLogManager is the mock implementation of
// WriteAheadLogManager for TESTING PURPOSES ONLY. It starts
// with the given metadata and methods which mutate state are
// no-op and only their counts are maintained
type MockWriteAheadLogManager struct {
	ShouldSucceed                bool
	UpdateMaxCommittedIndexCount uint64
	AppendEntryCount             uint64
	WriteEntryCount              uint64
	WriteEntryAfterCount         uint64

	Entries map[uint64]Entry
	WriteAheadLogMetadata
}

// NewMockWriteAheadLogManager creates a new instance of mock write-ahead
// log manager with the given initial state. If the
func NewMockWriteAheadLogManager(succeed bool, initState WriteAheadLogMetadata, entries map[uint64]Entry) *MockWriteAheadLogManager {
	return &MockWriteAheadLogManager{
		ShouldSucceed:         succeed,
		WriteAheadLogMetadata: initState,
		Entries:               entries,
	}
}

// GetDefaultMockWriteAheadLogManager returns the mock write-ahead log manager with default settings
// This is ONLY FOR TESTING PURPOSES
func GetDefaultMockWriteAheadLogManager(succeeds bool) *MockWriteAheadLogManager {
	return NewMockWriteAheadLogManager(succeeds,
		WriteAheadLogMetadata{
			TailEntryID: EntryID{
				TermID: 2,
				Index:  3,
			},
			MaxCommittedIndex: 2,
		},
		map[uint64]Entry{
			0: &SentinelEntry{},
			1: &UpsertEntry{TermID: 1, Key: "a", Value: "1"},
			2: &UpsertEntry{TermID: 2, Key: "b", Value: "2"},
			3: &DeleteEntry{TermID: 2, Key: "a"},
		},
	)
}

// UpdateMaxCommittedIndex here is a no-op and just increments
// the updateMaxCommittedIndexCount by 1
func (w *MockWriteAheadLogManager) UpdateMaxCommittedIndex(index uint64) (uint64, error) {
	if !w.ShouldSucceed {
		return 0, errWriteAheadLog
	}
	w.UpdateMaxCommittedIndexCount++
	return w.WriteAheadLogMetadata.MaxCommittedIndex, nil
}

// AppendEntry is a no-op and increments AppendEntryCount by 1
func (w *MockWriteAheadLogManager) AppendEntry(entry Entry) (EntryID, error) {
	if !w.ShouldSucceed {
		return EntryID{}, errWriteAheadLog
	}
	w.AppendEntryCount++
	return w.WriteAheadLogMetadata.TailEntryID, nil
}

// WriteEntry is a no-op and increments WriteEntryCount by 1
func (w *MockWriteAheadLogManager) WriteEntry(index uint64, entry Entry) (EntryID, error) {
	if !w.ShouldSucceed {
		return EntryID{}, errWriteAheadLog
	}
	w.WriteEntryCount++
	return w.WriteAheadLogMetadata.TailEntryID, nil
}

// WriteEntryAfter is a no-op and increments WriteEntryAfterCount by 1
func (w *MockWriteAheadLogManager) WriteEntryAfter(beforeEntryID EntryID, curIndex uint64, entry Entry) (EntryID, error) {
	if !w.ShouldSucceed {
		return EntryID{}, errWriteAheadLog
	}
	w.WriteEntryAfterCount++
	return w.WriteAheadLogMetadata.TailEntryID, nil
}

// GetEntry returns the entry if it exists or generic error
func (w *MockWriteAheadLogManager) GetEntry(index uint64) (Entry, error) {
	if !w.ShouldSucceed {
		return nil, errWriteAheadLog
	}
	if entry, present := w.Entries[index]; present {
		return entry, nil
	}
	return nil, errWriteAheadLog
}

// GetMetadata returns the metadata
func (w *MockWriteAheadLogManager) GetMetadata() (WriteAheadLogMetadata, error) {
	return w.WriteAheadLogMetadata, nil
}

// Start is a no-op
func (w *MockWriteAheadLogManager) Start() error {
	return nil
}

// Destroy is a no-op
func (w *MockWriteAheadLogManager) Destroy() error {
	return nil
}

// Recover is a no-op
func (w *MockWriteAheadLogManager) Recover() error {
	return nil
}
