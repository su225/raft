#!/usr/bin/ruby

# Copyright (c) 2019 Suchith J N
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.

require 'optparse'
require 'fileutils'
require 'json'

# nodeid_prefix defines the prefix of each node in
# the cluster. Default value is "node"
nodeid_prefix = "node"

# rpcport_start specifies the start of the port range
# for RPC servers. Default value is 6666. If the cluster
# size is 3, then rpc ports will be 6666, 6667 and 6668
rpcport_start = 6666

# apiport_start specifies the start of the port range
# for API servers. Default value is 7777. If the cluster
# size is 3, then api ports will be 7777, 7778, 7779
apiport_start = 7777

# cluster_size specifies the number of nodes in the cluster
# It is recommended to have odd number of nodes
cluster_size = 3

# cluster_dir specifies the root of the directory where
# all cluster data is stored for local testing
cluster_dir = './local-cluster'

# run_cluster runs the cluster locally (uses Tmux)
run_cluster = false

OptionParser.new do |opts|
    opts.banner = "Usage: setup_cluster_dir.rb [options]"

    opts.on('--run-cluster', 'If this flag is on, then cluster is launched') { |v| run_cluster = true }
    opts.on('--nodeid-prefix PREFIX', 'Prefix for generated node ID') { |v| nodeid_prefix = v }
    opts.on('--rpcport-start RPCPORT_START', 'Start of rpc-server port range') { |v| rpcport_start = v.to_i }
    opts.on('--apiport-start APIPORT_START', 'Start of api-server port range') { |v| apiport_start = v.to_i }
    opts.on('--cluster-size CLUSTER_SIZE', 'Size of the cluster') { |v| cluster_size = v.to_i }
    opts.on('--cluster-dir CLUSTER_DIR', 'Root directory for cluster') { |v| cluster_dir = v }
end.parse!

# NodeInfo represents the information on a particular node in the
# raft cluster like its name, url of RPC and API servers. This can
# also be serialized and deserialized to/from JSON
class NodeInfo
    attr_accessor :node_id, :rpc_url, :api_url, :rpc_port, :api_port

    def initialize(node_id, rpc_port, api_port)
        @node_id = node_id
        @rpc_port, @api_port = rpc_port, api_port
        @rpc_url = "localhost:#{rpc_port}"
        @api_url = "localhost:#{api_port}"
    end
    
    # Convert NodeInfo to JSON format
    def to_json(*a)
        {
            node_id: @node_id, 
            rpc_url: @rpc_url, 
            api_url: @api_url
        }.to_json(*a)
    end

    # Parse JSON and try to get NodeInfo
    def self.from_json string
        data = JSON.load string
        self.new data['node_id'], data['rpc_url'], data['api_url'] 
    end
end

# Generate node information in JSON from given parameters
cluster_node_info = (1..cluster_size).map do |serial_no|
    cur_node_id = "#{nodeid_prefix}-#{serial_no}"
    cur_rpc_port = rpcport_start + (serial_no - 1)
    cur_api_port = apiport_start + (serial_no - 1)
    NodeInfo.new cur_node_id, cur_rpc_port, cur_api_port
end

json_cluster_node_info = cluster_node_info.to_json

def get_node_dir_structure(cluster_dir, nodeinfo)
    cur_node_id = nodeinfo.node_id
    node_dir = "#{cluster_dir}/#{cur_node_id}"
    {
        node_dir:           node_dir,
        cluster_config_dir: "#{node_dir}/cluster",
        entry_dir:          "#{node_dir}/data/entry",
        metadata_dir:       "#{node_dir}/data/metadata",
        state_dir:          "#{node_dir}/state",
        snapshot_dir:       "#{node_dir}/snapshot"
    }
end

# All timeouts are milliseconds
base_election_timeout = 3000
heartbeat_interval = 500
rpc_timeout = 1000
api_timeout = 2000
api_fwd_timeout = 1500
max_conn_retry_attempts = 3
command_for_node = []

# Generate directory structure for each node. The entire cluster
# information will be kept in cluster/ directory. It will have a
# directory for each node with node_id being its name. It will have
# subdirectory for cluster configuration, entry, metadata, state
# and snapshot persistence.
cluster_node_info.each do |node_info|
    cur_node_id = node_info.node_id
    puts "Generating directory for #{cur_node_id}"
    node_directories = get_node_dir_structure(cluster_dir, node_info)
    node_directories.each do |dir_purpose, dir_path|
        puts "* Generating #{dir_path}"
        FileUtils.mkdir_p dir_path
    end

    # Write cluster configuration file to the appropriate directory
    # Name of the file should be config.json
    cluster_config_file = "#{node_directories[:cluster_config_dir]}/config.json"
    File.open(cluster_config_file, 'w+') do |f|
        f.write json_cluster_node_info
    end

    command_for_node << """./raft \
        --id=#{node_info.node_id} \
        --api-port=#{node_info.api_port} \
        --rpc-port=#{node_info.rpc_port} \
        --log-entry-path=#{node_directories[:entry_dir]} \
        --log-metadata-path=#{node_directories[:metadata_dir]}/metadata.json \
        --raft-state-path=#{node_directories[:state_dir]}/state.json \
        --join-mode=static \
        --cluster-config-path=#{node_directories[:cluster_config_dir]}/config.json \
        --election-timeout=#{base_election_timeout} \
        --heartbeat=#{heartbeat_interval} \
        --rpc-timeout=#{rpc_timeout} \
        --api-timeout=#{api_timeout} \
        --api-fwd-timeout=#{api_fwd_timeout} \
        --max-conn-retry-attempts=#{max_conn_retry_attempts} \
        --snapshot-path=#{node_directories[:snapshot_dir]} 
    """
    base_election_timeout += 1500
end

if run_cluster
    puts command_for_node[0]
    puts command_for_node[1]
    puts command_for_node[2]

    # TODO: fix this later. Find a better way of doing this
    `tmux new-session "#{command_for_node[0]}" \\\; \
        split-window "#{command_for_node[1]}" \\\; \
        split-window "#{command_for_node[2]}" \\\; \
        select-layout tiled
`
end