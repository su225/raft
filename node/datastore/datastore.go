package datastore

// KVPair represent the key-value pair stored
// in the data-store
type KVPair struct {
	Key   string `json:"k"`
	Value string `json:"v"`
}

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
}

// NewRaftKeyValueStore creates a new instance of Raft
// key-value store and returns the same
func NewRaftKeyValueStore() *RaftKeyValueStore {
	return &RaftKeyValueStore{}
}

// PutData adds the given key-value pair if it does not exist
// or updates the value if the key already exists. If there is
// some error in the process then it is returned
func PutData(key string, value string) error {
	return nil
}

// GetData returns the data for the given key-value pair if it
// exists or an error otherwise.
func GetData(key string) (string, error) {
	return "", nil
}

// DeleteData deletes the key-value pair with the given key if
// it exists. If it doesn't then it is a no-op and still not an
// error. If there is an error during the operation like failure
// to replicate it to a majority of nodes in the cluster then
// it is returned.
func DeleteData(key string) error {
	return nil
}
