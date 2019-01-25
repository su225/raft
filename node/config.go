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

package node

const (
	// StaticFileJoinMode is the default mode and is useful
	// running cluster locally. It looks for a cluster
	// configuration file containing the list of nodes
	// in the cluster. It expects JSON format
	StaticFileJoinMode = "static"

	// KubernetesJoinMode is enabled when this is run
	// in Kubernetes.
	KubernetesJoinMode = "k8s"
)

// Config represents the configuration passed
// to the raft node like NodeID, API and RPC ports
type Config struct {
	NodeID  string `json:"node_id"`
	APIPort uint32 `json:"api_port"`
	RPCPort uint32 `json:"rpc_port"`

	WriteAheadLogEntryPath    string `json:"write_ahead_log_entry_path"`
	WriteAheadLogMetadataPath string `json:"write_ahead_log_metadata_path"`
	RaftStatePath             string `json:"raft_state_path"`

	JoinMode          string `json:"cluster_mode"`
	ClusterConfigPath string `json:"cluster_config_path"`

	ElectionTimeoutInMillis   int64 `json:"election_timeout_in_millis"`
	HeartbeatIntervalInMillis int64 `json:"heartbeat_interval_in_millis"`
	RPCTimeoutInMillis        int64 `json:"rpc_timeout_in_millis"`

	MaxConnectionRetryAttempts uint32 `json:"max_conn_retry_attempts"`
}

// Validate goes through the configuration and returns a list of
// validation errors found or an empty list otherwise
func Validate(config *Config) []error {
	// TODO: Write validation logic later
	return []error{}
}
