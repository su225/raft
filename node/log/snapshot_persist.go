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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/su225/raft/node/data"
)

// SnapshotMetadata represents the metadata associated
// with the snapshot like its epoch and index
type SnapshotMetadata struct {
	Epoch        uint64
	LastLogEntry Entry
	EntryID
}

// SnapshotPersistence is responsible for persistence
// operations of the snapshot - like storing/deleting key-value
// pairs, creating new snapshot epoch and so on
type SnapshotPersistence interface {
	// PersistKeyValuePair persists the given key-value pair in the given epoch.
	// If there is any issue during the operation then error is returned
	PersistKeyValuePair(epoch uint64, keyValuePair data.KVPair) error

	// RetrieveKeyValuePair retrieves the key-value pair for the given key given
	// the epoch. If there is any retrieval related errors then it is returned
	RetrieveKeyValuePair(epoch uint64, key string) (data.KVPair, error)

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
	ForEachKey(epoch uint64, operation func(kv data.KVPair) error) error
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
func (sp *SimpleFileBasedSnapshotPersistence) PersistKeyValuePair(epoch uint64, keyValuePair data.KVPair) error {
	kvPath := sp.getKVPairFilename(epoch, keyValuePair.Key)
	kvBytes, marshalErr := json.Marshal(keyValuePair)
	if marshalErr != nil {
		return marshalErr
	}
	return ioutil.WriteFile(kvPath, kvBytes, 0600)
}

// RetrieveKeyValuePair retrieves the key-value pair in the given epoch. If there is any error
// during the operation then it is reported.
func (sp *SimpleFileBasedSnapshotPersistence) RetrieveKeyValuePair(epoch uint64, key string) (data.KVPair, error) {
	kvPath := sp.getKVPairFilename(epoch, key)
	kvPair, err := sp.getKVPairFromFile(kvPath)
	if err != nil {
		if os.IsNotExist(err) {
			return data.KVPair{}, &KeyValuePairDoesNotExistInSnapshotError{
				Key:   key,
				Epoch: epoch,
			}
		}
		return data.KVPair{}, err
	}
	return kvPair, err
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
func (sp *SimpleFileBasedSnapshotPersistence) ForEachKey(epoch uint64, operation func(kv data.KVPair) error) error {
	epochPath := sp.getEpochPath(epoch)
	return filepath.Walk(epochPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
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
func (sp *SimpleFileBasedSnapshotPersistence) getKVPairFromFile(filePath string) (data.KVPair, error) {
	kvBytes, ioErr := ioutil.ReadFile(filePath)
	if ioErr != nil {
		return data.KVPair{}, ioErr
	}
	var kvPair data.KVPair
	if unmarshalErr := json.Unmarshal(kvBytes, &kvPair); unmarshalErr != nil {
		return data.KVPair{}, unmarshalErr
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

type persistableSnapshotMetadata struct {
	LastEntryID EntryID           `json:"last_entry_id"`
	LastEntry   *persistableEntry `json:"last_entry"`
	Epoch       uint64            `json:"epoch"`
}

func inMemToPersistableSnapshotMetadata(metadata *SnapshotMetadata) *persistableSnapshotMetadata {
	return &persistableSnapshotMetadata{
		LastEntryID: metadata.EntryID,
		LastEntry:   getPersistableEntry(metadata.LastLogEntry),
		Epoch:       metadata.Epoch,
	}
}

func persistableToInMemSnapshotMetadata(metadata *persistableSnapshotMetadata) *SnapshotMetadata {
	return &SnapshotMetadata{
		EntryID:      metadata.LastEntryID,
		LastLogEntry: getInMemEntry(metadata.LastEntry),
		Epoch:        metadata.Epoch,
	}
}

// PersistMetadata persists snapshot metadata as a file. If there is any error then it is returned
func (smp *SimpleFileBasedSnapshotMetadataPersistence) PersistMetadata(metadata *SnapshotMetadata) error {
	persistableMetadata := inMemToPersistableSnapshotMetadata(metadata)
	metadataBytes, marshalErr := json.Marshal(persistableMetadata)
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
	var snapshotMetadata persistableSnapshotMetadata
	if unmarshalErr := json.Unmarshal(metadataBytes, &snapshotMetadata); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return persistableToInMemSnapshotMetadata(&snapshotMetadata), nil
}

// getSnapshotMetadataPath returns the path to the snapshot metadata
func (smp *SimpleFileBasedSnapshotMetadataPersistence) getSnapshotMetadataPath() string {
	return filepath.Join(smp.SnapshotMetadataPath, "snapshot-metadata.json")
}

// IsKeyValuePairDoesNotExistInSnapshotError is a utility method to check if the error
// is due to key-value pair not existing in snapshot
func IsKeyValuePairDoesNotExistInSnapshotError(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*KeyValuePairDoesNotExistInSnapshotError)
	return ok
}
