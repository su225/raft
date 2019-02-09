package election

import (
	"testing"

	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/mock"
	"github.com/su225/raft/node/state"
)

func TestRaftLeaderElectionAlgorithmReturnsTrueWhenItGetsMajorityVotesAndBecomesLeader(t *testing.T) {
	currentNodeID := mock.SampleNodeID0
	mockClient := newMockVoteRequesterClient([]string{
		currentNodeID,
		mock.SampleNodeID1,
		mock.SampleNodeID2,
	})
	mockStateManager := state.GetDefaultMockRaftStateManager(currentNodeID, true)
	electionAlgorithm := NewRaftLeaderElectionAlgorithm(
		currentNodeID,
		mockClient,
		mockStateManager,
		log.GetDefaultMockWriteAheadLogManager(true),
		cluster.GetDefaultMockMembershipManager(currentNodeID),
	)
	votedAsLeader, _ := electionAlgorithm.ConductElection()
	if !votedAsLeader {
		t.FailNow()
	}
	if mockStateManager.BecomeCandidateCount < 1 ||
		mockStateManager.UpgradeToLeaderCount < 1 {
		t.FailNow()
	}
}

func TestRaftLeaderElectionAlgorithmReturnsFalseWhenItDoesNotGetMajorityAndDoesNotBecomeLeader(t *testing.T) {
	currentNodeID := mock.SampleNodeID0
	mockClient := newMockVoteRequesterClient([]string{currentNodeID})
	mockStateManager := state.GetDefaultMockRaftStateManager(currentNodeID, true)
	electionAlgorithm := NewRaftLeaderElectionAlgorithm(
		currentNodeID,
		mockClient,
		mockStateManager,
		log.GetDefaultMockWriteAheadLogManager(true),
		cluster.GetDefaultMockMembershipManager(currentNodeID),
	)
	votedAsLeader, _ := electionAlgorithm.ConductElection()
	if votedAsLeader {
		t.FailNow()
	}
	if mockStateManager.BecomeCandidateCount < 1 ||
		mockStateManager.UpgradeToLeaderCount > 0 {
		t.FailNow()
	}
}

// mockVoteRequesterClient represents the mocked RPC Client
// which is used to request votes. This is for TESTING purposes
type mockVoteRequesterClient struct {
	VoteGranters []string
}

// NewMockVoteRequesterClient creates a new instance of mock
// RPC client to be used for requesting votes only
func newMockVoteRequesterClient(voteGranters []string) *mockVoteRequesterClient {
	return &mockVoteRequesterClient{VoteGranters: voteGranters}
}

// RequestVote checks the nodeID against the list of vote granters and grants
// votes if the name of the node is in the list of granters
func (m *mockVoteRequesterClient) RequestVote(curTermID uint64, nodeID string, lastLogEntryID log.EntryID) (bool, error) {
	for _, granterNodeID := range m.VoteGranters {
		if granterNodeID == nodeID {
			return true, nil
		}
	}
	return false, nil
}

// Heartbeat panics since the election algorithm should not be heartbeating
// to other nodes as part of election algorithm
func (m *mockVoteRequesterClient) Heartbeat(curTermID uint64, nodeID string, maxCommittedIndex uint64) (bool, error) {
	panic("cannot use heartbeat as part of raft election algorithm")
}

// AppendEntry panics since the election algorithm should not be calling appendEntry
// as part of its algorithm
func (m *mockVoteRequesterClient) AppendEntry(curTermID uint64, nodeID string, prevEntryID log.EntryID, index uint64, entry log.Entry) (bool, error) {
	panic("cannot use appendEntry as part of raft election algorithm")
}

// InstallSnapshot panics since the election algorithm should not be using it.
func (m *mockVoteRequesterClient) InstallSnapshot(curTermID uint64, nodeID string) (uint64, error) {
	panic("cannot use installSnapshot as part of raft election algorithm")
}

// Start is a no-op in testing scenarios
func (m *mockVoteRequesterClient) Start() error {
	return nil
}

// Destroy is a no-op in testing scenarios
func (m *mockVoteRequesterClient) Destroy() error {
	return nil
}
