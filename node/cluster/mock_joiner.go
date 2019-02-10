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
