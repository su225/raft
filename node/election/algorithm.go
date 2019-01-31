package election

import (
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/rpc"
	"github.com/su225/raft/node/state"
)

// LeaderElectionAlgorithm is used to define the details
// of the leader election algorithm/strategy.
type LeaderElectionAlgorithm interface {
	// ConductElection actually runs the leader election algorithm
	// and returns true if this node is elected as the leader or
	// false otherwise. If there is an error then it is returned.
	// But note that the node can still be a leader with some errors.
	// (Implementation can be fault-tolearant)
	ConductElection() (electedAsLeader bool, electionErr error)
}

var raftElectionAlgo = "RAFT-ELECTION-ALGO"

// RaftLeaderElectionAlgorithm is the leader election algorithm
// as defined in the original Raft paper.
type RaftLeaderElectionAlgorithm struct {
	// CurrentNodeID represents the identity of the
	// current node in the Raft cluster
	CurrentNodeID string

	// RaftProtobufClient is needed to request remote
	// node for votes.
	rpc.RaftProtobufClient

	// RaftStateManager is needed to transition between
	// various roles and get state related information
	state.RaftStateManager

	// WriteAheadLogManager is needed to get the write-ahead
	// log metadata like tail entry ID
	log.WriteAheadLogManager

	// MembershipManager is needed to get the list of all
	// nodes known to this node in the cluster. The node
	// sends voting request to all other nodes in the cluster
	cluster.MembershipManager
}

// NewRaftLeaderElectionAlgorithm creates a new instance of
// raft leader election algorithm(strategy) and returns the same
func NewRaftLeaderElectionAlgorithm(
	currentNodeID string,
	raftClient rpc.RaftProtobufClient,
	stateMgr state.RaftStateManager,
	writeAheadLogMgr log.WriteAheadLogManager,
	membershipMgr cluster.MembershipManager,
) *RaftLeaderElectionAlgorithm {
	return &RaftLeaderElectionAlgorithm{
		CurrentNodeID:        currentNodeID,
		RaftProtobufClient:   raftClient,
		RaftStateManager:     stateMgr,
		WriteAheadLogManager: writeAheadLogMgr,
		MembershipManager:    membershipMgr,
	}
}

// ConductElection executes the raft leader election protocol. It switches
// to the next term and becomes candidate if it is a follower or candidate.
// It votes itself and asks remote nodes for votes. If it obtains majority
// then it declares itself as the leader of the cluster and tries to
// establish authority over other nodes (through heartbeat)
func (ea *RaftLeaderElectionAlgorithm) ConductElection() (bool, error) {
	beforeElectionState := ea.RaftStateManager.GetRaftState()
	candidateTermID := beforeElectionState.CurrentTermID + 1
	if opErr := ea.BecomeCandidate(candidateTermID); opErr != nil {
		return false, opErr
	}
	beforeElectionWALogMetadata, _ := ea.WriteAheadLogManager.GetMetadata()
	candidateTailEntryID := beforeElectionWALogMetadata.TailEntryID

	clusterNodes := ea.MembershipManager.GetAllNodes()
	remoteNodeCount, majorityCount := len(clusterNodes)-1, int32(len(clusterNodes)/2+1)

	var voterWaitGroup sync.WaitGroup
	voterWaitGroup.Add(remoteNodeCount)

	// totalVotesObtained starts from 1, not 0 because the node
	// will vote itself when it becomes the candidate.
	totalVotesObtained := int32(1)

	// Now iterate over each node and skip the current node and request for
	// their votes. If there is any error then it is as good as not granting vote
	for _, nodeInfo := range clusterNodes {
		if nodeInfo.ID == ea.CurrentNodeID {
			continue
		}
		go func(ni cluster.NodeInfo) {
			defer voterWaitGroup.Done()
			voteGranted, votingErr := ea.RaftProtobufClient.RequestVote(candidateTermID, ni.ID, candidateTailEntryID)
			if votingErr != nil {
				logrus.WithFields(logrus.Fields{
					logfield.ErrorReason: votingErr.Error(),
					logfield.Component:   raftElectionAlgo,
					logfield.Event:       "REQUEST-VOTE",
				}).Errorf("error while requesting vote to %s", ni.ID)
				return
			}
			if voteGranted {
				logrus.WithFields(logrus.Fields{
					logfield.Component: raftElectionAlgo,
					logfield.Event:     "REQUEST-VOTE",
				}).Debugf("obtained vote for term %d from %s", candidateTermID, ni.ID)
				atomic.AddInt32(&totalVotesObtained, 1)
			}
		}(nodeInfo)
	}
	voterWaitGroup.Wait()

	logrus.WithFields(logrus.Fields{
		logfield.Component: raftElectionAlgo,
		logfield.Event:     "VOTE-COUNTING",
	}).Debugf("obtained voted = %d, required = %d",
		totalVotesObtained, majorityCount)

	// If majority is obtained, then try to upgrade to leader. If this node fails to
	// upgrade as leader then it fails to establish authority and there will be another
	// leader election. This does not violate safety, but some liveliness issues.
	if totalVotesObtained >= majorityCount {
		logrus.WithFields(logrus.Fields{
			logfield.Component: raftElectionAlgo,
			logfield.Event:     "ELECTION-RESULT",
		}).Debugf("majority obtained. Try to upgrade to leader for term %d", candidateTermID)

		if opErr := ea.RaftStateManager.UpgradeToLeader(candidateTermID); opErr != nil {
			logrus.WithFields(logrus.Fields{
				logfield.Component:   raftElectionAlgo,
				logfield.Event:       "UPGRADE-TO-LEADER",
				logfield.ErrorReason: opErr.Error(),
			}).Debugf("failed to upgrade to leadership in spite of majority")
			return false, opErr
		}
		return true, nil
	}
	return false, nil
}
