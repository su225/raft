# Raft
Implementation of a simple key-value store based on Raft consensus protocol in Go. It supports add, delete and query operations on a particular key. Original Raft consensus protocol paper - [link](https://web.stanford.edu/~ouster/cgi-bin/papers/raft-atc14). It started as a project to learn Go and Raft consensus protocol.

[![Build Status](https://travis-ci.org/su225/raft.svg?branch=master)](https://travis-ci.org/su225/raft)
[![Go Report Card](https://goreportcard.com/badge/github.com/su225/raft)](https://goreportcard.com/report/github.com/su225/raft)
[![Maintainability](https://api.codeclimate.com/v1/badges/e3ba9542f7c1d78b7b35/maintainability)](https://codeclimate.com/github/su225/raft/maintainability)
[![Test Coverage](https://api.codeclimate.com/v1/badges/e3ba9542f7c1d78b7b35/test_coverage)](https://codeclimate.com/github/su225/raft/test_coverage)

## Table of contents
1. [Quick start](#start-a-3-node-cluster-locally-with-default-settings)
2. [Build, test and run](#build-test-and-run)
    1. [Building and running tests](#building-and-running-tests)
    2. [Setting up cluster directory structure](#setting-up-cluster-directory-structure)
    3. [Setting up cluster configuration](#setting-up-cluster-configuration)
    4. [Running an individual node locally](#running-an-individual-node-locally)


## Start a 3-node cluster locally with default settings
Assuming you have Ruby, Go and Tmux installed in your computer, run the following command to freshly build the source code, create default configuration and launch the nodes in Tmux
```shell
make clean && make run-local-cluster
```
If you have docker installed, a cluster of raft Docker containers can be created by running the following command. It creates a 3-node cluster of docker containers connected to a local network called "dockerraft". Port mapping will also be established so that the REST operations and queries described next are still applicable.
```shell
make clean && make run-docker-cluster
```

Once done, you could destroy the local docker cluster with the following command
```shell
make destroy-docker-cluster
```

This command if successful would create a 3-node cluster with nodes _node-1_, _node-2_ and _node-3_. The REST server in each of these nodes will be listening on ports 7777, 7778 and 7779 respectively. You can then use your favorite REST client to make requests to one of the nodes in the cluster and start testing. The example uses curl (you can also use something like Postman)

1. Create a new key-value pair (a -> 1)
   ```shell
    curl -v --header "Content-Type: application/json" \
            --request POST --data '{"k":"a", "v":"1"}' \
            http://localhost:7777/v1/data
   ```
2. Query for the key-value pair just created. This should return the key-value pair we just created in JSON format.
   ```shell
    curl -v http://localhost:7777/v1/data/a
   ```
3. Delete the key "a" from the store
   ```shell
    curl -X DELETE http://localhost:7777/v1/data/a
   ```
4. Now query for the same key and it should return 404 indicating that the key-value pair is not in the store
   ```shell
   curl -v http://localhost:7777/v1/data/a
   ```
**NOTE**:  The response can be 500 for some time in the middle (say during leader election or recovery after a crash in which case the leader may not be known to forward the request to). If that is the case, then try again. If it still persists, it might be a bug (feel free to raise a PR in that case)

---

## Build, test and run

### Building and running tests
In order to build the project you need to have protobuf (protoc) and Go compilers installed. First clone the repository.

1. Build the source code by running the command below. This will also generate some files from protobuf
   ```shell
    make build
   ``` 
2. Run all tests with this command
   ```shell
    make test-all
   ```

### Setting up cluster directory structure

The following instructions assume that you have Go installed. Although you can create your own directory structure, it is recommended to use `scripts/setup_cluster_dir.rb` (needs Ruby) for that purpose. The script accepts some arguments to customize some parts of configuration. It will also output the commands that must be run in separate terminal windows (this can be automated and the work is on the way to do the same)

Arguments accepted by setup_cluster_dir.rb script

| Parameter | Meaning |
|-----------|---------|
| --run-cluster | If true then it not only generates the configuration but also launches it in a TMux session (default value: false) |
| --nodeid-prefix | Prefix for each of the node-IDs that are generated (default value: "node") |
| --rpcport-start | Start of the RPC port range. If start is given as 6666 and the cluster size is 3, then the nodes' RPC ports will be 6666, 6667 and 6668 (default value: 6666) |
| --apiport-start | Start of the API port range. If start is given as 7777 and the cluster size is 3, then the nodes' RPC ports will be 7777, 7778 and 7779 (default value: 7777) |
| --cluster-size | Size of the cluster. It is recommended to set an odd number as the value (default: 3) |
| --cluster-dir | Root of the directory where cluster directory hierarchy is generated (default: ./local-cluster) |

**NOTE**: Make sure that RPC and API port ranges are non-overlapping. At this point in time the script does not validate that. 

### Setting up cluster configuration
If you use the `setup_cluster_dir.rb` it will generate the cluster configuration. Otherwise you have to create a JSON configuration file containing information about all the cluster nodes with their ID, contact information (RPC and API ports) and so on. Here is an example configuration for a 3-node cluster
```json
[
    {"node_id":"node-1","rpc_url":"localhost:6666","api_url":"localhost:7777"},
    {"node_id":"node-2","rpc_url":"localhost:6667","api_url":"localhost:7778"},
    {"node_id":"node-3","rpc_url":"localhost:6668","api_url":"localhost:7779"}
]
```

### Running an individual node locally
Use the command to get the launch commands for each node which can be used to launch each node in a separate terminal
```shell
scripts/cluster_manager.rb --generate --launch --dry-run
```  
You can also add `--docker-mode` flag to the above command get the commands for running the local cluster on docker (make sure that the network to which the containers are attached exists)


| Parameter | Default | Meaning |
|-----------|---------|---------|
| --id      |         | This field is required. It denotes the ID of this node (node_id field in cluster configuration file)
| --api-port | 6666   | This field is required. It denotes the port used by this node for the REST server through which the client can contact this node |
| --rpc-port | 6667   | This field is required. It denotes the port used by this node for RPC server (which listens for protocol messages) |
| --log-entry-path | . | This field is required. It is the directory where write-ahead log entries are stored |
| --log-metadata-path | . | This field is required. It denotes the directory where the metadata of the write-ahead log is stored |
| --raft-state-path | . | This field is required. It denotes the directory where Raft consensus protocol related state is stored |
| --join-mode | cluster-file | It specifies how cluster is formed. The default value "cluster-file" means that a cluster configuration file must be given. In future "k8s" mode will be supported for running on Kubernetes |
| --cluster-config-path | . | This field must be specified only when the joining mode is "cluster-file". Otherwise it will be ignored |
| --election-timeout | 2000 ms | Election timeout in milliseconds. It is recommended to set different election timeouts for each nodes so that all of them won't start election at the same time and cause liveliness issues. This must be much more than heartbeat interval |
| --heartbeat-interval | 500 ms | Heartbeat interval in milliseconds - that is, the time between two heartbeats sent by the leader node. It is recommended to keep this many times less than the election timeout to avoid unnecessary liveliness issues |
| --rpc-timeout | 1000 ms | RPC timeout in milliseconds. The other node is expected to respond to protocol messages within this time |
| --api-timeout | 2000 ms | API timeout in milliseconds. The time after which a REST call is timed out |
| --api-fwd-timeout | 1500 ms | API forward timeout in milliseconds. When the API call hits a non-leader node it is forwarded to the leader. The leader node should then respond within this time. Otherwise the call is considered as failed |
| --max-conn-retry-attempts | 5 | This is optional. It denotes the maximum number of connection retry attempts |
| --snapshot-path | . | This is required. Snapshot path is the directory where snapshot and its metadata are stored. This is needed for log compaction and fast forwarding features |

Here is an example command to launch a raft node. Switch to the directory where Raft's binary is built (it is usually the root directory in the repository)
```shell
    ./raft --id=node-1 \
           --api-port=7777 \
           --rpc-port=6666 \
           --log-entry-path=./local-cluster/node-1/data/entry \
           --log-metadata-path=./local-cluster/node-1/data/metadata/metadata.json \
           --raft-state-path=./local-cluster/node-1/state/state.json \
           --join-mode=cluster-file \
           --cluster-config-path=./local-cluster/node-1/cluster/config.json \
           --election-timeout=3000 \
           --heartbeat=500 \
           --rpc-timeout=1000 \
           --api-timeout=2000 \
           --api-fwd-timeout=1500 \
           --max-conn-retry-attempts=3 \
           --snapshot-path=./local-cluster/node-1/snapshot 
```