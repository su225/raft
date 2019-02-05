package snapshot

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/su225/raft/node/datastore"
)

// SnapshotPersistence is responsible for persistence
// operations of the snapshot - like storing/deleting key-value
// pairs, creating new snapshot epoch and so on
type SnapshotPersistence interface {
	// PersistKeyValuePair persists the given key-value pair in the given epoch.
	// If there is any issue during the operation then error is returned
	PersistKeyValuePair(epoch uint64, keyValuePair datastore.KVPair) error

	// RetrieveKeyValuePair retrieves the key-value pair for the given key given
	// the epoch. If there is any retrieval related errors then it is returned
	RetrieveKeyValuePair(epoch uint64, key string) (datastore.KVPair, error)

	// DeleteKeyValuePair deletes the given key value pair if it exists. If there
	// is any error during operation then it is returned
	DeleteKeyValuePair(epoch uint64, key string) error

	// StartEpoch starts a new epoch. If the epoch already exists then
	// error is returned. Note that the monotonicity of epochs is not checked
	// in this place and it should be taken care elsewhere
	StartEpoch(epoch uint64) error

	// DeleteEpoch deletes the given epoch. If there is an error while deleting
	// the given epoch then it is returned
	DeleteEpoch(epoch uint64) error

	// ForEachKey iterates over each key in the given epoch and applies the
	// operation on each of the key-value pair
	ForEachKey(epoch uint64, operation func(kv datastore.KVPair) error) error
}

// SnapshotMetadataPersistence is responsible for persisting
// and retrieving snapshot related metadata like the index upto which
// snapshot is taken and the current epoch for this node
type SnapshotMetadataPersistence interface {
	// PersistMetadata persists the snapshot metadata to the disk
	// Error is returned if there are any issues during operations
	PersistMetadata(metadata *SnapshotMetadata) error

	// RetrieveMetadata retrieves the snapshot metadata from the
	// disk. If there is any error during the process then it is returned
	// along with nil for SnapshotMetadata.
	RetrieveMetadata() (*SnapshotMetadata, error)
}

// SimpleFileBasedSnapshotPersistence uses files to store each key-value pair. One
// file per key-value pair and the name of the file would be SHA256 hash of the key
// (to take care of cases where there will be restricted characters in the key).
type SimpleFileBasedSnapshotPersistence struct {
	SnapshotPath string
}

// NewSimpleFileBasedSnapshotPersistence creates a new instance of simple file based
// snapshot persistence. "snapshotPath" is the path where snapshot data is kept
func NewSimpleFileBasedSnapshotPersistence(snapshotPath string) *SimpleFileBasedSnapshotPersistence {
	return &SimpleFileBasedSnapshotPersistence{SnapshotPath: snapshotPath}
}

// PersistKeyValuePair persists the key-value pair in the given epoch. If there is any error during
// the operation then it is reported
func (sp *SimpleFileBasedSnapshotPersistence) PersistKeyValuePair(epoch uint64, keyValuePair datastore.KVPair) error {
	kvPath := sp.getKVPairFilename(epoch, keyValuePair.Key)
	kvBytes, marshalErr := json.Marshal(keyValuePair)
	if marshalErr != nil {
		return marshalErr
	}
	return ioutil.WriteFile(kvPath, kvBytes, 0600)
}

// RetrieveKeyValuePair retrieves the key-value pair in the given epoch. If there is any error
// during the operation then it is reported.
func (sp *SimpleFileBasedSnapshotPersistence) RetrieveKeyValuePair(epoch uint64, key string) (datastore.KVPair, error) {
	kvPath := sp.getKVPairFilename(epoch, key)
	return sp.getKVPairFromFile(kvPath)
}

// DeleteKeyValuePair deletes the key-value pair from persistence. If there is any error
// during the operation then it is reported.
func (sp *SimpleFileBasedSnapshotPersistence) DeleteKeyValuePair(epoch uint64, key string) error {
	kvPath := sp.getKVPairFilename(epoch, key)
	return os.Remove(kvPath)
}

// StartEpoch starts a new epoch. If there is any error like if the epoch already exists then it is reported
func (sp *SimpleFileBasedSnapshotPersistence) StartEpoch(epoch uint64) error {
	epochPath := sp.getEpochPath(epoch)
	_, err := os.Stat(epochPath)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(epochPath, 0755); err != nil {
			return err
		}
	}
	return err
}

// DeleteEpoch deletes an existing epoch. If the epoch is not present then error is reported
func (sp *SimpleFileBasedSnapshotPersistence) DeleteEpoch(epoch uint64) error {
	epochPath := sp.getEpochPath(epoch)
	_, err := os.Stat(epochPath)
	if os.IsNotExist(err) {
		return err
	}
	return os.RemoveAll(epochPath)
}

// ForEachKey iterates over each key-value pair in the given epoch and performs the given
// "operation" on each of the key-value pair.
func (sp *SimpleFileBasedSnapshotPersistence) ForEachKey(epoch uint64, operation func(kv datastore.KVPair) error) error {
	epochPath := sp.getEpochPath(epoch)
	return filepath.Walk(epochPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		kvPair, readErr := sp.getKVPairFromFile(path)
		if readErr != nil {
			return readErr
		}
		return operation(kvPair)
	})
}

// getKVPairFromFile reads the key-value pair from file and unmarshals it to KVPair. If there is
// any error during the process then it is returned.
func (sp *SimpleFileBasedSnapshotPersistence) getKVPairFromFile(filePath string) (datastore.KVPair, error) {
	kvBytes, ioErr := ioutil.ReadFile(filePath)
	if ioErr != nil {
		return datastore.KVPair{}, ioErr
	}
	var kvPair datastore.KVPair
	if unmarshalErr := json.Unmarshal(kvBytes, &kvPair); unmarshalErr != nil {
		return datastore.KVPair{}, unmarshalErr
	}
	return kvPair, nil
}

// getKVPairFilename returns the filename based on key-value pair and epoch
func (sp *SimpleFileBasedSnapshotPersistence) getKVPairFilename(epoch uint64, key string) string {
	sha256Hash := sha256.Sum256([]byte(key))
	return filepath.Join(sp.SnapshotPath, fmt.Sprintf("%d", epoch), fmt.Sprintf("%x", sha256Hash))
}

// getEpochPath returns the path of the epoch
func (sp *SimpleFileBasedSnapshotPersistence) getEpochPath(epoch uint64) string {
	return filepath.Join(sp.SnapshotPath, fmt.Sprintf("%d", epoch))
}

// SimpleFileBasedSnapshotMetadataPersistence stores snapshot related metadata as a file
type SimpleFileBasedSnapshotMetadataPersistence struct {
	SnapshotMetadataPath string
}

// NewSimpleFileBasedSnapshotMetadataPersistence creates a new instance of simple file-based
// metadata persistence and returns the same
func NewSimpleFileBasedSnapshotMetadataPersistence(snapshotMetadataPath string) *SimpleFileBasedSnapshotMetadataPersistence {
	return &SimpleFileBasedSnapshotMetadataPersistence{SnapshotMetadataPath: snapshotMetadataPath}
}

// PersistMetadata persists snapshot metadata as a file. If there is any error then it is returned
func (smp *SimpleFileBasedSnapshotMetadataPersistence) PersistMetadata(metadata *SnapshotMetadata) error {
	metadataBytes, marshalErr := json.Marshal(metadata)
	if marshalErr != nil {
		return marshalErr
	}
	metadataPath := smp.getSnapshotMetadataPath()
	return ioutil.WriteFile(metadataPath, metadataBytes, 0600)
}

// RetrieveMetadata retrieves snapshot metadata from the disk. If there is any error then it is returned
func (smp *SimpleFileBasedSnapshotMetadataPersistence) RetrieveMetadata() (*SnapshotMetadata, error) {
	metadataBytes, readErr := ioutil.ReadFile(smp.getSnapshotMetadataPath())
	if readErr != nil {
		return nil, readErr
	}
	var snapshotMetadata SnapshotMetadata
	if unmarshalErr := json.Unmarshal(metadataBytes, &snapshotMetadata); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return &snapshotMetadata, nil
}

// getSnapshotMetadataPath returns the path to the snapshot metadata
func (smp *SimpleFileBasedSnapshotMetadataPersistence) getSnapshotMetadataPath() string {
	return filepath.Join(smp.SnapshotMetadataPath, "snapshot-metadata.json")
}
