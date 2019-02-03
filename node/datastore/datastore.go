package datastore

import (
	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/replication"
	"github.com/su225/raft/node/state"
)

// KVPair represent the key-value pair stored
// in the data-store
type KVPair struct {
	Key   string `json:"k"`
	Value string `json:"v"`
}

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
}

// NewRaftKeyValueStore creates a new instance of Raft
// key-value store and returns the same
func NewRaftKeyValueStore(
	replicationCtrl replication.EntryReplicationController,
	raftStateManager state.RaftStateManager,
	writeAheadLogMgr log.WriteAheadLogManager,
) *RaftKeyValueStore {
	return &RaftKeyValueStore{
		EntryReplicationController: replicationCtrl,
		RaftStateManager:           raftStateManager,
		WriteAheadLogManager:       writeAheadLogMgr,
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
	// slow and dumb version of get-data
	// TODO: Implement snapshot feature
	kvStore := make(map[string]string)
	metadata, _ := ds.WriteAheadLogManager.GetMetadata()
	for i := uint64(1); i <= metadata.MaxCommittedIndex; i++ {
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
