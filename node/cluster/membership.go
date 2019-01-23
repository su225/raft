package cluster

// NodeInfo represents the information of the node
// in a raft cluster. ID represents its name, RPCURL
// and APIURL represent their RPC and API endpoints
// respectively.
type NodeInfo struct {
	ID     string
	RPCURL string
	APIURL string
}

// MembershipManager defines the operations to be
// supported by the component wishing to manage cluster
// membership in this node
type MembershipManager interface {
	// AddNode adds a new node to the list of nodes in
	// the cluster known to this node. If a node already
	// exists with that name then error is returned
	// informing the same.
	AddNode(nodeID string, info NodeInfo) error
	// RemoveNode removes the node with the given nodeID
	// from the list of cluster members known to this node
	// If the node doesn't exist then an error is returned
	RemoveNode(nodeID string) error
	// GetNode returns the node information of the given
	// nodeID. If it does not exist then error is returned
	GetNode(nodeID string) (NodeInfo, error)
	// GetAllNodes returns the list of nodes known to this
	// node in the cluster. It must include the current node
	GetAllNodes() []NodeInfo
	// DiscoverNodes discovers nodes in the cluster by the
	// mechanism that must be specified by the implementor
	// If there is an error then it is returned
	DiscoverNodes() error
}

// RealMembershipManager implements operations that are
// needed to maintain membership
type RealMembershipManager struct {
	CurrentNodeInfo NodeInfo
	commandChannel  chan membershipCommand
}

// NewRealMembershipManager creates a new membership manager
func NewRealMembershipManager(currentNodeInfo NodeInfo) *RealMembershipManager {
	return &RealMembershipManager{
		CurrentNodeInfo: currentNodeInfo,
		commandChannel:  make(chan membershipCommand),
	}
}

// Start starts the real membership manager component and
// makes this component operational. If it is already destroyed
// then error is returned. If already started, then it is no-op
func (m *RealMembershipManager) Start() error {
	return nil
}

// Destroy destroys the real membership manager component and
// makes this component non-operational. Any operation invoked
// after this call would return error complaining about destruction
func (m *RealMembershipManager) Destroy() error {
	return nil
}

// AddNode adds a new node to the membership table if it does not exist. If it
// exists then an error is returned.
func (m *RealMembershipManager) AddNode(nodeID string, info NodeInfo) error {
	return nil
}

// RemoveNode removes the node from the membership table if it exists, else error
func (m *RealMembershipManager) RemoveNode(nodeID string) error {
	return nil
}

// GetNode gets the node with the given ID from the membership table or returns
// an error complaining that node with given ID does not exist
func (m *RealMembershipManager) GetNode(nodeID string) (NodeInfo, error) {
	return NodeInfo{}, nil
}

// GetAllNodes returns all the nodes in the membership table
func (m *RealMembershipManager) GetAllNodes() []NodeInfo {
	return []NodeInfo{}
}

// DiscoverNodes discovers nodes in the cluster and returns error
// if there was an error in the discovery process.
func (m *RealMembershipManager) DiscoverNodes() error {
	return nil
}

// membershipManagerState is responsible for maintaining the
// state of the component including the table of known members
type membershipManagerState struct {
	isStarted   bool
	isDestroyed bool
	NodeTable   map[string]NodeInfo
}

// loop listens to commands and responds accordingly
func (m *RealMembershipManager) loop() {
	state := &membershipManagerState{
		isStarted:   false,
		isDestroyed: false,
		NodeTable:   make(map[string]NodeInfo),
	}
	for {
		cmd := <-m.commandChannel
		switch c := cmd.(type) {
		case *startMembershipManager:
			c.errChan <- m.handleStartMembershipManager(state, c)
		case *destroyMembershipManager:
			c.errChan <- m.handleDestroyMembershipManager(state, c)
		case *addNode:
			c.errChan <- m.handleAddNode(state, c)
		case *removeNode:
			c.errChan <- m.handleRemoveNode(state, c)
		case *getNode:
			nodeInfo, err := m.handleGetNode(state, c)
			c.replyChan <- &getNodeReply{NodeInfo: nodeInfo, err: err}
		case *getAllNodes:
			c.replyChan <- m.handleGetAllNodes(state, c)
		}
	}
}

func (m *RealMembershipManager) handleStartMembershipManager(state *membershipManagerState, cmd *startMembershipManager) error {
	return nil
}

func (m *RealMembershipManager) handleDestroyMembershipManager(state *membershipManagerState, cmd *destroyMembershipManager) error {
	return nil
}

func (m *RealMembershipManager) handleAddNode(state *membershipManagerState, cmd *addNode) error {
	return nil
}

func (m *RealMembershipManager) handleRemoveNode(state *membershipManagerState, cmd *removeNode) error {
	return nil
}

func (m *RealMembershipManager) handleGetNode(state *membershipManagerState, cmd *getNode) (NodeInfo, error) {
	return NodeInfo{}, nil
}

func (m *RealMembershipManager) handleGetAllNodes(state *membershipManagerState, cmd *getAllNodes) []NodeInfo {
	return []NodeInfo{}
}
