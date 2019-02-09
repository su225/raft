package heartbeat

import (
	"errors"
	"testing"

	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/mock"
	"github.com/su225/raft/node/state"
)

var defaultHeartbeatInterval = int64(10000)

func TestSendHeartbeatShouldReturnTrueIfItCanReachMajorityWithNoDenial(t *testing.T) {
	currentNodeID := mock.SampleNodeID0
	acceptors := []string{currentNodeID, mock.SampleNodeID1}
	deniers := []string{}
	unresponsive := []string{mock.SampleNodeID2}
	heartbeatController := NewRealLeaderHeartbeatController(
		currentNodeID,
		defaultHeartbeatInterval,
		newMockHeartbeatClient(acceptors, deniers, unresponsive),
		state.GetDefaultMockRaftStateManager(currentNodeID, true),
		log.GetDefaultMockWriteAheadLogManager(true),
		cluster.GetDefaultMockMembershipManager(currentNodeID),
	)
	acceptedAsLeader, _ := heartbeatController.SendHeartbeat()
	if !acceptedAsLeader {
		t.FailNow()
	}
}

func TestSendHeartbeatShouldReturnFalseIfItCannotReachMajority(t *testing.T) {
	currentNodeID := mock.SampleNodeID0
	acceptors := []string{currentNodeID}
	deniers := []string{}
	unresponsive := []string{mock.SampleNodeID1, mock.SampleNodeID2}
	mockStateManager := state.GetDefaultMockRaftStateManager(currentNodeID, true)
	heartbeatController := NewRealLeaderHeartbeatController(
		currentNodeID,
		defaultHeartbeatInterval,
		newMockHeartbeatClient(acceptors, deniers, unresponsive),
		mockStateManager,
		log.GetDefaultMockWriteAheadLogManager(true),
		cluster.GetDefaultMockMembershipManager(currentNodeID),
	)
	acceptedAsLeader, _ := heartbeatController.SendHeartbeat()
	if acceptedAsLeader {
		t.FailNow()
	}
	if mockStateManager.DowngradeToFollowerCount == 0 {
		t.FailNow()
	}
}

func TestSendHeartbeatShouldReturnFalseIfAtLeastOneNodeRejectsAuthority(t *testing.T) {
	currentNodeID := mock.SampleNodeID0
	acceptors := []string{currentNodeID, mock.SampleNodeID1}
	deniers := []string{mock.SampleNodeID2}
	unresponsive := []string{}
	mockStateManager := state.GetDefaultMockRaftStateManager(currentNodeID, true)
	heartbeatController := NewRealLeaderHeartbeatController(
		currentNodeID,
		defaultHeartbeatInterval,
		newMockHeartbeatClient(acceptors, deniers, unresponsive),
		mockStateManager,
		log.GetDefaultMockWriteAheadLogManager(true),
		cluster.GetDefaultMockMembershipManager(currentNodeID),
	)
	acceptedAsLeader, _ := heartbeatController.SendHeartbeat()
	if acceptedAsLeader {
		t.FailNow()
	}
	if mockStateManager.DowngradeToFollowerCount == 0 {
		t.FailNow()
	}
}

// mockHeartbeatClient represents the mocked RPC Client
// which is used to send heartbeats. This is for TESTING purposes
type mockHeartbeatClient struct {
	AcceptAuthority []string
	DenyAuthority   []string
	Unresponsive    []string
}

var errHeartbeatClient = errors.New("heartbeat client error")

// NewMockHeartbeatClient creates a new instance of mock
// RPC client to be used for requesting votes only
func newMockHeartbeatClient(acceptors, deniers, unresponsive []string) *mockHeartbeatClient {
	return &mockHeartbeatClient{
		AcceptAuthority: acceptors,
		DenyAuthority:   deniers,
		Unresponsive:    unresponsive,
	}
}

// RequestVote checks the nodeID against the list of vote granters and grants
// votes if the name of the node is in the list of granters
func (m *mockHeartbeatClient) RequestVote(curTermID uint64, nodeID string, lastLogEntryID log.EntryID) (bool, error) {
	panic("cannot use request-vote as part of heartbeating")
}

// Heartbeat panics since the election algorithm should not be heartbeating
// to other nodes as part of election algorithm
func (m *mockHeartbeatClient) Heartbeat(curTermID uint64, nodeID string, maxCommittedIndex uint64) (bool, error) {
	for _, id := range m.AcceptAuthority {
		if nodeID == id {
			return true, nil
		}
	}
	for _, id := range m.DenyAuthority {
		if nodeID == id {
			return false, nil
		}
	}
	for _, id := range m.Unresponsive {
		if nodeID == id {
			return true, errHeartbeatClient
		}
	}
	return true, nil
}

// AppendEntry panics since the election algorithm should not be calling appendEntry
// as part of its algorithm
func (m *mockHeartbeatClient) AppendEntry(curTermID uint64, nodeID string, prevEntryID log.EntryID, index uint64, entry log.Entry) (bool, error) {
	panic("cannot use appendEntry as part of heartbeating")
}

func (m *mockHeartbeatClient) InstallSnapshot(curTermID uint64, nodeID string) (uint64, error) {
	panic("cannot use installSnapshot as part of heartbeating")
}

// Start is a no-op in testing scenarios
func (m *mockHeartbeatClient) Start() error {
	return nil
}

// Destroy is a no-op in testing scenarios
func (m *mockHeartbeatClient) Destroy() error {
	return nil
}
