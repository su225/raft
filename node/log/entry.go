package log

// Entry represents an entry in the write-ahead log
type Entry interface {
	// GetTermID returns the termID in which the
	// current log entry is created
	GetTermID() uint64
}

// SentinelEntry represents the entry at index 0
// and termID 0. This is not a real entry
type SentinelEntry struct{}

// GetTermID always returns 0 for sentinel entry
func (s *SentinelEntry) GetTermID() uint64 { return 0 }

// UpsertEntry represents the operation to add the key-value
// pair to the store. If the key exists, then its value is
// updated (and the old version is destroyed)
type UpsertEntry struct {
	TermID uint64
	Key    string
	Value  string
}

// GetTermID for an upsert entry returns the termID in
// which it was created.
func (u *UpsertEntry) GetTermID() uint64 {
	return u.TermID
}

// DeleteEntry represents the operation to delete a key value
// pair. If the key does not exist or is already delete then
// that would be a no-op
type DeleteEntry struct {
	TermID uint64
	Key    string
}

// GetTermID for a delete entry returns the termID in
// which it was created.
func (d *DeleteEntry) GetTermID() uint64 {
	return d.TermID
}
