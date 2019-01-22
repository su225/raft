package node

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
}

// Validate goes through the configuration and returns a list of
// validation errors found or an empty list otherwise
func Validate(config *Config) []error {
	// TODO: Write validation logic later
	return []error{}
}
