package cluster

import (
	"encoding/json"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
)

// Joiner specifies how to join the cluster or
// how node discovery happens in the cluster
type Joiner interface {
	// DiscoverNodes tries to discover nodes in the
	// cluster and if it is successful then it returns
	// a list containing information of nodes discovered
	// If there is some error then it is returned
	DiscoverNodes() ([]NodeInfo, error)
}

var staticFileBasedJoiner = "STATIC-JOIN"

// StaticFileBasedJoiner reads the config file
// containing information about cluster and parses
// the list of members from it. Config must be in JSON
type StaticFileBasedJoiner struct {
	ClusterConfigFile string
}

// NewStaticFileBasedJoiner creates a new instance of
// static file based joiner and returns it. ClusterConfigFile
// is the path (full or relative) to the JSON config file.
func NewStaticFileBasedJoiner(clusterConfigFile string) *StaticFileBasedJoiner {
	return &StaticFileBasedJoiner{clusterConfigFile}
}

// DiscoverNodes reads the config file provided and gets the
// list of nodes in the cluster. Any error in the process is returned
func (joiner *StaticFileBasedJoiner) DiscoverNodes() ([]NodeInfo, error) {
	configBytes, readErr := ioutil.ReadFile(joiner.ClusterConfigFile)
	if readErr != nil {
		return []NodeInfo{}, readErr
	}
	var clusterNodes *[]NodeInfo
	if unmarshalErr := json.Unmarshal(configBytes, &clusterNodes); unmarshalErr != nil {
		return []NodeInfo{}, unmarshalErr
	}
	for _, node := range *clusterNodes {
		logrus.WithFields(logrus.Fields{
			logfield.Component: staticFileBasedJoiner,
			logfield.Event:     "PEER-DISCOVERY",
		}).Debugf("Discovered ID=%s API=%s RPC=%s",
			node.ID, node.APIURL, node.RPCURL)
	}
	return *clusterNodes, nil
}
