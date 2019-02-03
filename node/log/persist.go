package log

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// EntryPersistence defines the operations that must be
// supported to handle persistence of write-ahead log entries
type EntryPersistence interface {
	// PersistEntry persists the log entry. If there is any
	// error on persistence, then it returns the error message
	PersistEntry(index uint64, entry Entry) error

	// RetrieveEntry retrieves the entry at the given index. If
	// the entry does not exist then error is returned.
	RetrieveEntry(index uint64) (entry Entry, err error)
}

// MetadataPersistence defines the operations that must be
// supported to handle persistence of write-ahead log metadata
type MetadataPersistence interface {
	// PersistMetadata persists metadata to the durable storage
	// In case of trouble in persistence, error is returned
	PersistMetadata(metadata *WriteAheadLogMetadata) error

	// RetrieveMetadata retrieves metadata from the durable
	// storage. If there is any issue then error is returned
	// along with nil for metadata.
	RetrieveMetadata() (*WriteAheadLogMetadata, error)
}

// FileBasedEntryPersistence is based on persisting each entry
// as a file with index as the name of the entry. Each entry is
// persisted in JSON format
type FileBasedEntryPersistence struct {
	// EntryDirectory is the directory in which entries are stored
	EntryDirectory string
}

// NewFileBasedEntryPersistence creates a new instance of file based
// entry persistence and returns the same. entryDir is the directory
// in which entries are stored.
func NewFileBasedEntryPersistence(entryDir string) *FileBasedEntryPersistence {
	return &FileBasedEntryPersistence{EntryDirectory: entryDir}
}

// PersistEntry persists entry to the file with the name of the file being index
// If there is any error in the operation it is returned
func (ep *FileBasedEntryPersistence) PersistEntry(index uint64, entry Entry) error {
	entryOnDisk := ep.getPersistableEntry(entry)
	marshaledEntry, marshalErr := json.Marshal(entryOnDisk)
	if marshalErr != nil {
		return marshalErr
	}
	entryFilePath := ep.getEntryFilePath(index)
	if writeErr := ioutil.WriteFile(entryFilePath, marshaledEntry, 0600); writeErr != nil {
		return writeErr
	}
	return nil
}

// RetrieveEntry retrieves entry for the given index. Here it just reads the file,
// unmarshals the JSON and returns the entry. If there is any error during these
// operations then it is returned
func (ep *FileBasedEntryPersistence) RetrieveEntry(index uint64) (Entry, error) {
	entryFilePath := ep.getEntryFilePath(index)
	entryBytes, readErr := ioutil.ReadFile(entryFilePath)
	if readErr != nil {
		return nil, readErr
	}
	var entry persistableEntry
	unmarshalErr := json.Unmarshal(entryBytes, &entry)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}
	inMemoryEntry := ep.getEntry(&entry)
	return inMemoryEntry, nil
}

// persistableEntry is the format in which an entry resides on disk
type persistableEntry struct {
	EntryTypeCode string `json:"entry_type"`
	TermID        uint64 `json:"term_id"`
	Key           string `json:"key"`
	Value         string `json:"value"`
}

// getEntryFilePath returns the name of the file containing entry for a given index
func (ep *FileBasedEntryPersistence) getEntryFilePath(index uint64) string {
	return fmt.Sprintf("%s/%d", ep.EntryDirectory, index)
}

// getPersistableEntry converts the entry to the persistable format for storage
func (ep *FileBasedEntryPersistence) getPersistableEntry(entry Entry) *persistableEntry {
	switch e := entry.(type) {
	case *UpsertEntry:
		return &persistableEntry{
			EntryTypeCode: "A",
			TermID:        e.GetTermID(),
			Key:           e.Key,
			Value:         e.Value,
		}
	case *DeleteEntry:
		return &persistableEntry{
			EntryTypeCode: "D",
			TermID:        e.GetTermID(),
			Key:           e.Key,
		}
	default:
		return &persistableEntry{
			EntryTypeCode: "?",
			TermID:        0,
		}
	}
	return nil
}

// getEntry converts the entry from persistable format to the in-memory format. If the
// EntryTypeCode is not understood then SentinelEntry is returned
func (ep *FileBasedEntryPersistence) getEntry(entry *persistableEntry) Entry {
	switch entry.EntryTypeCode {
	case "A":
		return &UpsertEntry{
			TermID: entry.TermID,
			Key:    entry.Key,
			Value:  entry.Value,
		}
	case "D":
		return &DeleteEntry{
			TermID: entry.TermID,
			Key:    entry.Key,
		}
	default:
		return &SentinelEntry{}
	}
	return nil
}

// FileBasedMetadataPersistence persists metadata to the file. The metadata
// is stored in JSON format.
type FileBasedMetadataPersistence struct {
	// MetadataPath is the path to the file in which
	// write-ahead log metadata is stored
	MetadataPath string
}

// NewFileBasedMetadataPersistence creates a new instance of file-based metadata
// persistence and returns it. "metadataPath" is the path to the file where metadata
// must be stored
func NewFileBasedMetadataPersistence(metadataPath string) *FileBasedMetadataPersistence {
	return &FileBasedMetadataPersistence{MetadataPath: metadataPath}
}

// PersistMetadata persists write-ahead log metadata to the specified file
func (mp *FileBasedMetadataPersistence) PersistMetadata(metadata *WriteAheadLogMetadata) error {
	marshaledBytes, marshalErr := json.Marshal(metadata)
	if marshalErr != nil {
		return marshalErr
	}
	if writeErr := ioutil.WriteFile(mp.MetadataPath, marshaledBytes, 0600); writeErr != nil {
		return writeErr
	}
	return nil
}

// RetrieveMetadata reads the metadata from the file specified, parses the JSON and returns the
// in-memory representation of metadata - that is WriteAheadLogMetadata. If there is any error
// during IO or unmarshaling then it is returned
func (mp *FileBasedMetadataPersistence) RetrieveMetadata() (*WriteAheadLogMetadata, error) {
	metadataBytes, readErr := ioutil.ReadFile(mp.MetadataPath)
	if readErr != nil {
		return nil, readErr
	}
	var metadata WriteAheadLogMetadata
	if unmarshalErr := json.Unmarshal(metadataBytes, &metadata); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return &metadata, nil
}
