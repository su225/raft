#!/usr/bin/env ruby

# Copyright (c) 2019 Suchith J N

# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:

# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.

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

cluster_dir = 'local-cluster'
cluster_size = 3
cluster_mode = :local

node_prefix = 'node'
rpc_port_range = 6666
api_port_range = 7777

generate = true
launch = false
dry_run = false
docker_destroy = false
docker_network_name = 'dockerraft'

OptionParser.new do |opt|
    opt.on('--cluster-dir CLUSTER_DIR', 'Root directory for cluster-data') { |v| cluster_dir = v }
    opt.on('--cluster-size CLUSTER_SZ', 'Size of the cluster') { |v| cluster_size = v.to_i }

    opt.on('--docker-mode', 'Set the configuration to docker mode') { |v| cluster_mode = :docker }
    opt.on('--docker-destroy', 'Destroy docker containers') { |v| docker_destroy = true }
    opt.on('--docker-network-name DOCKER_NW', 'Name of the docker network') { |v| docker_network_name = v }

    opt.on('--generate', 'If true, then generates directory structure') { |v| generate = true }
    opt.on('--launch', 'If true, then launches the cluster (requires tmux)') { |v| launch = true }
    opt.on('--dry-run', 'If true, then prints the commands, but does not execute it') { |v| dry_run = true }

    opt.on('--node-prefix NODE_PREFIX', 'Prefix of the node ID generated') { |v| node_prefix = v }
    opt.on('--rpc-port RPC_PORT', 'Start of the RPC port range') { |v| rpc_port_range = v.to_i }
    opt.on('--api-port API_PORT', 'Start of the API port range') { |v| api_port_range = v.to_i }
end.parse!

# NodeInfo represents the information about the node
# like name, api and rpc ports
class NodeInfo
    attr_accessor :id, :rpc_port, :api_port, :dir_structure
end

# NodeDirectoryStructure represents the directory structure
# for the given node. Directory structure is required to pass
# arguments to the node while launching or to mount volumes in
# appropriate places if running on docker
class NodeDirectoryStructure
    attr_accessor :log, :state, :snapshot, :cluster
end

# Cluster is a collection to hold all the members of
# local raft cluster. It consists of information about nodes
class Cluster
    def initialize
        @cluster_members = []
    end

    # add_member adds a new member to the cluster
    # It is expected that the item is of type NodeInfo
    def add_member(node_info)
        @cluster_members << node_info
    end

    # generate_config generates cluster configuration for
    # the given cluster members. If the mode is local then
    # generated config has urls with localhost. If it is
    # docker then it has node_id as its name.
    def generate_config_for(mode)
        config_items = []
        @cluster_members.each do |mem|
            # The config item should match the structure NodeInfo
            # in node/cluster/membership.go in order for that one
            # to unmarshal successfully.
            config_item = {node_id: mem.id}
            if :docker.eql? mode
                config_item[:rpc_url] = "#{mem.id}:#{mem.rpc_port}"
                config_item[:api_url] = "#{mem.id}:#{mem.api_port}"
            else
                config_item[:rpc_url] = "localhost:#{mem.rpc_port}"
                config_item[:api_url] = "localhost:#{mem.api_port}"
            end
            config_items << config_item
        end
        config_items
    end
end

# Now generate the NodeInfo for each of the nodes and add them
# to the cluster. Just generate cluster and node information
cluster = Cluster.new
nodes = []
for i in (1..cluster_size) do
    node_id = "#{node_prefix}-#{i}"
    node_dir = "#{cluster_dir}/#{node_id}"

    # Generate directory structure information
    dir_structure = NodeDirectoryStructure.new
    dir_structure.log      = "#{node_dir}/log"
    dir_structure.state    = "#{node_dir}/state"
    dir_structure.snapshot = "#{node_dir}/snapshot"
    dir_structure.cluster  = "#{node_dir}/cluster"

    # Generate the current node information
    node_info = NodeInfo.new
    node_info.id = node_id
    node_info.dir_structure = dir_structure

    # Calculate the ports depending on the mode. In
    # docker mode, ports will be the same since they
    # run in different containers (and by default share
    # no network namespace). In local mode, they cannot
    # share the port and hence it must be different.
    if :docker.eql? cluster_mode
        node_info.rpc_port = rpc_port_range
        node_info.api_port = api_port_range
    else
        node_info.rpc_port = rpc_port_range + (i - 1)
        node_info.api_port = api_port_range + (i - 1)
    end

    nodes << node_info
    cluster.add_member node_info
end

# If the action specified in the command is a generate then
# create the directories and place the cluster configuration
# according to the mode as a JSON file inside the cluster folder
if generate
    FileUtils.rm_rf cluster_dir

    cluster_config = cluster.generate_config_for cluster_mode
    puts "cluster configuration:\n#{JSON.pretty_generate cluster_config}"

    json_config = cluster_config.to_json
    nodes.each do |node|
        puts "generating directory structure for #{node.id}"
        FileUtils.mkdir_p node.dir_structure.log
        FileUtils.mkdir_p node.dir_structure.state
        FileUtils.mkdir_p node.dir_structure.snapshot
        FileUtils.mkdir_p node.dir_structure.cluster

        cluster_config_file = "#{node.dir_structure.cluster}/config.json"
        puts "storing cluster configuration in #{cluster_config_file}"
        File.open(cluster_config_file, 'w+') { |f| f << json_config } 
    end
end

# get_command takes the node_info denoting the node information
# and the cluster_mode (docker or local) and returns the command
# to be run to start an individual node
def get_node_launch_command(node_info, mode, offset, nw_name)
    pwd = `pwd`.gsub(/\n+/,"")
    ds = node_info.dir_structure
    if :docker.eql? mode
        docker_rpc_port = 6666
        docker_api_port = 7777
        "docker run \
            --name #{node_info.id} \
            --hostname #{node_info.id} \
            -e NODE_ID=#{node_info.id} \
            -e ELECTION_TIMEOUT_MILLIS=3000 \
            --mount type=bind,source=#{pwd}/#{ds.log},target=/node/cluster-data/log \
            --mount type=bind,source=#{pwd}/#{ds.state},target=/node/cluster-data/state \
            --mount type=bind,source=#{pwd}/#{ds.snapshot},target=/node/cluster-data/snapshot \
            --mount type=bind,source=#{pwd}/#{ds.cluster},target=/node/cluster-data/cluster,readonly \
            -p #{docker_api_port + offset}:#{docker_api_port} \
            --network=#{nw_name} \
            raft:local \
        ".gsub(/\s+/, " ")
    else
        "./raft 
            --id=#{node_info.id} \
            --api-port=#{node_info.api_port} \
            --rpc-port=#{node_info.rpc_port} \
            --log-entry-path=#{ds.log} \
            --log-metadata-path=#{ds.log}/metadata.json \
            --raft-state-path=#{ds.state}/state.json \
            --election-timeout=3000 \
            --rpc-timeout=2000 \
            --api-timeout=2000 \
            --api-fwd-timeout=1500 \
            --max-conn-retry-attempts=5 \
            --snapshot-path=#{ds.snapshot} \
            --cluster-config-path=#{ds.cluster}/config.json \
        ".gsub(/\s+/, " ")
    end
end


# get_tmux_session_command returns TMUX session command for a
# given list of commands
def get_tmux_session_command(commands)
    tmux_command = ""
    for i in 0...commands.size()
        if i == 0
            tmux_command << "tmux new-session \"#{commands[i]}\" \\; "
        else
            tmux_command << "split-window \"#{commands[i]}\" \\; "
        end
    end
    tmux_command << " select-layout tiled"
    tmux_command.gsub(/\s+/, " ")
end

# If launching is enabled then it launches the commands in a
# tmux session (requires tmux)
if launch
    port_offset = 0

    # If cluster must be setup in docker mode and it is not a dry
    # run then create a new network for the cluster
    if :docker.eql? cluster_mode
        nw_create_command = "docker network create #{docker_network_name}"
        puts "command for docker network: #{nw_create_command}"
        if not dry_run
            `#{nw_create_command}`
        end
    end
    node_launch_commands = []
    nodes.each do |node|
        command = get_node_launch_command(node, cluster_mode, port_offset, docker_network_name)
        node_launch_commands << command
        port_offset = port_offset + 1
        puts "command for #{node.id}:\n#{command}\n\n"
    end
    if not dry_run
        `#{get_tmux_session_command(node_launch_commands)}`
    end
end

# get_docker_node_destroy_command returns the command for forcibly killing
# a raft node running in a Docker container.
def get_docker_node_destroy_command_for(node)
    "docker kill #{node.id}; docker rm #{node.id}"
end

# Docker destroy destroys the docker containers and releases the names
# so that it can be reused some time later.
if :docker.eql? cluster_mode and docker_destroy
    nodes.each do |node|
        command = get_docker_node_destroy_command_for node
        puts "command for #{node.id}:\n#{command}\n\n"
        if not dry_run
            `#{command}`
        end
    end

    # After bringing down the cluster by force, destroy the network in
    # which in which the cluster was setup (only in docker mode)
    if :docker.eql? mode
        nw_destroy_command = "docker network rm #{docker_network_name}"
        puts "command for docker network: #{nw_destroy_command}"
        if not dry_run
            `#{nw_destroy_command}`
        end
    end
end
