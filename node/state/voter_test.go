package state_test

import (
	"testing"

	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/mock"
	"github.com/su225/raft/node/state"
)

func TestDecideVoteShouldGrantIfRemoteNodeIsInHigherTermAndAtLeastAsUptoDateAsCurrent(t *testing.T) {
	mockRaftStateManager := getMockRaftStateManagerWithDefaults(true)
	mockWriteAheadLogManager := getMockWriteAheadLogManagerWithDefaults(true)
	voter := state.NewVoter(mockRaftStateManager, mockWriteAheadLogManager)
	votingDecision, votingErr := voter.DecideVote(mock.SampleNodeID1, 3, log.EntryID{TermID: 2, Index: 2})
	if votingErr != nil || votingDecision == false || mockRaftStateManager.SetVotedForTermCount != 1 {
		t.FailNow()
	}
}

func TestDecideVoteShouldRejectIfRemoteTermIsLessThanOrEqualToCurrentTerm(t *testing.T) {
	mockRaftStateManager := getMockRaftStateManagerWithDefaults(true)
	mockWriteAheadLogManager := getMockWriteAheadLogManagerWithDefaults(true)
	voter := state.NewVoter(mockRaftStateManager, mockWriteAheadLogManager)
	votingDecision, votingErr := voter.DecideVote(mock.SampleNodeID1, 2, log.EntryID{TermID: 2, Index: 2})
	if votingErr != nil || votingDecision == true || mockRaftStateManager.SetVotedForTermCount != 0 {
		t.FailNow()
	}
}

func TestDecideVoteShouldRejectIfTermIsHigherButLogIsNotUptoDate(t *testing.T) {
	mockRaftStateManager := getMockRaftStateManagerWithDefaults(true)
	mockWriteAheadLogManager := getMockWriteAheadLogManagerWithDefaults(true)
	voter := state.NewVoter(mockRaftStateManager, mockWriteAheadLogManager)
	votingDecision, votingErr := voter.DecideVote(mock.SampleNodeID1, 3, log.EntryID{TermID: 1, Index: 1})
	if votingErr != nil || votingDecision == true || mockRaftStateManager.SetVotedForTermCount != 0 {
		t.FailNow()
	}
}

func TestDecideVoteShouldFailIfRaftStatePersistenceFails(t *testing.T) {
	mockRaftStateManager := getMockRaftStateManagerWithDefaults(false)
	mockWriteAheadLogManager := getMockWriteAheadLogManagerWithDefaults(true)
	voter := state.NewVoter(mockRaftStateManager, mockWriteAheadLogManager)
	votingDecision, votingErr := voter.DecideVote(mock.SampleNodeID1, 3, log.EntryID{TermID: 2, Index: 2})
	if votingErr == nil || votingDecision == true || mockRaftStateManager.SetVotedForTermCount != 0 {
		t.FailNow()
	}
}

func getMockRaftStateManagerWithDefaults(succeeds bool) *state.MockRaftStateManager {
	return state.NewMockRaftStateManager(succeeds, state.RaftState{
		RaftDurableState: state.RaftDurableState{
			CurrentNodeID: mock.SampleNodeID0,
			CurrentTermID: 2,
			VotedFor:      "",
		},
		CurrentRole:   state.RoleFollower,
		CurrentLeader: "",
	})
}

func getMockWriteAheadLogManagerWithDefaults(succeeds bool) *log.MockWriteAheadLogManager {
	return log.NewMockWriteAheadLogManager(succeeds, log.WriteAheadLogMetadata{
		TailEntryID: log.EntryID{
			TermID: 2,
			Index:  2,
		},
		MaxCommittedIndex: 2,
	}, map[uint64]log.Entry{
		0: &log.SentinelEntry{},
		1: &log.UpsertEntry{TermID: 1, Key: "a", Value: "1"},
		2: &log.DeleteEntry{TermID: 2, Key: "a"},
	})
}
