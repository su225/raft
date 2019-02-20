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

package k8s

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
)

const k8sJoiner = "K8S-JOINER"
const raftClusterIDLabel = "raft-cluster-id"

// KubernetesJoiner is responsible for cluster formation when
// the Raft cluster is running on Kubernetes. It contacts the
// discovery provider service which fetches all other pods with
// given tag and returns the list of nodes.
type KubernetesJoiner struct {
	ClusterConfigPath string
	RPCPort           uint32
	APIPort           uint32
	kvRegex           *regexp.Regexp
}

const k8sLabelKVRegexp = `^(?P<key>\S+)="(?P<value>\S*)"$`

// NewKubernetesJoiner creates a new instance of Kubernetes joiner with
// the given cluster configuration path, api-port and rpc-port. This makes
// an assumption that API and RPC ports used by all raft-nodes are the same
func NewKubernetesJoiner(
	clusterConfigPath string,
	apiPort, rpcPort uint32,
) *KubernetesJoiner {
	return &KubernetesJoiner{
		ClusterConfigPath: clusterConfigPath,
		APIPort:           apiPort,
		RPCPort:           rpcPort,
		kvRegex:           regexp.MustCompile(k8sLabelKVRegexp),
	}
}

// DiscoverNodes invokes kubernetes discovery mechanism and returns the nodes
// discovered as part of the raft-cluster. If there is an error in the process
// then it is returned. The error can be that the kubernetes discovery provider
// couldn't be contacted or there is some apiserver related error and so on.
func (kj *KubernetesJoiner) DiscoverNodes() ([]cluster.NodeInfo, error) {
	// First, get the address of the kubernetes discovery provider
	providerHostPort, err := kj.getDiscoveryProviderHostPort()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.Component:   k8sJoiner,
			logfield.ErrorReason: err.Error(),
			logfield.Event:       "GET-HOSTPORT",
		}).Errorf("cannot get host:port of the k8s discovery provider")
		return []cluster.NodeInfo{}, err
	}
	// Second, get the cluster label and the current namespace
	raftNamespace, raftLabel, err := kj.getRaftClusterID()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.Component:   k8sJoiner,
			logfield.ErrorReason: err.Error(),
			logfield.Event:       "GET-CLUSTER-ID",
		}).Errorf("cannot get the raft cluster identifier (app label and namespace)")
		return []cluster.NodeInfo{}, err
	}
	// Third, query the discovery provider for the pods with the given
	// label in the given namespace.
	clusterPods, err := kj.getPodsInRaftCluster(providerHostPort, raftNamespace, raftLabel)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.Component:   k8sJoiner,
			logfield.ErrorReason: err.Error(),
			logfield.Event:       "GET-CLUSTER-POD",
		}).Errorf("error while getting pod names in the cluster")
		return []cluster.NodeInfo{}, err
	}
	// Finally, build the list of raft-nodes discovered and return
	discoveredNodes := []cluster.NodeInfo{}
	for _, podName := range clusterPods {
		nodeInfo := cluster.NodeInfo{
			ID:     podName,
			RPCURL: fmt.Sprintf("%s:%d", podName, kj.RPCPort),
			APIURL: fmt.Sprintf("%s:%d", podName, kj.APIPort),
		}
		discoveredNodes = append(discoveredNodes, nodeInfo)
	}
	return discoveredNodes, nil
}

// getDiscoveryProviderHostPort returns the host:port of the kubernetes
// discovery provider service. If it couldn't find it error is returned
func (kj *KubernetesJoiner) getDiscoveryProviderHostPort() (string, error) {
	providerHostPortPath := kj.getClusterConfigFilePath("provider-addr")
	return kj.readFileContentsAsString(providerHostPortPath)
}

// getRaftClusterID returns the namespace and the kubernetes pod label corresponding
// to the cluster identifier. If there is an error while retrieving it, it is returned
func (kj *KubernetesJoiner) getRaftClusterID() (string, string, error) {
	namespace, err := kj.getRaftClusterNamespace()
	if err != nil {
		return "", "", err
	}
	label, err := kj.getRaftClusterLabel()
	if err != nil {
		return "", "", err
	}
	return namespace, label, nil
}

// getRaftClusterNamespace returns the kubernetes namespace in which the
// raft cluster is running
func (kj *KubernetesJoiner) getRaftClusterNamespace() (string, error) {
	nsFilename := kj.getClusterConfigFilePath("k8s-ns")
	return kj.readFileContentsAsString(nsFilename)
}

// getRaftClusterLabel reads the labels on the pod and tries to get the
// "raft-cluster-id" label on it in the same namespace on which this node
// resides. If that label is not present in the current pod then an error
// is returned indicating that the pod should have that label
func (kj *KubernetesJoiner) getRaftClusterLabel() (string, error) {
	labelsPath := kj.getClusterConfigFilePath("labels")
	labelsFile, err := os.Open(labelsPath)
	if err != nil {
		return "", err
	}
	defer labelsFile.Close()

	// Read the file line-by-line and return the
	// key-value pair representing label. Currently
	// Kubernetes stores each label in the format
	// key="value".
	scanner := bufio.NewScanner(labelsFile)
	for scanner.Scan() {
		curLine := scanner.Text()
		logrus.Debugf("current label: %s", curLine)
		labelKey, labelValue, err := kj.parseLabel(curLine)
		if err != nil {
			return "", err
		}
		if labelKey == raftClusterIDLabel {
			return labelValue, nil
		}
	}
	return "", nil
}

func (kj *KubernetesJoiner) parseLabel(labelKV string) (string, string, error) {
	match := kj.kvRegex.FindStringSubmatch(labelKV)
	var result map[string]string
	for i, name := range kj.kvRegex.SubexpNames() {
		if i != 0 && len(name) > 0 {
			result[name] = match[i]
		}
	}
	return result["key"], result["value"], nil
}

// getPodsInRaftCluster returns the pods in the raft cluster. It contacts the kubernetes
// discovery provider through the address provided and queries for all the pods with the
// given label in the given namespace.
func (kj *KubernetesJoiner) getPodsInRaftCluster(providerHostPort, raftNamespace, raftLabel string) ([]string, error) {
	var discErr error
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		escapedRaftNamespace := url.PathEscape(raftNamespace)
		escapedRaftLabel := url.PathEscape(raftLabel)
		providerPath := &url.URL{
			Scheme: "http",
			Host:   providerHostPort,
			Path:   fmt.Sprintf("v1/nodes/%s/%s", escapedRaftNamespace, escapedRaftLabel),
		}
		logrus.WithFields(logrus.Fields{
			logfield.Component: k8sJoiner,
			logfield.Event:     "GET-PEER-REQ",
		}).Debugf("sending GET request to %s", providerPath.String())

		// Send a GET request to the rest endpoint
		httpClient := http.Client{Timeout: time.Duration(10 * time.Second)}
		resp, err := httpClient.Get(providerPath.String())
		if err != nil {
			return []string{}, err
		}
		defer resp.Body.Close()

		// parse the body and retrieve the list of nodes discovered
		var raftNodesDiscovered []string
		discErr = json.NewDecoder(resp.Body).Decode(&raftNodesDiscovered)
		if discErr == nil {
			return raftNodesDiscovered, nil
		}
		// If the query doesn't succeed then log the error, wait for some time
		// and then try again. Just return error if it is the last attempt.
		logrus.WithFields(logrus.Fields{
			logfield.Component:   k8sJoiner,
			logfield.Event:       "DISCOVERY-QUERY",
			logfield.ErrorReason: discErr.Error(),
		}).Errorf("attempt #%d - failed to obtain peer information", attempt)

		if attempt < maxAttempts {
			<-time.After(time.Duration(attempt*2) * time.Second)
		}
	}
	return []string{}, discErr
}

// getClusterConfigFilePath returns the path to the file relative to the cluster
// configuration file.
func (kj *KubernetesJoiner) getClusterConfigFilePath(fileName string) string {
	return filepath.Join(kj.ClusterConfigPath, fileName)
}

// readFileContentsAsString reads the file content and returns the contents as string
// if successful. Otherwise it returns an error
func (kj *KubernetesJoiner) readFileContentsAsString(filePath string) (string, error) {
	readBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(readBytes), nil
}
