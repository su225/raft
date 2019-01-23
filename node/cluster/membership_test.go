package cluster_test

import (
	"testing"

	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/mock"
)

func TestMembershipManagerAddsNodeIfItDoesNotExistYet(t *testing.T) {
	membershipManager := SetupRealMembershipManagerForTesting([]cluster.NodeInfo{})
	newNodeID, newlyAddedNodeInfo := mock.SampleNodeID3, mock.SampleNodeInfo3
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
	membershipManager := SetupRealMembershipManagerForTesting([]cluster.NodeInfo{
		mock.SampleNodeInfo0,
		mock.SampleNodeInfo1,
		mock.SampleNodeInfo2,
	})
	if opErr := membershipManager.RemoveNode(mock.SampleNodeID3); opErr == nil {
		t.FailNow()
	}
	removedNodeID := mock.SampleNodeInfo1.ID
	if opErr := membershipManager.RemoveNode(removedNodeID); opErr != nil {
		t.FailNow()
	}
	if _, retrieveErr := membershipManager.GetNode(removedNodeID); retrieveErr == nil {
		t.FailNow()
	}
}

func TestMembershipManagerReturnsNodeOnGetNodeIfItExists(t *testing.T) {
	membershipManager := SetupRealMembershipManagerForTesting([]cluster.NodeInfo{
		mock.SampleNodeInfo0,
		mock.SampleNodeInfo1,
		mock.SampleNodeInfo2,
	})
	existentNodeID := mock.SampleNodeInfo0.ID
	nonExistentNodeID := mock.SampleNodeInfo3.ID
	if retrievedNodeInfo, retrievalErr := membershipManager.GetNode(existentNodeID); retrievalErr != nil || retrievedNodeInfo != mock.SampleNodeInfo0 {
		t.FailNow()
	}
	if _, retrievalErr := membershipManager.GetNode(nonExistentNodeID); retrievalErr == nil {
		t.FailNow()
	}
}

func TestMembershipManagerReturnsAllDiscoveredNodesOnGetAllNodes(t *testing.T) {
	expectedNodesInCluster := []cluster.NodeInfo{
		mock.SampleNodeInfo0,
		mock.SampleNodeInfo1,
		mock.SampleNodeInfo2,
	}
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
		mock.SampleNodeInfo0,
		&mock.MockJoiner{
			ShouldDiscoverySucceed: false,
			DiscoveredNodes:        initNodeList,
		},
	)
	realMembershipManager.Start()
	return realMembershipManager
}
