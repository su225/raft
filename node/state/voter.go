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
