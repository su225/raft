package cluster

import (
	"errors"

	"github.com/su225/raft/node/mock"
)

// ErrDiscovery is a dummy discovery related error
var ErrDiscovery = errors.New("error while discovery")

// MockJoiner represents the mock joiner
// which just returns a set of nodes as
// discovered nodes
type MockJoiner struct {
	ShouldDiscoverySucceed bool
	DiscoveredNodes        []NodeInfo
}

// DiscoverNodes returns the expected list of discovered nodes or error
// if discovery is set so that it should not be successful
func (joiner *MockJoiner) DiscoverNodes() ([]NodeInfo, error) {
	if joiner.ShouldDiscoverySucceed {
		return []NodeInfo{}, ErrDiscovery
	}
	return joiner.DiscoveredNodes, nil
}

// GetDefaultMockJoiner returns mock joiner with sensible default
func GetDefaultMockJoiner(discoveryResult bool) *MockJoiner {
	return &MockJoiner{
		ShouldDiscoverySucceed: discoveryResult,
		DiscoveredNodes:        GetDefaultCluster(),
	}
}

// GetDefaultCluster returns the default cluster
func GetDefaultCluster() []NodeInfo {
	return []NodeInfo{
		NodeInfo{ID: mock.SampleNodeID0, APIURL: mock.APIURL0, RPCURL: mock.RPCURL0},
		NodeInfo{ID: mock.SampleNodeID1, APIURL: mock.APIURL1, RPCURL: mock.RPCURL1},
		NodeInfo{ID: mock.SampleNodeID2, APIURL: mock.APIURL2, RPCURL: mock.RPCURL2},
	}
}
