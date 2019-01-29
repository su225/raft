package state

import (
	"testing"

	"github.com/su225/raft/node/mock"
)

func TestDowngradeToFollowerMustDowngradeIfRemoteTermIDIsHigher(t *testing.T) {
	state := getRaftStateForLeader(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleDowngradeToFollower(state, &downgradeToFollower{
		leaderNodeID: "",
		remoteTermID: 12,
	})
	if err != nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 12 ||
		state.RaftDurableState.VotedFor != "" ||
		state.CurrentRole != RoleFollower ||
		state.CurrentLeader != "" {
		t.FailNow()
	}
}

func TestDowngradeToFollowerMustDowngradeIfRemoteTermIDIsEqualAndCurrentNodeIsCandidate(t *testing.T) {
	state := getRaftStateForCandidate(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleDowngradeToFollower(state, &downgradeToFollower{
		leaderNodeID: mock.SampleNodeID1,
		remoteTermID: 10,
	})
	if err != nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleFollower ||
		state.CurrentLeader != mock.SampleNodeID1 {
		t.FailNow()
	}
}

func TestDowngradeToFollowerMustUpdateLeaderIfRemoteTermIDIsEqualAndCurrentNodeIsFollower(t *testing.T) {
	state := getRaftStateForFollower(mock.SampleNodeID0, 10, mock.SampleNodeID1, "")
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleDowngradeToFollower(state, &downgradeToFollower{
		leaderNodeID: mock.SampleNodeID1,
		remoteTermID: 10,
	})
	if err != nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID1 ||
		state.CurrentRole != RoleFollower ||
		state.CurrentLeader != mock.SampleNodeID1 {
		t.FailNow()
	}
}

func TestDowngradeToFollowerMustNotDowngradeIfRemoteTermIDIsLowerThanCurrent(t *testing.T) {
	state := getRaftStateForLeader(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleDowngradeToFollower(state, &downgradeToFollower{
		leaderNodeID: mock.SampleNodeID1,
		remoteTermID: 8,
	})
	if err != nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleLeader ||
		state.CurrentLeader != mock.SampleNodeID0 {
		t.FailNow()
	}
}

func TestDowngradeToFollowerMustFailWithErrorIfStatePersistenceFailsDuringTermIDSwitchButStillStepDown(t *testing.T) {
	state := getRaftStateForLeader(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(false, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleDowngradeToFollower(state, &downgradeToFollower{
		leaderNodeID: mock.SampleNodeID2,
		remoteTermID: 12,
	})
	if err == nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleFollower ||
		state.CurrentLeader != mock.SampleNodeID2 {
		t.FailNow()
	}
}

func TestUpgradeToLeaderMustSucceedIfTermIDGivenAndCurrentTermIDMatchAndNodeIsCandidate(t *testing.T) {
	state := getRaftStateForCandidate(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleUpgradeToLeader(state, &upgradeToLeader{leaderTermID: 10})
	if err != nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleLeader ||
		state.CurrentLeader != mock.SampleNodeID0 {
		t.FailNow()
	}
}

func TestUpgradeToLeaderMustSucceedEvenIfPersistenceFails(t *testing.T) {
	state := getRaftStateForCandidate(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(false, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleUpgradeToLeader(state, &upgradeToLeader{leaderTermID: 10})
	if err != nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleLeader ||
		state.CurrentLeader != mock.SampleNodeID0 {
		t.FailNow()
	}
}

func TestUpgradeToLeaderMustFailIfTermIDDoesNotMatch(t *testing.T) {
	state := getRaftStateForCandidate(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleUpgradeToLeader(state, &upgradeToLeader{leaderTermID: 12})
	if err == nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleCandidate ||
		state.CurrentLeader != "" {
		t.FailNow()
	}
}

func TestUpgradeToLeaderMustBeNoOpIfCurrentNodeIsAlreadyLeaderForGivenTerm(t *testing.T) {
	state := getRaftStateForLeader(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	err := raftStateMgr.handleUpgradeToLeader(state, &upgradeToLeader{leaderTermID: 10})
	if err != nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleLeader ||
		state.CurrentLeader != mock.SampleNodeID0 {
		t.FailNow()
	}
}

func TestBecomeCandidateMustFailIfGivenTermIsNotHigherThanCurrentTerm(t *testing.T) {
	state := getRaftStateForFollower(mock.SampleNodeID0, 10, "", "")
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	if err := raftStateMgr.handleBecomeCandidate(state, &becomeCandidate{electionTermID: 10}); err == nil {
		t.FailNow()
	}
	if err := raftStateMgr.handleBecomeCandidate(state, &becomeCandidate{electionTermID: 9}); err == nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != "" ||
		state.CurrentRole != RoleFollower ||
		state.CurrentLeader != "" {
		t.FailNow()
	}

}

func TestBecomeCandidateMustFailIfCurrentRoleIsLeader(t *testing.T) {
	state := getRaftStateForLeader(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	if err := raftStateMgr.handleBecomeCandidate(state, &becomeCandidate{electionTermID: 11}); err == nil {
		t.FailNow()
	}
	if state.RaftDurableState.CurrentTermID != 10 ||
		state.RaftDurableState.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleLeader ||
		state.CurrentLeader != mock.SampleNodeID0 {
		t.FailNow()
	}
}

func TestSetVotedForTermIsSuccessfulIfTheRemoteTermIsGreaterThanCurrentTerm(t *testing.T) {
	defaultTermID := uint64(10)
	setVotedForMessage := &setVotedForTerm{
		votingTermID: defaultTermID + 2,
		votedForNode: mock.SampleNodeID2,
	}
	containsExpectedValues := func(s *raftStateManagerState) bool {
		return s.CurrentRole == RoleFollower &&
			s.CurrentTermID == defaultTermID+2 &&
			s.VotedFor == mock.SampleNodeID2
	}
	t.Run("Role=Leader", func(t *testing.T) {
		state := getRaftStateForLeader(mock.SampleNodeID0, defaultTermID)
		persistence := NewInMemoryRaftStatePersistence(true, nil)
		raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
		if err := raftStateMgr.handleSetVotedForTerm(state, setVotedForMessage); err != nil {
			t.FailNow()
		}
		if !containsExpectedValues(state) {
			t.FailNow()
		}
	})
	t.Run("Role=Candidate", func(t *testing.T) {
		state := getRaftStateForCandidate(mock.SampleNodeID0, defaultTermID)
		persistence := NewInMemoryRaftStatePersistence(true, nil)
		raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
		if err := raftStateMgr.handleSetVotedForTerm(state, setVotedForMessage); err != nil {
			t.FailNow()
		}
		if !containsExpectedValues(state) {
			t.FailNow()
		}
	})
	t.Run("Role=Follower", func(t *testing.T) {
		state := getRaftStateForFollower(mock.SampleNodeID0, defaultTermID, mock.SampleNodeID1, mock.SampleNodeID1)
		persistence := NewInMemoryRaftStatePersistence(true, nil)
		raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
		if err := raftStateMgr.handleSetVotedForTerm(state, setVotedForMessage); err != nil {
			t.FailNow()
		}
		if !containsExpectedValues(state) {
			t.FailNow()
		}
	})
}

func TestSetVotedForTermFailsIfRemoteTermIsLessThanOrEqualToCurrentTerm(t *testing.T) {
	state := getRaftStateForFollower(mock.SampleNodeID0, 10, "", "")
	persistence := NewInMemoryRaftStatePersistence(true, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	setVotedForMessage := &setVotedForTerm{votingTermID: 10, votedForNode: mock.SampleNodeID2}
	if err := raftStateMgr.handleSetVotedForTerm(state, setVotedForMessage); err == nil {
		t.FailNow()
	}
	if state.CurrentTermID != 10 ||
		state.VotedFor != "" ||
		state.CurrentRole != RoleFollower ||
		state.CurrentLeader != "" {
		t.FailNow()
	}
}

func TestSetVotedForTermFailsIfPersistenceFails(t *testing.T) {
	state := getRaftStateForCandidate(mock.SampleNodeID0, 10)
	persistence := NewInMemoryRaftStatePersistence(false, nil)
	raftStateMgr := NewRealRaftStateManager(mock.SampleNodeID0, persistence)
	setVotedForMessage := &setVotedForTerm{votingTermID: 12, votedForNode: mock.SampleNodeID2}
	if err := raftStateMgr.handleSetVotedForTerm(state, setVotedForMessage); err == nil {
		t.FailNow()
	}
	if state.CurrentTermID != 10 ||
		state.VotedFor != mock.SampleNodeID0 ||
		state.CurrentRole != RoleFollower ||
		state.CurrentLeader != "" {
		t.FailNow()
	}
}

func getRaftStateForLeader(nodeID string, termID uint64) *raftStateManagerState {
	return &raftStateManagerState{
		isStarted:   true,
		isDestroyed: false,
		RaftState: RaftState{
			RaftDurableState: RaftDurableState{
				CurrentNodeID: nodeID,
				CurrentTermID: termID,
				VotedFor:      nodeID,
			},
			CurrentRole:   RoleLeader,
			CurrentLeader: nodeID,
		},
	}
}

func getRaftStateForCandidate(nodeID string, candidateTermID uint64) *raftStateManagerState {
	return &raftStateManagerState{
		isStarted:   true,
		isDestroyed: false,
		RaftState: RaftState{
			RaftDurableState: RaftDurableState{
				CurrentNodeID: nodeID,
				CurrentTermID: candidateTermID,
				VotedFor:      nodeID,
			},
			CurrentRole:   RoleCandidate,
			CurrentLeader: "",
		},
	}
}

func getRaftStateForFollower(nodeID string, termID uint64, votedFor, currentLeaderID string) *raftStateManagerState {
	return &raftStateManagerState{
		isStarted:   true,
		isDestroyed: false,
		RaftState: RaftState{
			RaftDurableState: RaftDurableState{
				CurrentNodeID: nodeID,
				CurrentTermID: termID,
				VotedFor:      votedFor,
			},
			CurrentRole:   RoleFollower,
			CurrentLeader: currentLeaderID,
		},
	}
}
