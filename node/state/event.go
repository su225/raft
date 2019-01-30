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
