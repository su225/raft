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
