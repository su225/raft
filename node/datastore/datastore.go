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

package datastore

import (
	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/replication"
	"github.com/su225/raft/node/state"
)

const dataStore = "DS"

// DataStore defines the operations that must
// be supported by the key-value store.
type DataStore interface {
	PutData(key string, value string) (putErr error)
	GetData(key string) (value string, retrievalErr error)
	DeleteData(key string) (delErr error)
}

// RaftKeyValueStore is the implementation of DataStore.
// This implements the distributed and strongly consistent
// key-value store built on top of Raft consensus protocol
type RaftKeyValueStore struct {
	replication.EntryReplicationController
	state.RaftStateManager
	log.WriteAheadLogManager
	log.SnapshotHandler
}

// NewRaftKeyValueStore creates a new instance of Raft
// key-value store and returns the same
func NewRaftKeyValueStore(
	replicationCtrl replication.EntryReplicationController,
	raftStateManager state.RaftStateManager,
	writeAheadLogMgr log.WriteAheadLogManager,
	snapshotHandler log.SnapshotHandler,
) *RaftKeyValueStore {
	return &RaftKeyValueStore{
		EntryReplicationController: replicationCtrl,
		RaftStateManager:           raftStateManager,
		WriteAheadLogManager:       writeAheadLogMgr,
		SnapshotHandler:            snapshotHandler,
	}
}

// PutData adds the given key-value pair if it does not exist
// or updates the value if the key already exists. If there is
// some error in the process then it is returned
func (ds *RaftKeyValueStore) PutData(key string, value string) error {
	return ds.appendEntryAndReplicate(&log.UpsertEntry{
		TermID: ds.RaftStateManager.GetCurrentTermID(),
		Key:    key,
		Value:  value,
	})
}

// GetData returns the data for the given key-value pair if it
// exists or an error otherwise.
func (ds *RaftKeyValueStore) GetData(key string) (string, error) {
	ds.SnapshotHandler.Freeze()
	defer ds.SnapshotHandler.Unfreeze()

	curSnapshotMetadata := ds.SnapshotHandler.GetSnapshotMetadata()
	curEpoch := curSnapshotMetadata.Epoch
	curSnapshotIndex := curSnapshotMetadata.Index

	kvStore := make(map[string]string)
	value, kvErr := ds.SnapshotHandler.GetKeyValuePair(curEpoch, key)
	logStartIndex := uint64(0)

	if kvErr == nil || log.IsKeyValuePairDoesNotExistInSnapshotError(kvErr) {
		logStartIndex = curSnapshotIndex + 1
	}
	if kvErr == nil {
		kvStore[key] = value
	}
	metadata, _ := ds.WriteAheadLogManager.GetMetadata()
	for i := logStartIndex; i <= metadata.MaxCommittedIndex; i++ {
		entry, retrieveErr := ds.WriteAheadLogManager.GetEntry(i)
		if retrieveErr != nil {
			return "", retrieveErr
		}
		switch e := entry.(type) {
		case *log.UpsertEntry:
			kvStore[e.Key] = e.Value
		case *log.DeleteEntry:
			delete(kvStore, e.Key)
		}
	}
	if _, present := kvStore[key]; !present {
		return "", &KeyNotFoundError{Key: key}
	}
	return kvStore[key], nil
}

// DeleteData deletes the key-value pair with the given key if
// it exists. If it doesn't then it is a no-op and still not an
// error. If there is an error during the operation like failure
// to replicate it to a majority of nodes in the cluster then
// it is returned.
func (ds *RaftKeyValueStore) DeleteData(key string) error {
	return ds.appendEntryAndReplicate(&log.DeleteEntry{
		TermID: ds.RaftStateManager.GetCurrentTermID(),
		Key:    key,
	})
}

// appendEntryAndReplicate appends the entry to the log and tries to replicate
// in the log of other nodes in the cluster
func (ds *RaftKeyValueStore) appendEntryAndReplicate(entry log.Entry) error {
	tailEntryID, appendErr := ds.WriteAheadLogManager.AppendEntry(entry)
	if appendErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: appendErr.Error(),
			logfield.Component:   dataStore,
			logfield.Event:       "APPEND-ENTRY",
		}).Errorf("error while appending entry")
		return appendErr
	}
	return ds.EntryReplicationController.ReplicateEntry(tailEntryID, entry)
}
