package state

import (
	"github.com/su225/raft/node/common"
)

// RaftRole represents the role of this node in
// the cluster. It can be either Leader, Follower or Candidate
type RaftRole uint8

const (
	// RoleLeader denotes that the node thinks it is the
	// leader in the cluster for the given term.
	RoleLeader RaftRole = iota

	// RoleCandidate denotes that the node is a candidate for
	// the present term to become a leader. This happens only
	// when the election is triggered by this node (and hence
	// this node doesn't know the leader when it is a candidate)
	RoleCandidate

	// RoleFollower denotes that the node is a follower. The
	// leader of the cluster may or may not be known.
	RoleFollower
)

// RaftDurableState represents the part of the raft state that
// must be recoverable across restarts and hence must be persisted
type RaftDurableState struct {
	// CurrentNodeID represents the name of the
	// current node in the cluster
	CurrentNodeID string `json:"current_node_id"`

	// CurrentTermID is the term in which the present
	// node is on. Term acts as a logical clock for
	// the cluster. This must be monotonically increasing
	CurrentTermID uint64 `json:"current_term_id"`

	// VotedFor is the node to which the current node
	// granted vote in the current term. If it has not
	// voted then it will be an empty string
	VotedFor string `json:"voted_for"`
}

// RaftState represents the state of the node in the cluster -
// both volatile and non-volatile. Since some of them are
// volatile all parts are not recovered.
type RaftState struct {
	// RaftDurableState represents the durable part
	// of the raft-related state of this node
	RaftDurableState

	// CurrentRole represents the current role of the
	// node in the cluster. It always starts from follower
	// (RoleFollower). When this node initiates election
	// then it becomes candidate (RoleCandidate) and when
	// it is elected then it becomes leader (RoleLeader).
	// When the node starts up it will be a follower.
	CurrentRole RaftRole

	// CurrentLeader represents the current leader of the
	// cluster.If the leader for the term is not known then
	// it will be an empty string. When the node starts up
	// it will be an empty string.
	CurrentLeader string
}

// RaftStateManager is responsible for managing state of the
// raft node like keeping track of role, status, voting info
// and so on. It is also responsible for persisting some of
// these information to disk
type RaftStateManager interface {
	// DowngradeToFollower downgrades the current node to the
	// status of follower for the given termID. The node accepts
	// the authority of node called leaderNodeID if it is
	// specified. Otherwise, it will be waiting for the heartbeat
	// from the current cluster leader to update current leader.
	// This operation must be transactional
	DowngradeToFollower(leaderNodeID string, termID uint64) error

	// UpgradeToLeader upgrades the current node to the status
	// of the leader. According to the Raft paper, the node must
	// first be a candidate in the term and obtain majority votes
	// from the cluster in leader election. This operation must
	// be transactional
	UpgradeToLeader(termID uint64) error

	// BecomeCandidate turns this node into a candidate for the
	// term. When the node is a candidate then it will vote for
	// itself and the leader for the term is not known. This is
	// usually invoked while triggering leader election/ This
	// operation must be transactional
	BecomeCandidate(termID uint64) error

	// SetVotedForTerm tries to record the vote for the given
	// term. If the vote is already granted for the term or if
	// there was some error related to persisting the information
	// an error is returned. This must be transactional
	SetVotedForTerm(termID uint64, votedFor string) error

	// GetVotedForTerm returns the node to which the current node
	// voted (it can be itself) in the term returned.
	GetVotedForTerm() (currentTermID uint64, votedFor string)

	// GetRaftState represents a snapshot of the current state
	// of this node. It should not be modified.
	GetRaftState() RaftState

	// GetCurrentTermID returns the current term ID of the node
	GetCurrentTermID() uint64

	// GetCurrentRole returns the current role of the node
	// It can be either leader, follower or candidate
	GetCurrentRole() RaftRole

	// GetCurrentLeader returns the current leader of the raft
	// cluster. If it is not known then it will be an empty string
	// The second return value indicates if the leader is known
	// (true if known, false otherwise)
	GetCurrentLeader() (string, bool)

	// Recoverable indicates that there is some state which
	// must be recoverable between crashes
	common.Recoverable

	// ComponentLifecycle tells that the implementation must
	// be a component which can be started and destroyed
	common.ComponentLifecycle
}

// RealRaftStateManager is the implmentation of RaftStateManager
// It is responsible for managing some part of the state of node
type RealRaftStateManager struct {
	CurrentNodeID  string
	commandChannel chan raftStateManagerCommand
	RaftStatePersistence
}

// NewRealRaftStateManager creates a new instance of real raft state
// manager and returns the same. StatePersistence is the persistence
// mechanism used to persist the state.
func NewRealRaftStateManager(
	currentNodeID string,
	statePersistence RaftStatePersistence,
) *RealRaftStateManager {
	return &RealRaftStateManager{
		CurrentNodeID:        currentNodeID,
		commandChannel:       make(chan raftStateManagerCommand),
		RaftStatePersistence: statePersistence,
	}
}

// Start starts the RealRaftStateManager and makes it operational. If
// the component is already destroyed then it doesn't makes sense.
// This must be an idempotent operation.
func (s *RealRaftStateManager) Start() error {
	go s.loop()
	if recoveryErr := s.Recover(); recoveryErr != nil {
		return recoveryErr
	}
	errorChannel := make(chan error)
	s.commandChannel <- &raftStateManagerStart{errorChan: errorChannel}
	return <-errorChannel
}

// Destroy destroys the component and cleans up the resources owned
// by components like file descriptors, sockets, connections etc. This
// makes the component non-operational. This must be idempotent
func (s *RealRaftStateManager) Destroy() error {
	errorChannel := make(chan error)
	s.commandChannel <- &raftStateManagerDestroy{errorChan: errorChannel}
	return <-errorChannel
}

// Recover is used to recover the state of the RealRaftStateManager
// at startup (usually after a crash). This need not be idempotent.
func (s *RealRaftStateManager) Recover() error {
	errorChannel := make(chan error)
	s.commandChannel <- &raftStateManagerRecover{errorChan: errorChannel}
	return <-errorChannel
}

// DowngradeToFollower downgrades the node to the follower for the given
// term ID. It accepts the authority of the node with ID, leaderNodeID
// and switches to the term if higher.
//
// Valid transition description:
// 1. LEADER -> FOLLOWER => happens when this node discovers that there is a
//    node with higher term. This might happen when there is a network
//    partition and this node ended up in minor part or when another node
//    starts election for the next term and got elected and this node
//    somehow didn't know it (links might be broken)
//
// 2. CANDIDATE -> FOLLOWER => happens when this node started election,
//    but couldn't get elected in the term and found out that there is
//    another node which got elected for the same or higher term. This
//    can also happen when the candidate in the next term requests vote
//    from this node. Here, the node switches to next term and becomes
//    follower, but the leader is not yet known.
//
// 3. FOLLOWER -> FOLLOWER => This might happen when this node finds out
//    that there is a leader node with same or higher term ID. In that
//    case it will accept authority of the leader node and switches to
//    the termID of the discovered leader node
func (s *RealRaftStateManager) DowngradeToFollower(leaderNodeID string, termID uint64) error {
	errorChannel := make(chan error)
	s.commandChannel <- &downgradeToFollower{
		leaderNodeID: leaderNodeID,
		remoteTermID: termID,
		errorChan:    errorChannel,
	}
	return <-errorChannel
}

// UpgradeToLeader upgrades the current node to the leader. Some of the
// pre-conditions that must be satisfied for this to succeed
// 1. The node MUST be a candidate for the speicified termID.
// 2. [Not checked here] The node must obtain majority votes from
//    other nodes in the cluster.
//
// Valid transition description:
// 1. CANDIDATE -> LEADER => happens when this node obtains majority votes
//    in the election for the termID in which it is a candidate.
func (s *RealRaftStateManager) UpgradeToLeader(termID uint64) error {
	return nil
}

// BecomeCandidate transforms the node to a candidate. The node must be a
// follower or candidate in termID-1 before becoming a candidate in termID.
//
// Valid transition description:
// 1. FOLLOWER -> CANDIDATE => happens when this node couldn't get the heartbeat
//    from the leader-node within some period of time (called election timeout).
//    In this case the node switches to the next term, becomes candidate and
//    initiates leader election
//
// 2. CANDIDATE -> CANDIDATE => happens when no candidate obtains majority and
//    there is another election timeout. In this case, the candidate becomes
//    candidate for the next term.
func (s *RealRaftStateManager) BecomeCandidate(termID uint64) error {
	return nil
}

// SetVotedForTerm is used to record the candidate to which vote was granted in
// the current term. If the node has already voted in the current term then it
// cannot vote in this term and hence it is an error.
//
// The operation succeeds under one of the following conditions
// 1. TermID is the same as current term ID and the node has not yet voted
// 2. TermID is higher than the current term ID.
func (s *RealRaftStateManager) SetVotedForTerm(termID uint64, votedFor string) error {
	return nil
}

// GetVotedForTerm returns the node to which the current node voted along with
// the term. This should not result in error or panic.
func (s *RealRaftStateManager) GetVotedForTerm() (uint64, string) {
	return 0, ""
}

// GetCurrentTermID returns the current term ID
func (s *RealRaftStateManager) GetCurrentTermID() uint64 {
	return 0
}

// GetCurrentRole returns the current role of the node
func (s *RealRaftStateManager) GetCurrentRole() RaftRole {
	return RoleFollower
}

// GetCurrentLeader returns the current leader in the cluster
func (s *RealRaftStateManager) GetCurrentLeader() (string, bool) {
	return "", false
}

// GetRaftState returns the snapshot of the current raft state.
func (s *RealRaftStateManager) GetRaftState() RaftState {
	return RaftState{}
}

type raftStateManagerState struct {
	isStarted, isDestroyed bool
	RaftState
}

// loop handles all commands to RaftStateManager. It is also responsible
// for managing state accordingly.
func (s *RealRaftStateManager) loop() {
	state := &raftStateManagerState{
		isStarted:   false,
		isDestroyed: false,
		RaftState: RaftState{
			RaftDurableState: RaftDurableState{
				CurrentNodeID: s.CurrentNodeID,
				CurrentTermID: 0,
				VotedFor:      "",
			},
			CurrentRole:   RoleFollower,
			CurrentLeader: "",
		},
	}

	for {
		cmd := <-s.commandChannel
		switch c := cmd.(type) {
		case *raftStateManagerStart:
			c.errorChan <- s.handleRaftStateManagerStart(state, c)
		case *raftStateManagerDestroy:
			c.errorChan <- s.handleRaftStateManagerDestroy(state, c)
		case *raftStateManagerRecover:
			c.errorChan <- s.handleRaftStateManagerRecover(state, c)
		case *downgradeToFollower:
			c.errorChan <- s.handleDowngradeToFollower(state, c)
		case *upgradeToLeader:
			c.errorChan <- s.handleUpgradeToLeader(state, c)
		case *becomeCandidate:
			c.errorChan <- s.handleBecomeCandidate(state, c)
		case *setVotedForTerm:
			c.errorChan <- s.handleSetVotedForTerm(state, c)
		case *getVotedForTerm:
			c.replyChan <- s.handleGetVotedForTerm(state, c)
		case *getRaftState:
			c.replyChan <- s.handleGetRaftState(state, c)
		}
	}
}

func (s *RealRaftStateManager) handleRaftStateManagerStart(state *raftStateManagerState, cmd *raftStateManagerStart) error {
	return nil
}

func (s *RealRaftStateManager) handleRaftStateManagerDestroy(state *raftStateManagerState, cmd *raftStateManagerDestroy) error {
	return nil
}

func (s *RealRaftStateManager) handleRaftStateManagerRecover(state *raftStateManagerState, cmd *raftStateManagerRecover) error {
	return nil
}

func (s *RealRaftStateManager) handleDowngradeToFollower(state *raftStateManagerState, cmd *downgradeToFollower) error {
	return nil
}

func (s *RealRaftStateManager) handleUpgradeToLeader(state *raftStateManagerState, cmd *upgradeToLeader) error {
	return nil
}

func (s *RealRaftStateManager) handleBecomeCandidate(state *raftStateManagerState, cmd *becomeCandidate) error {
	return nil
}

func (s *RealRaftStateManager) handleSetVotedForTerm(state *raftStateManagerState, cmd *setVotedForTerm) error {
	return nil
}

func (s *RealRaftStateManager) handleGetVotedForTerm(state *raftStateManagerState, cmd *getVotedForTerm) *getVotedForTermReply {
	return nil
}

func (s *RealRaftStateManager) handleGetRaftState(state *raftStateManagerState, cmd *getRaftState) *getRaftStateReply {
	return nil
}
