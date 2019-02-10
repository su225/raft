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
