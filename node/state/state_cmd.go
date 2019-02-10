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

type raftStateManagerCommand interface {
	IsRaftStateManagerCommand() bool
}

type raftStateManagerStart struct {
	raftStateManagerCommand
	errorChan chan error
}

type raftStateManagerDestroy struct {
	raftStateManagerCommand
	errorChan chan error
}

type raftStateManagerRecover struct {
	raftStateManagerCommand
	errorChan chan error
}

type downgradeToFollower struct {
	raftStateManagerCommand
	leaderNodeID string
	remoteTermID uint64
	errorChan    chan error
}

type upgradeToLeader struct {
	raftStateManagerCommand
	leaderTermID uint64
	errorChan    chan error
}

type becomeCandidate struct {
	raftStateManagerCommand
	electionTermID uint64
	errorChan      chan error
}

type setVotedForTerm struct {
	raftStateManagerCommand
	votingTermID uint64
	votedForNode string
	errorChan    chan error
}

type getVotedForTerm struct {
	raftStateManagerCommand
	replyChan chan *getVotedForTermReply
}

type getVotedForTermReply struct {
	votedForNode string
	votingTermID uint64
}

type getRaftState struct {
	raftStateManagerCommand
	replyChan chan *getRaftStateReply
}

type getRaftStateReply struct {
	state RaftState
}
