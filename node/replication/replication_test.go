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

package replication

import (
	"errors"
	"testing"

	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/mock"
)

// needs RPCClient, WriteAheadLogManager and MembershipManager

func TestReplicationMustBeSuccessfulIfReplicatedInMajorityAndCommittable(t *testing.T) {
	rpcClient := NewMockReplicationRPCClient(
		[]string{mock.SampleNodeID0, mock.SampleNodeID1},
		[]string{mock.SampleNodeID2},
	)
	writeAheadLogMgr := log.GetDefaultMockWriteAheadLogManager(true)
	membershipMgr := cluster.GetDefaultMockMembershipManager(mock.SampleNodeID0)
	replicationCtrl := NewRealEntryReplicationController(
		rpcClient,
		writeAheadLogMgr,
		mock.SampleNodeID0,
		membershipMgr,
		log.GetDefaultMockSnapshotHandler(),
	)
	replicationCtrl.Start()
	replicationErr := replicationCtrl.ReplicateEntry(
		log.EntryID{TermID: 2, Index: 3},
		&log.DeleteEntry{TermID: 2, Key: "x"},
	)
	if replicationErr != nil {
		t.FailNow()
	}
}

func TestReplicationMustBeFailureIfReplicationFailsInMajority(t *testing.T) {
	rpcClient := NewMockReplicationRPCClient(
		[]string{mock.SampleNodeID0},
		[]string{mock.SampleNodeID1, mock.SampleNodeID2},
	)
	writeAheadLogMgr := log.GetDefaultMockWriteAheadLogManager(true)
	membershipMgr := cluster.GetDefaultMockMembershipManager(mock.SampleNodeID0)
	replicationCtrl := NewRealEntryReplicationController(
		rpcClient,
		writeAheadLogMgr,
		mock.SampleNodeID0,
		membershipMgr,
		log.GetDefaultMockSnapshotHandler(),
	)
	replicationCtrl.Start()
	replicationErr := replicationCtrl.ReplicateEntry(
		log.EntryID{TermID: 2, Index: 3},
		&log.DeleteEntry{TermID: 2, Key: "x"},
	)
	if replicationErr == nil {
		t.FailNow()
	}
}

func TestReplicationMustBeFailureIfCommittingFails(t *testing.T) {
	rpcClient := NewMockReplicationRPCClient(
		[]string{mock.SampleNodeID0, mock.SampleNodeID1},
		[]string{mock.SampleNodeID2},
	)
	writeAheadLogMgr := log.GetDefaultMockWriteAheadLogManager(false)
	membershipMgr := cluster.GetDefaultMockMembershipManager(mock.SampleNodeID0)
	replicationCtrl := NewRealEntryReplicationController(
		rpcClient,
		writeAheadLogMgr,
		mock.SampleNodeID0,
		membershipMgr,
		log.GetDefaultMockSnapshotHandler(),
	)
	replicationCtrl.Start()
	replicationErr := replicationCtrl.ReplicateEntry(
		log.EntryID{TermID: 2, Index: 3},
		&log.DeleteEntry{TermID: 2, Key: "x"},
	)
	if replicationErr == nil {
		t.FailNow()
	}
}

// MockReplicationRPCClient is the client used to replicate log
// entries across machines in tests. **FOR TESTING PURPOSES ONLY**
type MockReplicationRPCClient struct {
	PositiveNodes  []string
	ErroneousNodes []string
}

var errReplication = errors.New("rpc-client : error during replication")

func NewMockReplicationRPCClient(
	positiveNodes []string,
	erroneousNodes []string,
) *MockReplicationRPCClient {
	return &MockReplicationRPCClient{
		PositiveNodes:  positiveNodes,
		ErroneousNodes: erroneousNodes,
	}
}

func (m *MockReplicationRPCClient) RequestVote(curTermID uint64, nodeID string, lastLogEntryID log.EntryID) (bool, error) {
	panic("request vote should not be used for replication")
}

func (m *MockReplicationRPCClient) Heartbeat(curTermID uint64, nodeID string, maxCommittedIndex uint64) (bool, error) {
	panic(`heartbeat should not be used for replication (and propagating 
		   commit-index before responding to client is not necessary as leader
		   election algorithm will make sure that the newly elected leader has
		   the entry replicated so that it can commit it`)
}

func (m *MockReplicationRPCClient) AppendEntry(curTermID uint64, nodeID string, prevEntryID log.EntryID, index uint64, entry log.Entry) (bool, error) {
	for _, errNode := range m.ErroneousNodes {
		if errNode == nodeID {
			return false, errReplication
		}
	}
	return true, nil
}

func (m *MockReplicationRPCClient) InstallSnapshot(curTermID uint64, nodeID string) (uint64, error) {
	return 0, errors.New("replication-installSnapshot mock not implemented")
}

func (m *MockReplicationRPCClient) Start() error {
	return nil
}

func (m *MockReplicationRPCClient) Destroy() error {
	return nil
}
