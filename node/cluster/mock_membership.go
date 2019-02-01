package cluster

import (
	"github.com/su225/raft/node/mock"
)

// MockMembershipManager implements the mocked version of
// membership manager which must be used ONLY FOR TESTING PURPOSES
type MockMembershipManager struct {
	Nodes map[string]NodeInfo
}

// NewMockMembershipManager returns a new instance of mock membership manager
// with the given list of nodes for testing purposes
func NewMockMembershipManager(nodes []NodeInfo) *MockMembershipManager {
	mockMembershipMgr := &MockMembershipManager{Nodes: make(map[string]NodeInfo)}
	for _, nodeInfo := range nodes {
		mockMembershipMgr.Nodes[nodeInfo.ID] = nodeInfo
	}
	return mockMembershipMgr
}

// GetDefaultMockMembershipManager returns the mock membership manager with default settings
// This is ONLY FOR TESTING PURPOSES
func GetDefaultMockMembershipManager(currentNodeID string) *MockMembershipManager {
	return NewMockMembershipManager([]NodeInfo{
		NodeInfo{ID: currentNodeID},
		NodeInfo{ID: mock.SampleNodeID1},
		NodeInfo{ID: mock.SampleNodeID2},
	})
}

// AddNode adds a new node to the membership table
func (m *MockMembershipManager) AddNode(nodeID string, info NodeInfo) error {
	m.Nodes[nodeID] = info
	return nil
}

// RemoveNode removes the node from the membership table
func (m *MockMembershipManager) RemoveNode(nodeID string) error {
	delete(m.Nodes, nodeID)
	return nil
}

// GetNode returns the node with given ID
func (m *MockMembershipManager) GetNode(nodeID string) (NodeInfo, error) {
	return m.Nodes[nodeID], nil
}

// GetAllNodes returns all nodes
func (m *MockMembershipManager) GetAllNodes() []NodeInfo {
	nodes := make([]NodeInfo, 0)
	for _, nodeInfo := range m.Nodes {
		nodes = append(nodes, nodeInfo)
	}
	return nodes
}

// Start is a lifecycle method and it does nothing in test
func (m *MockMembershipManager) Start() error {
	return nil
}

// Destroy is a lifecycle method and it does nothing in test
func (m *MockMembershipManager) Destroy() error {
	return nil
}
