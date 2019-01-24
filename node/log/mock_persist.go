package log

import (
	"errors"
)

var errPersistence = errors.New("persistence-error")

// InMemoryEntryPersistence keeps all data in-memory and
// must be used only for testing purposes
type InMemoryEntryPersistence struct {
	ShouldSucceed bool
	Entries       map[uint64]Entry
}

// NewInMemoryEntryPersistence creates a new instance of in-memory
// entry persistence.
func NewInMemoryEntryPersistence(succeed bool) *InMemoryEntryPersistence {
	return &InMemoryEntryPersistence{
		ShouldSucceed: succeed,
		Entries:       make(map[uint64]Entry),
	}
}

// PersistEntry adds entry to the in-memory map for the given index
func (ep *InMemoryEntryPersistence) PersistEntry(index uint64, entry Entry) error {
	if !ep.ShouldSucceed {
		return errPersistence
	}
	ep.Entries[index] = entry
	return nil
}

// RetrieveEntry retrieves entry at the given index if it exists or error
func (ep *InMemoryEntryPersistence) RetrieveEntry(index uint64) (Entry, error) {
	if !ep.ShouldSucceed {
		return nil, errPersistence
	}
	if entry, isPresent := ep.Entries[index]; !isPresent {
		return nil, errPersistence
	} else {
		return entry, nil
	}
}

// InMemoryMetadataPersistence keeps the metadata in-memory. This
// must be used only for testing purposes
type InMemoryMetadataPersistence struct {
	ShouldSucceed bool
	WriteAheadLogMetadata
}

// NewInMemoryMetadataPersistence creates a new instance of in-memory metadata persistence
func NewInMemoryMetadataPersistence(succeed bool) *InMemoryMetadataPersistence {
	return &InMemoryMetadataPersistence{
		ShouldSucceed: succeed,
	}
}

// PersistMetadata persists keeps metadata in-memory
func (mp *InMemoryMetadataPersistence) PersistMetadata(metadata *WriteAheadLogMetadata) error {
	if !mp.ShouldSucceed {
		return errPersistence
	}
	mp.WriteAheadLogMetadata.MaxCommittedIndex = metadata.MaxCommittedIndex
	mp.WriteAheadLogMetadata.TailEntryID = metadata.TailEntryID
	return nil
}

// RetrieveMetadata returns the metadata stored
func (mp *InMemoryMetadataPersistence) RetrieveMetadata() (*WriteAheadLogMetadata, error) {
	if !mp.ShouldSucceed {
		return nil, errPersistence
	}
	return &mp.WriteAheadLogMetadata, nil
}
