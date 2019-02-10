# Raft
Implementation of a simple key-value store based on Raft consensus protocol in Go-lang. It supports add, delete and query operations on a particular key. Original Raft consensus protocol paper - [link](https://web.stanford.edu/~ouster/cgi-bin/papers/raft-atc14)

## How to start a 3-node cluster with default settings?
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