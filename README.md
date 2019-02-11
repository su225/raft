# Raft
Implementation of a simple key-value store based on Raft consensus protocol in Go. It supports add, delete and query operations on a particular key. Original Raft consensus protocol paper - [link](https://web.stanford.edu/~ouster/cgi-bin/papers/raft-atc14).

## Start a 3-node cluster locally with default settings
Assuming you have Ruby, Go and Tmux installed in your computer, run the following command to freshly build the source code, create default configuration and launch the nodes in Tmux
```shell
make clean && make build && make setup-local-cluster
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

### Setting up cluster configuration (for local testing only)
If you use the `setup_cluster_dir.rb` it will generate the cluster configuration. Otherwise you have to create a JSON configuration file containing information about all the cluster nodes with their ID, contact information (RPC and API ports) and so on. Here is an example configuration for a 3-node cluster
```json
[
    {"node_id":"node-1","rpc_url":"localhost:6666","api_url":"localhost:7777"},
    {"node_id":"node-2","rpc_url":"localhost:6667","api_url":"localhost:7778"},
    {"node_id":"node-3","rpc_url":"localhost:6668","api_url":"localhost:7779"}
]
```
