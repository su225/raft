package replication

import (
	"fmt"

	"github.com/su225/raft/node/log"
)

// EntryReplicationFailedError occurs when replication of log entries
// is not successful till the given target tail index
type EntryReplicationFailedError struct {
	TargetTailEntryID log.EntryID
}

// EntryCannotBeCommittedError is raised when the log upto the given
// index cannot be committed
type EntryCannotBeCommittedError struct {
	Index uint64
}

func (e *EntryReplicationFailedError) Error() string {
	return fmt.Sprintf("entry replication failed till (%d,%d)",
		e.TargetTailEntryID.TermID,
		e.TargetTailEntryID.Index)
}

func (e *EntryCannotBeCommittedError) Error() string {
	return fmt.Sprintf("log cannot be committed till index %d", e.Index)
}
