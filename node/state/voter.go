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

package state

import (
	"github.com/su225/raft/node/log"
)

// Voter is responsible for deciding whether to
// grant or deny vote for a remote node
type Voter struct {
	RaftStateManager
	log.WriteAheadLogManager
}

// NewVoter initializes a new instance of voter which
// gets information from given RaftStateManager and
// log metadata from WriteAheadLogManager
func NewVoter(stateMgr RaftStateManager, waLogMgr log.WriteAheadLogManager) *Voter {
	return &Voter{
		RaftStateManager:     stateMgr,
		WriteAheadLogManager: waLogMgr,
	}
}

// DecideVote decides whether to grant or deny vote to the
// requesting remote node with given term ID (remoteTermID) and
// its write-ahead log's tail entry ID (remoteTailEntryID). If
// the decision is 'grant' then it tries to persist and if it
// fails then it is as good as 'deny' and error is returned. If
// the decision is grant then true is returned with nil for error
func (v *Voter) DecideVote(remoteNodeID string, remoteTermID uint64, remoteTailEntryID log.EntryID) (bool, error) {
	curRaftState := v.RaftStateManager.GetRaftState()
	curWriteAheadLogMetadata, _ := v.WriteAheadLogManager.GetMetadata()

	currentTermID := curRaftState.CurrentTermID
	if currentTermID >= remoteTermID {
		return false, nil
	}
	currentTailEntryID := curWriteAheadLogMetadata.TailEntryID
	if (currentTailEntryID.TermID > remoteTailEntryID.TermID) ||
		((currentTailEntryID.TermID == remoteTailEntryID.TermID) &&
			(currentTailEntryID.Index > remoteTailEntryID.Index)) {
		return false, nil
	}
	if setVotedErr := v.RaftStateManager.SetVotedForTerm(remoteTermID, remoteNodeID); setVotedErr != nil {
		return false, setVotedErr
	}
	return true, nil
}
