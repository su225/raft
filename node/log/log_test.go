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
	"fmt"
	"reflect"
	"testing"
)

func getDefaultWriteAheadLogState() *writeAheadLogManagerState {
	return &writeAheadLogManagerState{
		isStarted:   true,
		isDestroyed: false,
		WriteAheadLogMetadata: WriteAheadLogMetadata{
			TailEntryID: EntryID{
				TermID: 10,
				Index:  102,
			},
			MaxCommittedIndex: 100,
		},
	}
}

func getWriteAheadLogManagerWithMockedPersistence(entrySucceeds, metaSucceeds bool) *WriteAheadLogManagerImpl {
	return NewWriteAheadLogManagerImpl(
		NewInMemoryEntryPersistence(entrySucceeds),
		NewInMemoryMetadataPersistence(metaSucceeds),
		GetDefaultMockSnapshotHandler(),
	)
}

func getEntryPersistenceWithDefaultEntries(succeeds bool, max uint64) *InMemoryEntryPersistence {
	entryPersistence := NewInMemoryEntryPersistence(succeeds)
	entryPersistence.Entries[0] = &SentinelEntry{}
	for i := uint64(1); i <= max; i++ {
		entryPersistence.Entries[i] = &UpsertEntry{
			TermID: 10,
			Key:    fmt.Sprintf("key-%d", i),
			Value:  fmt.Sprintf("value-%d", i),
		}
	}
	return entryPersistence
}

func TestUpdateMaxCommittedIndexIsMonotonic(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	cmd1 := &updateMaxCommittedIndex{index: 99}
	cmd2 := &updateMaxCommittedIndex{index: 102}
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	reply1 := waLogManager.handleUpdateMaxCommittedIndex(waLogState, cmd1)
	if reply1 == nil || reply1.updateErr != nil || waLogState.MaxCommittedIndex != 100 {
		t.FailNow()
	}
	reply2 := waLogManager.handleUpdateMaxCommittedIndex(waLogState, cmd2)
	if reply2 == nil || reply2.updateErr != nil || waLogState.MaxCommittedIndex != 102 {
		t.FailNow()
	}
}

func TestUpdateMaxCommittedIndexFailsIfGivenIndexIsGreaterThanTail(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	cmd := &updateMaxCommittedIndex{index: 110}
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	reply := waLogManager.handleUpdateMaxCommittedIndex(waLogState, cmd)
	if reply == nil || reply.updateErr == nil || waLogState.MaxCommittedIndex != 100 {
		t.FailNow()
	}
}

func TestGetMaxCommittedIndexReturnsTheMaximumCommittedIndex(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	expectedWALogState := *waLogState
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	cmd := &getMetadata{}
	reply := waLogManager.handleGetMetadata(waLogState, cmd)
	if reply == nil || reply.retrievalErr != nil {
		t.FailNow()
	}
	if reflect.DeepEqual(expectedWALogState, reply.metadata) {
		t.FailNow()
	}
}

func TestAppendEntryUpdatesTailEntryIDOnSuccess(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	cmd := &appendEntry{entry: &DeleteEntry{TermID: 10, Key: "test"}}
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	reply := waLogManager.handleAppendEntry(waLogState, cmd)
	if reply == nil || reply.appendErr != nil {
		t.FailNow()
	}
	if reply.tailLogEntryID.TermID != 10 || reply.tailLogEntryID.Index != 103 {
		t.FailNow()
	}
}

func TestAppendEntryDoesNotUpdateTailEntryIfPersistenceFails(t *testing.T) {
	cmd := &appendEntry{entry: &DeleteEntry{TermID: 10, Key: "test"}}
	for _, metaPersistSucceeds := range []bool{true, false} {
		for _, entryPersistSucceeds := range []bool{true, false} {
			if metaPersistSucceeds && entryPersistSucceeds {
				continue
			}
			waLogState := getDefaultWriteAheadLogState()
			previousTailID := waLogState.TailEntryID
			metadataSucceeds, entrySucceeds := metaPersistSucceeds, entryPersistSucceeds
			subtestName := fmt.Sprintf("Metadata=%v,Entry=%v", metaPersistSucceeds, entryPersistSucceeds)
			t.Run(subtestName, func(t *testing.T) {
				waLogManager := getWriteAheadLogManagerWithMockedPersistence(entrySucceeds, metadataSucceeds)
				reply := waLogManager.handleAppendEntry(waLogState, cmd)
				if reply == nil || reply.appendErr == nil || waLogState.TailEntryID != previousTailID {
					t.FailNow()
				}
			})
		}
	}
}

func TestWriteEntryUpdatesTailEntryIDOnSuccess(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	cmd := &writeEntry{
		index: 101,
		entry: &DeleteEntry{TermID: 10, Key: "test"},
	}
	reply := waLogManager.handleWriteEntry(waLogState, cmd)
	if reply == nil || reply.appendErr != nil {
		t.FailNow()
	}
	if waLogState.TailEntryID.TermID != cmd.entry.GetTermID() || waLogState.TailEntryID.Index != 101 {
		t.FailNow()
	}
}

func TestWriteEntryFailsIfIndexIsLessThanOrEqualToMaxCommittedIndex(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	cmd := &writeEntry{
		index: waLogState.WriteAheadLogMetadata.MaxCommittedIndex - 1,
		entry: &DeleteEntry{TermID: 10, Key: "test"},
	}
	beforeTailEntryID := waLogState.WriteAheadLogMetadata.TailEntryID
	reply := waLogManager.handleWriteEntry(waLogState, cmd)
	if reply == nil || reply.appendErr == nil {
		t.FailNow()
	}
	if beforeTailEntryID != waLogState.WriteAheadLogMetadata.TailEntryID {
		t.FailNow()
	}
}

func TestWriteEntryFailsIfIndexIsBeyondTailIndexPlusOne(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	cmd := &writeEntry{
		index: waLogState.WriteAheadLogMetadata.TailEntryID.Index + 10,
		entry: &UpsertEntry{TermID: 10, Key: "test", Value: "unit"},
	}
	beforeTailEntryID := waLogState.WriteAheadLogMetadata.TailEntryID
	reply := waLogManager.handleWriteEntry(waLogState, cmd)
	if reply == nil || reply.appendErr == nil {
		t.FailNow()
	}
	if beforeTailEntryID != waLogState.WriteAheadLogMetadata.TailEntryID {
		t.FailNow()
	}
}

func TestWriteEntryFailsIfEntryOrMetadataPersistenceFails(t *testing.T) {
	for _, metaResult := range []bool{true, false} {
		for _, entryResult := range []bool{true, false} {
			if metaResult && entryResult {
				continue
			}
			testName := fmt.Sprintf("Metadata=%v,Entry=%v", metaResult, entryResult)
			waLogState := getDefaultWriteAheadLogState()
			cmd := &writeEntry{
				index: waLogState.WriteAheadLogMetadata.TailEntryID.Index + 1,
				entry: &UpsertEntry{TermID: 10, Key: "test", Value: "unit"},
			}
			beforeTailEntryID := waLogState.WriteAheadLogMetadata.TailEntryID
			metadataSucceeds, entrySucceeds := metaResult, entryResult
			t.Run(testName, func(t *testing.T) {
				waLogManager := getWriteAheadLogManagerWithMockedPersistence(entrySucceeds, metadataSucceeds)
				reply := waLogManager.handleWriteEntry(waLogState, cmd)
				if reply == nil || reply.appendErr == nil {
					t.FailNow()
				}
				if beforeTailEntryID != waLogState.WriteAheadLogMetadata.TailEntryID {
					t.FailNow()
				}
			})
		}
	}
}

func TestWriteEntryFailsIfIndexIsZeroOrTermIDIsZero(t *testing.T) {
	waLogState := &writeAheadLogManagerState{
		isStarted:   true,
		isDestroyed: false,
		WriteAheadLogMetadata: WriteAheadLogMetadata{
			MaxCommittedIndex: 0,
			TailEntryID: EntryID{
				TermID: 1,
				Index:  1,
			},
		},
	}
	waLogManager := getWriteAheadLogManagerWithMockedPersistence(true, true)
	cmd := &writeEntry{index: 0, entry: &DeleteEntry{TermID: 1, Key: "test"}}
	beforeTailEntryID := waLogState.WriteAheadLogMetadata.TailEntryID
	reply := waLogManager.handleWriteEntry(waLogState, cmd)
	if reply == nil || reply.appendErr == nil {
		t.FailNow()
	}
	if beforeTailEntryID != waLogState.WriteAheadLogMetadata.TailEntryID {
		t.FailNow()
	}
}

func TestGetEntryReturnsEntryCorrectlyGivenValidIndex(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	entryPersistence := getEntryPersistenceWithDefaultEntries(true, waLogState.TailEntryID.Index)
	metadataPersistence := NewInMemoryMetadataPersistence(true)
	waLogManager := NewWriteAheadLogManagerImpl(entryPersistence, metadataPersistence, GetDefaultMockSnapshotHandler())
	cmd := &getEntry{index: 10}
	reply := waLogManager.handleGetEntry(waLogState, cmd)
	if reply == nil || reply.retrievalErr != nil {
		t.FailNow()
	}
	expectedEntry := &UpsertEntry{
		TermID: 10,
		Key:    "key-10",
		Value:  "value-10",
	}
	if !reflect.DeepEqual(expectedEntry, reply.entry) {
		t.FailNow()
	}
}

func TestGetEntryReturnsErrorIfEntryPersistenceFails(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	entryPersistence := getEntryPersistenceWithDefaultEntries(false, waLogState.TailEntryID.Index)
	metadataPersistence := NewInMemoryMetadataPersistence(true)
	waLogManager := NewWriteAheadLogManagerImpl(entryPersistence, metadataPersistence, GetDefaultMockSnapshotHandler())
	cmd := &getEntry{index: 10}
	reply := waLogManager.handleGetEntry(waLogState, cmd)
	if reply == nil || reply.retrievalErr == nil {
		t.FailNow()
	}
}

func TestGetEntryReturnsErrorIfEntryWithGivenIndexIsNotFound(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	entryPersistence := getEntryPersistenceWithDefaultEntries(true, waLogState.TailEntryID.Index)
	metadataPersistence := NewInMemoryMetadataPersistence(true)
	waLogManager := NewWriteAheadLogManagerImpl(entryPersistence, metadataPersistence, GetDefaultMockSnapshotHandler())
	cmd := &getEntry{index: waLogState.TailEntryID.Index + 1000}
	reply := waLogManager.handleGetEntry(waLogState, cmd)
	if reply == nil || reply.retrievalErr == nil {
		t.FailNow()
	}
}

func TestGetEntryAlwaysReturnsSentinelForIndexZero(t *testing.T) {
	waLogState := getDefaultWriteAheadLogState()
	entryPersistence := getEntryPersistenceWithDefaultEntries(true, waLogState.TailEntryID.Index)
	metadataPersistence := NewInMemoryMetadataPersistence(true)
	waLogManager := NewWriteAheadLogManagerImpl(entryPersistence, metadataPersistence, GetDefaultMockSnapshotHandler())
	cmd := &getEntry{index: 0}
	reply := waLogManager.handleGetEntry(waLogState, cmd)
	if reply == nil || reply.retrievalErr != nil || !reflect.DeepEqual(reply.entry, &SentinelEntry{}) {
		t.FailNow()
	}
}
