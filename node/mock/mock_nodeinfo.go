package mock

import "github.com/su225/raft/node/cluster"

var (
	SampleNodeID0 = "sample-node-0"
	SampleNodeID1 = "sample-node-1"
	SampleNodeID2 = "sample-node-2"
	SampleNodeID3 = "sample-node-3"

	RPCURL0 = ":6666"
	RPCURL1 = ":6667"
	RPCURL2 = ":6668"
	RPCURL3 = ":6669"

	APIURL0 = ":7777"
	APIURL1 = ":7778"
	APIURL2 = ":7779"
	APIURL3 = ":7780"
)

var (
	SampleNodeInfo0 = cluster.NodeInfo{ID: SampleNodeID0, RPCURL: RPCURL0, APIURL: APIURL0}
	SampleNodeInfo1 = cluster.NodeInfo{ID: SampleNodeID1, RPCURL: RPCURL1, APIURL: APIURL1}
	SampleNodeInfo2 = cluster.NodeInfo{ID: SampleNodeID2, RPCURL: RPCURL2, APIURL: APIURL2}
	SampleNodeInfo3 = cluster.NodeInfo{ID: SampleNodeID3, RPCURL: RPCURL3, APIURL: APIURL3}
)
