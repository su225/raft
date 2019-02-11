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