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
	entry, isPresent := ep.Entries[index]
	if !isPresent {
		return nil, errPersistence
	}
	return entry, nil
}

// DeleteEntry deletes the given entry if ShouldSucceed is true, error otherwise
func (ep *InMemoryEntryPersistence) DeleteEntry(index uint64) error {
	if !ep.ShouldSucceed {
		return errPersistence
	}
	if _, present := ep.Entries[index]; present {
		delete(ep.Entries, index)
	}
	return nil
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
