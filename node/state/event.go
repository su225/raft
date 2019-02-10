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

// RaftStateManagerEvent represents the event emitted
// by the RaftStateManager.
type RaftStateManagerEvent interface {
	IsRaftStateManagerEvent() bool
}

// UpgradeToLeaderEvent is emitted when the node is
// upgraded as the leader of the cluster
type UpgradeToLeaderEvent struct {
	RaftStateManagerEvent
	TermID uint64
}

// BecomeCandidateEvent is emitted when the node becomes
// the candidate for some term (happens during leader election)
type BecomeCandidateEvent struct {
	RaftStateManagerEvent
	TermID uint64
}

// DowngradeToFollowerEvent is emitted when the node is
// downgraded to the status of follower
type DowngradeToFollowerEvent struct {
	RaftStateManagerEvent
	TermID        uint64
	CurrentLeader string
}
