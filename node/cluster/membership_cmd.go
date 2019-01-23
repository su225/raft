package cluster

// membershipCommand represents the command to
// membership manager
type membershipCommand interface {
	IsMembershipCommand() bool
}

type startMembershipManager struct {
	membershipCommand
	CurrentNodeInfo NodeInfo
	errChan         chan error
}

type destroyMembershipManager struct {
	membershipCommand
	errChan chan error
}

type addNode struct {
	membershipCommand
	NodeInfo
	errChan chan error
}

type removeNode struct {
	membershipCommand
	nodeID  string
	errChan chan error
}

type getNode struct {
	membershipCommand
	nodeID    string
	replyChan chan *getNodeReply
}

type getNodeReply struct {
	NodeInfo
	err error
}

type getAllNodes struct {
	membershipCommand
	replyChan chan []NodeInfo
}
