package cluster_test

import (
	"testing"

	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/mock"
)

var defaultCluster = cluster.GetDefaultCluster()
var sampleNodeInfo0 = defaultCluster[0]
var sampleNodeInfo1 = defaultCluster[1]
var sampleNodeInfo2 = defaultCluster[2]
var sampleNodeInfo3 = cluster.NodeInfo{
	ID:     mock.SampleNodeID3,
	APIURL: mock.APIURL3,
	RPCURL: mock.RPCURL3,
}

func TestMembershipManagerAddsNodeIfItDoesNotExistYet(t *testing.T) {
	membershipManager := SetupRealMembershipManagerForTesting([]cluster.NodeInfo{})
	newNodeID, newlyAddedNodeInfo := mock.SampleNodeID3, sampleNodeInfo3
	if opErr := membershipManager.AddNode(newNodeID, newlyAddedNodeInfo); opErr != nil {
		t.FailNow()
	}
	if opErr := membershipManager.AddNode(newNodeID, newlyAddedNodeInfo); opErr == nil {
		t.FailNow()
	}
	if nodeInfo, retrieveErr := membershipManager.GetNode(newNodeID); retrieveErr != nil || nodeInfo != newlyAddedNodeInfo {
		t.FailNow()
	}
}

func TestMembershipManagerRemovesNodeIfItExists(t *testing.T) {
	membershipManager := SetupRealMembershipManagerForTesting(cluster.GetDefaultCluster())
	if opErr := membershipManager.RemoveNode(mock.SampleNodeID3); opErr == nil {
		t.FailNow()
	}
	removedNodeID := sampleNodeInfo1.ID
	if opErr := membershipManager.RemoveNode(removedNodeID); opErr != nil {
		t.FailNow()
	}
	if _, retrieveErr := membershipManager.GetNode(removedNodeID); retrieveErr == nil {
		t.FailNow()
	}
}

func TestMembershipManagerReturnsNodeOnGetNodeIfItExists(t *testing.T) {
	membershipManager := SetupRealMembershipManagerForTesting(cluster.GetDefaultCluster())
	existentNodeID := sampleNodeInfo0.ID
	nonExistentNodeID := sampleNodeInfo3.ID
	if retrievedNodeInfo, retrievalErr := membershipManager.GetNode(existentNodeID); retrievalErr != nil || retrievedNodeInfo != sampleNodeInfo0 {
		t.FailNow()
	}
	if _, retrievalErr := membershipManager.GetNode(nonExistentNodeID); retrievalErr == nil {
		t.FailNow()
	}
}

func TestMembershipManagerReturnsAllDiscoveredNodesOnGetAllNodes(t *testing.T) {
	expectedNodesInCluster := cluster.GetDefaultCluster()
	membershipManager := SetupRealMembershipManagerForTesting(expectedNodesInCluster)
	actualNodes := membershipManager.GetAllNodes()
	for _, expectedNode := range expectedNodesInCluster {
		expectedNodeFound := false
		for _, actualNode := range actualNodes {
			if actualNode == expectedNode {
				expectedNodeFound = true
				break
			}
		}
		if !expectedNodeFound {
			t.FailNow()
		}
	}
}

func SetupRealMembershipManagerForTesting(initNodeList []cluster.NodeInfo) *cluster.RealMembershipManager {
	realMembershipManager := cluster.NewRealMembershipManager(
		sampleNodeInfo0,
		&cluster.MockJoiner{
			ShouldDiscoverySucceed: false,
			DiscoveredNodes:        initNodeList,
		},
	)
	realMembershipManager.Start()
	return realMembershipManager
}
