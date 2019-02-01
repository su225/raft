package mock

import (
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/state"
)

// GetDefaultMockRaftStateManager returns the mock state manager with default settings
// This is ONLY FOR TESTING PURPOSES
func GetDefaultMockRaftStateManager(nodeID string, succeeds bool) *state.MockRaftStateManager {
	return state.NewMockRaftStateManager(succeeds, state.RaftState{
		RaftDurableState: state.RaftDurableState{
			CurrentNodeID: nodeID,
			CurrentTermID: 2,
			VotedFor:      SampleNodeID1,
		},
		CurrentRole:   state.RoleFollower,
		CurrentLeader: SampleNodeID2,
	})
}

// GetDefaultMockWriteAheadLogManager returns the mock write-ahead log manager with default settings
// This is ONLY FOR TESTING PURPOSES
func GetDefaultMockWriteAheadLogManager(succeeds bool) *log.MockWriteAheadLogManager {
	return log.NewMockWriteAheadLogManager(succeeds,
		log.WriteAheadLogMetadata{
			TailEntryID: log.EntryID{
				TermID: 2,
				Index:  3,
			},
			MaxCommittedIndex: 2,
		},
		map[uint64]log.Entry{
			0: &log.SentinelEntry{},
			1: &log.UpsertEntry{TermID: 1, Key: "a", Value: "1"},
			2: &log.UpsertEntry{TermID: 2, Key: "b", Value: "2"},
			3: &log.DeleteEntry{TermID: 2, Key: "a"},
		},
	)
}

// GetDefaultMockMembershipManager returns the mock membership manager with default settings
// This is ONLY FOR TESTING PURPOSES
func GetDefaultMockMembershipManager(currentNodeID string) *cluster.MockMembershipManager {
	return cluster.NewMockMembershipManager([]cluster.NodeInfo{
		cluster.NodeInfo{ID: currentNodeID},
		cluster.NodeInfo{ID: SampleNodeID1},
		cluster.NodeInfo{ID: SampleNodeID2},
	})
}
