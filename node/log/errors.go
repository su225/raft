package log

import "fmt"

// CommittedIndexTooFarAheadError is returned when the commit index to
// be updated is ahead of tail index.
type CommittedIndexTooFarAheadError struct {
	UpperLimit       uint64
	GivenCommitIndex uint64
}

// CannotWriteEntryError is returned when the entry cannot be
// written at a specified index
type CannotWriteEntryError struct {
	IndexAttempted uint64
	CommittedIndex uint64
	TailIndex      uint64
}

// EntryIDInvariantViolationError is returned when two successive
// entries don't satisfy invariants like term ID must be non-decreasing
// and their indices should differ by exactly 1.
type EntryIDInvariantViolationError struct {
	Message string
}

// CommittedIndexMonotonicityViolation is returned during forcibly
// setting metadata where committed index to be set is much behind
// the current commit index
type CommittedIndexMonotonicityViolation struct {
	CurCommittedIndex       uint64
	AttemptedCommittedIndex uint64
}

func (e *CommittedIndexTooFarAheadError) Error() string {
	return fmt.Sprintf("given commit index %d is ahead of the end %d",
		e.UpperLimit, e.GivenCommitIndex)
}

func (e *CannotWriteEntryError) Error() string {
	return fmt.Sprintf("entry cannot be written to %d. It must be between %d and %d",
		e.IndexAttempted, e.CommittedIndex, e.TailIndex+1)
}

func (e *EntryIDInvariantViolationError) Error() string {
	return fmt.Sprintf("entry ID invariant violation: %s", e.Message)
}

func (e *CommittedIndexMonotonicityViolation) Error() string {
	return fmt.Sprintf("commit index must be monotonically increasing (%d < %d)",
		e.AttemptedCommittedIndex, e.CurCommittedIndex)
}

// InvalidEpochError is returned when the valid epoch bounds
// for the operation are violated
type InvalidEpochError struct {
	StrictLowerBound uint64
	StrictUpperBound uint64
}

// SnapshotMetadataMonotonicityViolationError is returned when
// the metadata to be set either has lower epoch or same epoch
// with lower index
type SnapshotMetadataMonotonicityViolationError struct {
	BeforeMetadata SnapshotMetadata
	AfterMetadata  SnapshotMetadata
}

// KeyValuePairDoesNotExistInSnapshotError is returned when the
// key-value pair does not exist in the snapshot
type KeyValuePairDoesNotExistInSnapshotError struct {
	Key   string
	Epoch uint64
}

func (e *InvalidEpochError) Error() string {
	return fmt.Sprintf("epoch e must satisfy [%d < e < %d]",
		e.StrictLowerBound, e.StrictUpperBound)
}

func (e *SnapshotMetadataMonotonicityViolationError) Error() string {
	return fmt.Sprintf("snapshot metadata monotonicity error. Before=%v, After=%v",
		e.BeforeMetadata, e.AfterMetadata)
}

func (e *KeyValuePairDoesNotExistInSnapshotError) Error() string {
	return fmt.Sprintf("key %s does not exist in snapshot(epoch:%d)",
		e.Key, e.Epoch)
}
