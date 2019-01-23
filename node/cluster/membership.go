package cluster

import (
	"github.com/su225/raft/node/common"
)

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

	// MembershipManager should have start and destroy. At
	// the start it should perform peer discovery and get to
	// know other cluster members. On destruction, it should
	// not accept any handles as per the definition
	common.ComponentLifecycle
}

var membershipManager = "MEMBERSHIP-MANAGER"

var membershipManagerIsDestroyedError = &common.ComponentIsDestroyedError{
	ComponentName: membershipManager,
}

var membershipManagerHasNotStartedError = &common.ComponentHasNotStartedError{
	ComponentName: membershipManager,
}

// RealMembershipManager implements operations that are
// needed to maintain membership
type RealMembershipManager struct {
	CurrentNodeInfo NodeInfo
	commandChannel  chan membershipCommand
	Joiner
}

// NewRealMembershipManager creates a new membership manager
func NewRealMembershipManager(
	currentNodeInfo NodeInfo,
	joiner Joiner,
) *RealMembershipManager {
	return &RealMembershipManager{
		CurrentNodeInfo: currentNodeInfo,
		commandChannel:  make(chan membershipCommand),
		Joiner:          joiner,
	}
}

// Start starts the real membership manager component and
// makes this component operational. If it is already destroyed
// then error is returned. If already started, then it is no-op
func (m *RealMembershipManager) Start() error {
	go m.loop()
	feedbackChannel := make(chan error)
	m.commandChannel <- &startMembershipManager{
		currentNodeInfo: m.CurrentNodeInfo,
		errChan:         feedbackChannel,
	}
	return <-feedbackChannel
}

// Destroy destroys the real membership manager component and
// makes this component non-operational. Any operation invoked
// after this call would return error complaining about destruction
func (m *RealMembershipManager) Destroy() error {
	feedbackChannel := make(chan error)
	m.commandChannel <- &destroyMembershipManager{errChan: feedbackChannel}
	return <-feedbackChannel
}

// AddNode adds a new node to the membership table if it does not exist. If it
// exists then an error is returned.
func (m *RealMembershipManager) AddNode(nodeID string, info NodeInfo) error {
	feedbackChannel := make(chan error)
	m.commandChannel <- &addNode{
		nodeInfo: info,
		errChan:  feedbackChannel,
	}
	return <-feedbackChannel
}

// RemoveNode removes the node from the membership table if it exists, else error
func (m *RealMembershipManager) RemoveNode(nodeID string) error {
	feedbackChannel := make(chan error)
	m.commandChannel <- &removeNode{
		nodeID:  nodeID,
		errChan: feedbackChannel,
	}
	return <-feedbackChannel
}

// GetNode gets the node with the given ID from the membership table or returns
// an error complaining that node with given ID does not exist
func (m *RealMembershipManager) GetNode(nodeID string) (NodeInfo, error) {
	nodeInfoChannel := make(chan *getNodeReply)
	m.commandChannel <- &getNode{
		nodeID:    nodeID,
		replyChan: nodeInfoChannel,
	}
	reply := <-nodeInfoChannel
	return reply.nodeInfo, reply.err
}

// GetAllNodes returns all the nodes in the membership table
func (m *RealMembershipManager) GetAllNodes() []NodeInfo {
	nodesChannel := make(chan []NodeInfo)
	m.commandChannel <- &getAllNodes{replyChan: nodesChannel}
	return <-nodesChannel
}

// membershipManagerState is responsible for maintaining the
// state of the component including the table of known members
type membershipManagerState struct {
	isStarted   bool
	isDestroyed bool
	nodeTable   map[string]NodeInfo
}

// loop listens to commands and responds accordingly
func (m *RealMembershipManager) loop() {
	state := &membershipManagerState{
		isStarted:   false,
		isDestroyed: false,
		nodeTable:   make(map[string]NodeInfo),
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
			c.replyChan <- &getNodeReply{nodeInfo: nodeInfo, err: err}
		case *getAllNodes:
			c.replyChan <- m.handleGetAllNodes(state, c)
		}
	}
}

// handleStartMembershipManager starts the membership manager component. This is a lifecycle
// command. It triggers discovery of other nodes and cluster formation. If it is successful
// the NodeTable is populated with discovered nodes
func (m *RealMembershipManager) handleStartMembershipManager(state *membershipManagerState, cmd *startMembershipManager) error {
	if state.isStarted {
		return nil
	}
	if state.isDestroyed {
		return membershipManagerIsDestroyedError
	}
	discoveredNodes, discoveryErr := m.Joiner.DiscoverNodes()
	if discoveryErr != nil {
		return discoveryErr
	}
	state.nodeTable = make(map[string]NodeInfo)
	for _, node := range discoveredNodes {
		state.nodeTable[node.ID] = node
	}
	state.isStarted = true
	return nil
}

// handleDestroyMembershipManager is a lifecycle command which makes the component non-operational.
// If the component is already destroyed then this is a no-op
func (m *RealMembershipManager) handleDestroyMembershipManager(state *membershipManagerState, cmd *destroyMembershipManager) error {
	if state.isDestroyed {
		return nil
	}
	state.nodeTable = nil
	state.isDestroyed = true
	return nil
}

// handleAddNode adds new node to the node table if it is not there yet. If it already exists
// then MemberWithGivenIDAlreadyExistsError is returned with the given nodeID. This operation
// requires component to be operational.
func (m *RealMembershipManager) handleAddNode(state *membershipManagerState, cmd *addNode) error {
	if err := m.checkOperationalStatus(state); err != nil {
		return err
	}
	nodeID := cmd.nodeInfo.ID
	if _, isPresent := state.nodeTable[nodeID]; isPresent {
		return &MemberWithGivenIDAlreadyExistsError{nodeID}
	}
	state.nodeTable[nodeID] = cmd.nodeInfo
	return nil
}

// handleRemoveNode removes the node from the table if it exists. If it does not exist then
// MemberWithGivenIDDoesNotExistError is returned. This operation requires component to be operational
func (m *RealMembershipManager) handleRemoveNode(state *membershipManagerState, cmd *removeNode) error {
	if err := m.checkOperationalStatus(state); err != nil {
		return err
	}
	if _, isPresent := state.nodeTable[cmd.nodeID]; !isPresent {
		return &MemberWithGivenIDDoesNotExistError{cmd.nodeID}
	}
	delete(state.nodeTable, cmd.nodeID)
	return nil
}

// handleGetNode picks up the node information from NodeTable if it exists. If it does not then
// MemberWithGivenIDDoesNotExistError is returned. This operation requires component to be operational
func (m *RealMembershipManager) handleGetNode(state *membershipManagerState, cmd *getNode) (NodeInfo, error) {
	if err := m.checkOperationalStatus(state); err != nil {
		return NodeInfo{}, err
	}
	if nodeInfo, isPresent := state.nodeTable[cmd.nodeID]; !isPresent {
		return NodeInfo{}, &MemberWithGivenIDDoesNotExistError{cmd.nodeID}
	} else {
		return nodeInfo, nil
	}
}

// handleGetAllNodes returns all the nodes known to this node. Component must be operational
// otherwise empty list will be returned
func (m *RealMembershipManager) handleGetAllNodes(state *membershipManagerState, cmd *getAllNodes) []NodeInfo {
	if err := m.checkOperationalStatus(state); err != nil {
		return []NodeInfo{}
	}
	nodeList := make([]NodeInfo, 0)
	for _, node := range state.nodeTable {
		nodeList = append(nodeList, node)
	}
	return nodeList
}

// checkOperationalStatus checks if the component is operational. If the component is destroyed or
// not started yet then appropriate component lifecycle related errors are returned (see errors.go
// in node/common for lifecycle related errors)
func (m *RealMembershipManager) checkOperationalStatus(state *membershipManagerState) error {
	if state.isDestroyed {
		return membershipManagerIsDestroyedError
	}
	if !state.isStarted {
		return membershipManagerHasNotStartedError
	}
	return nil
}
