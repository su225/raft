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
