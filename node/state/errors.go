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
