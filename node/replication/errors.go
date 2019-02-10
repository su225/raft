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
