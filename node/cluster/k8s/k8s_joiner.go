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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
)

const k8sJoiner = "K8S-JOINER"
const raftClusterIDLabel = "raft-cluster/id"
const raftClusterSizeLabel = "raft-cluster/size"
const raftClusterSvcNameLabel = "raft-cluster/svcName"

// KubernetesJoiner is responsible for cluster formation when
// the Raft cluster is running on Kubernetes. It contacts the
// discovery provider service which fetches all other pods with
// given tag and returns the list of nodes.
type KubernetesJoiner struct {
	ClusterConfigPath string
	NodeID            string
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
	nodeID string,
	apiPort, rpcPort uint32,
) *KubernetesJoiner {
	return &KubernetesJoiner{
		ClusterConfigPath: clusterConfigPath,
		NodeID:            nodeID,
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
	discoveredNodes := []cluster.NodeInfo{}
	podLabels, err := kj.getRaftPodLabels()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: err.Error(),
			logfield.Component:   k8sJoiner,
			logfield.Event:       "GET-LABELS",
		}).Errorf("failed to get labels on the pod")
		return discoveredNodes, err
	}
	clusterID, present := podLabels[raftClusterIDLabel]
	if !present {
		logrus.WithFields(logrus.Fields{
			logfield.Component: k8sJoiner,
			logfield.Event:     "GET-RAFT-CLUSTER-ID",
		}).Warnf("couldn't find label: %s", raftClusterIDLabel)
		return discoveredNodes, &LabelNotPresentError{RequiredLabel: raftClusterIDLabel}
	}
	clusterSize, present := podLabels[raftClusterSizeLabel]
	if !present {
		logrus.WithFields(logrus.Fields{
			logfield.Component: k8sJoiner,
			logfield.Event:     "GET-RAFT-CLUSTER-SIZE",
		}).Warnf("couldn't find label: %s", raftClusterSizeLabel)
		return discoveredNodes, &LabelNotPresentError{RequiredLabel: raftClusterSizeLabel}
	}
	clusterSvcName, present := podLabels[raftClusterSvcNameLabel]
	if !present {
		logrus.WithFields(logrus.Fields{
			logfield.Component: k8sJoiner,
			logfield.Event:     "GET-RAFT-CLUSTER-SVC-NAME",
		}).Warnf("couldn't find label: %s", raftClusterSvcNameLabel)
		return discoveredNodes, &LabelNotPresentError{RequiredLabel: clusterSvcName}
	}
	var intClusterSize int
	_, err = fmt.Sscanf(clusterSize, "%d", &intClusterSize)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: err.Error(),
			logfield.Component:   k8sJoiner,
			logfield.Event:       "PARSE-RAFT-CLUSTER-SIZE",
		}).Errorf("error while parsing %s as integer", clusterSize)
		return discoveredNodes, err
	}
	ns, err := kj.getRaftClusterNamespace()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: err.Error(),
			logfield.Component:   k8sJoiner,
			logfield.Event:       "GET-POD-NAMESPACE",
		}).Errorf("error while getting pod namespace")
		return discoveredNodes, err
	}
	discoveredNodes = kj.getRaftClusterNodeInfo(clusterID, clusterSvcName, ns, intClusterSize)
	for _, node := range discoveredNodes {
		logrus.WithFields(logrus.Fields{
			logfield.Component: k8sJoiner,
			logfield.Event:     "DISCOVERY-DONE",
		}).Infof("discovered peer: nodeID=%s, rpcURL=%s, apiURL=%s",
			node.ID, node.RPCURL, node.APIURL)
	}
	return discoveredNodes, nil
}

// getRaftClusterNodeInfo builds the cluster configuration given cluster ID, namespace
// and the cluster size. Name of other peers can be derived since they have a predictable
// pattern (it must be deployed as StatefulSet)
func (kj *KubernetesJoiner) getRaftClusterNodeInfo(clusterID, clusterSvcName, ns string, clusterSize int) []cluster.NodeInfo {
	nodes := make([]cluster.NodeInfo, clusterSize)
	for i := 0; i < clusterSize; i++ {
		nodeID := fmt.Sprintf("%s-%d", clusterID, i)
		nodes[i] = cluster.NodeInfo{
			ID:     nodeID,
			RPCURL: fmt.Sprintf("%s.%s:%d", nodeID, clusterSvcName, kj.RPCPort),
			APIURL: fmt.Sprintf("%s.%s:%d", nodeID, clusterSvcName, kj.APIPort),
		}
	}
	return nodes
}

// getRaftClusterNamespace returns the kubernetes namespace in which the
// raft cluster is running
func (kj *KubernetesJoiner) getRaftClusterNamespace() (string, error) {
	nsFilename := kj.getClusterConfigFilePath("k8s-ns")
	return kj.readFileContentsAsString(nsFilename)
}

// getRaftPodLabels reads the labels on the pod and tries to get the
// "raft-cluster-id" label on it in the same namespace on which this node
// resides. If that label is not present in the current pod then an error
// is returned indicating that the pod should have that label
func (kj *KubernetesJoiner) getRaftPodLabels() (map[string]string, error) {
	labelsPath := kj.getClusterConfigFilePath("labels")
	labelsFile, err := os.Open(labelsPath)
	if err != nil {
		return map[string]string{}, err
	}
	defer labelsFile.Close()

	// Read the file line-by-line and return the
	// key-value pair representing label. Currently
	// Kubernetes stores each label in the format
	// key="value".
	scanner := bufio.NewScanner(labelsFile)
	labels := make(map[string]string)
	for scanner.Scan() {
		curLine := scanner.Text()
		logrus.Debugf("current label: %s", curLine)
		labelKey, labelValue, err := kj.parseLabel(curLine)
		if err != nil {
			return labels, err
		}
		labels[labelKey] = labelValue
	}
	return labels, nil
}

func (kj *KubernetesJoiner) parseLabel(labelKV string) (string, string, error) {
	match := kj.kvRegex.FindStringSubmatch(labelKV)
	result := make(map[string]string)
	for i, name := range kj.kvRegex.SubexpNames() {
		if i != 0 && len(name) > 0 {
			result[name] = match[i]
		}
	}
	return result["key"], result["value"], nil
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
