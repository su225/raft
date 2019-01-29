package state

import "errors"

var errMockStateManagerErr = errors.New("state-manager error")

// MockRaftStateManager is the implementation of RaftStateManager
// that must be used ONLY FOR TESTING PURPOSES
type MockRaftStateManager struct {
	ShouldSucceed bool
	RaftState

	DowngradeToFollowerCount uint64
	UpgradeToLeaderCount     uint64
	BecomeCandidateCount     uint64
	SetVotedForTermCount     uint64
}

// NewMockRaftStateManager returns a new mock instance of raft state
// manager at the given state. Succeed flag indicates if operations
// invoked on this instance must succeed
func NewMockRaftStateManager(succeed bool, state RaftState) *MockRaftStateManager {
	return &MockRaftStateManager{
		ShouldSucceed:            succeed,
		RaftState:                state,
		DowngradeToFollowerCount: 0,
		UpgradeToLeaderCount:     0,
		BecomeCandidateCount:     0,
	}
}

// DowngradeToFollower is not supported for mock. It just keeps a count of
// how many times the method was called
func (s *MockRaftStateManager) DowngradeToFollower(leaderNodeID string, termID uint64) error {
	if !s.ShouldSucceed {
		return errMockStateManagerErr
	}
	s.DowngradeToFollowerCount++
	return nil
}

// UpgradeToLeader is not supported for mock. It just keeps a counter
func (s *MockRaftStateManager) UpgradeToLeader(termID uint64) error {
	if !s.ShouldSucceed {
		return errMockStateManagerErr
	}
	s.UpgradeToLeaderCount++
	return nil
}

// BecomeCandidate is not supported for mock. It just keeps a counter
func (s *MockRaftStateManager) BecomeCandidate(termID uint64) error {
	if !s.ShouldSucceed {
		return errMockStateManagerErr
	}
	s.BecomeCandidateCount++
	return nil
}

// SetVotedForTerm is not supported for mock. It just increments a counter
func (s *MockRaftStateManager) SetVotedForTerm(termID uint64, votedFor string) error {
	if !s.ShouldSucceed {
		return errMockStateManagerErr
	}
	s.SetVotedForTermCount++
	return nil
}

// GetVotedForTerm returns current term ID and the node to which this node voted in that term
func (s *MockRaftStateManager) GetVotedForTerm() (uint64, string) {
	return s.RaftState.CurrentTermID, s.RaftState.VotedFor
}

// GetRaftState returns the Raft state
func (s *MockRaftStateManager) GetRaftState() RaftState {
	return s.RaftState
}

// GetCurrentTermID returns the current term ID
func (s *MockRaftStateManager) GetCurrentTermID() uint64 {
	return s.CurrentTermID
}

// GetCurrentRole returns the current role
func (s *MockRaftStateManager) GetCurrentRole() RaftRole {
	return s.CurrentRole
}

// GetCurrentLeader returns the current leader if known
func (s *MockRaftStateManager) GetCurrentLeader() (string, bool) {
	return s.CurrentLeader, len(s.CurrentLeader) > 0
}

// Recover is a no-op
func (s *MockRaftStateManager) Recover() error {
	return nil
}

// Start is a no-op
func (s *MockRaftStateManager) Start() error {
	return nil
}

// Destroy is a no-op
func (s *MockRaftStateManager) Destroy() error {
	return nil
}
