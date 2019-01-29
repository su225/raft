package state

import "fmt"

// InvalidRoleTransitionError occurs when role
// transition is invalid according to Raft protocol
// For instance, going from Leader to Candidate or
// from Follower to Leader is illegal.
type InvalidRoleTransitionError struct {
	FromRole RaftRole
	ToRole   RaftRole
}

// TermIDMismatchError occurs when the termID supplied
// is not equal to expected term ID
type TermIDMismatchError struct {
	ExpectedTermID, ActualTermID uint64
}

// TermIDMustBeGreaterError occurs when the given term ID
// is less than or equal to some lower bound
type TermIDMustBeGreaterError struct {
	StrictLowerBound uint64
}

// RoleToString converts RaftRole to string
func RoleToString(role RaftRole) string {
	switch role {
	case RoleFollower:
		return "FOLLOWER"
	case RoleCandidate:
		return "CANDIDATE"
	case RoleLeader:
		return "LEADER"
	}
	return ""
}

func (e *InvalidRoleTransitionError) Error() string {
	return fmt.Sprintf("invalid transition: %s -> %s",
		RoleToString(e.FromRole), RoleToString(e.ToRole))
}

func (e *TermIDMismatchError) Error() string {
	return fmt.Sprintf("unexpected term ID. expected=%d, actual=%d",
		e.ExpectedTermID, e.ActualTermID)
}

func (e *TermIDMustBeGreaterError) Error() string {
	return fmt.Sprintf("term ID must be greater than %d", e.StrictLowerBound)
}
