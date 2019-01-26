package rpc

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var raftProtocolClient = "RPC-CLIENT"

var raftProtocolClientIsNotStartedError = &common.ComponentHasNotStartedError{ComponentName: raftProtocolClient}
var raftProtocolClientIsDestroyedError = &common.ComponentIsDestroyedError{ComponentName: raftProtocolClient}

// RealRaftProtobufClient is responsible for handling all protocol
// related outgoing messages.
type RealRaftProtobufClient struct {
	clientMutex sync.RWMutex
	// MembershipManager is used to look up the address of the
	// remote node specified. It must be provided.
	cluster.MembershipManager

	// CurrentNodeID is the identifier of the current node
	// in the Raft cluster
	CurrentNodeID string

	// MaxConnectionRetryAttempts specify the maximum number of
	// times the client tries to connect to the remote node before
	// giving up.
	MaxConnectionRetryAttempts uint32

	// RPCTimeoutInMillis specifies RPC-timeout in milliseconds
	RPCTimeoutInMillis uint64

	// commandChannels represent the channel per client
	commandChannels map[string]chan protocolClientCommand

	// ComponentLifecycle indicates that this is a component and
	// has lifecycle events - start and destroy
	common.ComponentLifecycle
}

// NewRealRaftProtobufClient creates a new instance of real raft
// protocol client. This component is responsible for handling all
// outgoing communication.
func NewRealRaftProtobufClient(
	membershipManager cluster.MembershipManager,
	currentNodeID string,
	maxConnRetryAttempts uint32,
	rpcTimeoutInMillis uint64,
) *RealRaftProtobufClient {
	return &RealRaftProtobufClient{
		clientMutex:                sync.RWMutex{},
		MembershipManager:          membershipManager,
		CurrentNodeID:              currentNodeID,
		commandChannels:            make(map[string]chan protocolClientCommand),
		MaxConnectionRetryAttempts: maxConnRetryAttempts,
		RPCTimeoutInMillis:         rpcTimeoutInMillis,
	}
}

// Start starts the operation of the component unless it is already
// destroyed. This operation is idempotent. If there is any error
// during start-up this fails and returns error
func (rpcc *RealRaftProtobufClient) Start() error {
	for _, node := range rpcc.MembershipManager.GetAllNodes() {
		if node.ID == rpcc.CurrentNodeID {
			continue
		}
		if startErr := rpcc.startClient(node); startErr != nil {
			return startErr
		}
	}
	return nil
}

// startClient starts the client for the given node ID. At this point it does
// not try to connect to the remote node. It is done lazily.
func (rpcc *RealRaftProtobufClient) startClient(node cluster.NodeInfo) error {
	rpcc.putNode(node)
	go rpcc.loop(node)
	errChan := make(chan error)
	rpcc.commandChannels[node.ID] <- &startClient{errChan: errChan}
	return <-errChan
}

// Destroy makes the component non-operational. No operation can be
// invoked on this component after it is destroyed.
func (rpcc *RealRaftProtobufClient) Destroy() error {
	rpcc.clientMutex.Lock()
	defer rpcc.clientMutex.Unlock()
	var destructionErr error
	for _, nodeCmdChan := range rpcc.commandChannels {
		errChan := make(chan error)
		nodeCmdChan <- &destroyClient{errChan: errChan}
		if destroyErr := <-errChan; destroyErr != nil {
			destructionErr = destroyErr
		}
	}
	return destructionErr
}

// RequestVote requests the remote node. The function returns true if the remote
// node grants vote, false otherwise. If there is an error then it is returned.
func (rpcc *RealRaftProtobufClient) RequestVote(curTermID uint64, nodeID string, termID uint64, lastLogEntryID log.EntryID) (bool, error) {
	if cmdChan, isPresent := rpcc.getNode(nodeID); isPresent {
		replyChan := make(chan *clientRequestVoteReply)
		cmdChan <- &clientRequestVote{
			currentTermID:  curTermID,
			lastLogEntryID: lastLogEntryID,
			replyChan:      replyChan,
		}
		reply := <-replyChan
		return reply.voteGranted, reply.votingErr
	}
	return false, &cluster.MemberWithGivenIDDoesNotExistError{NodeID: nodeID}
}

// Heartbeat sends heartbeat to the given node in the cluster. The function returns true
// if the remote node accepts the authority of this node in the cluster (as leader), false
// otherwise. If there is an error during this operation, it is returned
func (rpcc *RealRaftProtobufClient) Heartbeat(curTermID uint64, nodeID string, termID, maxCommittedIndex uint64) (bool, error) {
	if cmdChan, isPresent := rpcc.getNode(nodeID); isPresent {
		replyChan := make(chan *clientHeartbeatReply)
		cmdChan <- &clientHeartbeat{
			currentTermID:     curTermID,
			maxCommittedIndex: maxCommittedIndex,
			replyChan:         replyChan,
		}
		reply := <-replyChan
		return reply.acceptedAsLeader, reply.heartbeatErr
	}
	return true, nil
}

// AppendEntry tries to append the given entry to the log. The function returns true if the
// given entry is successfully appended to the log or false otherwise. If there is an error
// during operation then it is returned.
func (rpcc *RealRaftProtobufClient) AppendEntry(curTermID uint64, nodeID string, prevEntryID log.EntryID, index uint64, entry log.Entry) (bool, error) {
	if cmdChan, isPresent := rpcc.getNode(nodeID); isPresent {
		replyChan := make(chan *clientAppendEntryReply)
		cmdChan <- &clientAppendEntry{
			currentTermID: curTermID,
			index:         index,
			entry:         entry,
			prevEntryID:   prevEntryID,
			replyChan:     replyChan,
		}
		reply := <-replyChan
		return reply.entryAppended, reply.appendErr
	}
	return false, nil
}

// putNode creates an entry for the given node and opens a channel. This
// operation is safe in concurrent environments
func (rpcc *RealRaftProtobufClient) putNode(info cluster.NodeInfo) {
	rpcc.clientMutex.Lock()
	defer rpcc.clientMutex.Unlock()
	rpcc.commandChannels[info.ID] = make(chan protocolClientCommand)
}

// getNode gets the channel for the given ID if it exists or returns nil.
// This operation is safe in concurrent environments
func (rpcc *RealRaftProtobufClient) getNode(id string) (chan protocolClientCommand, bool) {
	rpcc.clientMutex.RLock()
	defer rpcc.clientMutex.RUnlock()
	if cmdChan, present := rpcc.commandChannels[id]; present {
		return cmdChan, present
	}
	return nil, false
}

type raftProtocolClientState struct {
	isStarted, isDestroyed bool
	remoteNodeInfo         cluster.NodeInfo
	*grpc.ClientConn
}

// loop handles commands related to outgoing communication for a given node. The connection
// obtained can be cached for better performance
func (rpcc *RealRaftProtobufClient) loop(info cluster.NodeInfo) {
	state := &raftProtocolClientState{
		isStarted:      false,
		isDestroyed:    false,
		remoteNodeInfo: info,
		ClientConn:     nil,
	}
	curCommandChan := rpcc.commandChannels[info.ID]
	for {
		cmd := <-curCommandChan
		switch c := cmd.(type) {
		case *startClient:
			c.errChan <- rpcc.handleStartClient(state, c)
		case *destroyClient:
			c.errChan <- rpcc.handleDestroyClient(state, c)
		case *clientRequestVote:
			c.replyChan <- rpcc.handleRequestVote(state, c)
		case *clientHeartbeat:
			c.replyChan <- rpcc.handleHeartbeat(state, c)
		case *clientAppendEntry:
			c.replyChan <- rpcc.handleAppendEntry(state, c)
		case *clientReconnect:
			c.errChan <- rpcc.handleReconnect(state, c)
		}
	}
}

// handleStartClient makes the component operational for the remote node. If it is
// already destroyed then it cannot be made operational and error is returned informing the same.
func (rpcc *RealRaftProtobufClient) handleStartClient(state *raftProtocolClientState, cmd *startClient) error {
	if state.isDestroyed {
		return raftProtocolClientIsDestroyedError
	}
	if state.isStarted {
		return nil
	}
	state.isStarted = true
	return nil
}

// handleDestroyClient makes the component non-operational for the given remote node. It
// also closes the connection to the remote node if there was one
func (rpcc *RealRaftProtobufClient) handleDestroyClient(state *raftProtocolClientState, cmd *destroyClient) error {
	if state.isDestroyed {
		return nil
	}
	state.isDestroyed = true
	if state.ClientConn != nil {
		return state.ClientConn.Close()
	}
	return nil
}

// handleRequestVote sends vote request to the remote node and returns true or false depending on
// the voting decision of the remote node. If there is any error then it is returned
func (rpcc *RealRaftProtobufClient) handleRequestVote(state *raftProtocolClientState, cmd *clientRequestVote) *clientRequestVoteReply {
	if statusErr := rpcc.checkOperationalStatus(state); statusErr != nil {
		return &clientRequestVoteReply{
			voteGranted: false,
			votingErr:   statusErr,
		}
	}
	rpcClient, clientConnErr := rpcc.getRPCClient(state)
	if clientConnErr != nil {
		return &clientRequestVoteReply{
			voteGranted: false,
			votingErr:   clientConnErr,
		}
	}
	rpcContext, rpcCancelFunc := rpcc.getRPCCallContext()
	defer rpcCancelFunc()
	grantVoteRequest := &raftpb.GrantVoteRequest{
		SenderInfo: &raftpb.NodeInfo{
			NodeId: rpcc.CurrentNodeID,
			TermId: cmd.currentTermID,
		},
		LastEntryMetadata: &raftpb.OpEntryMetadata{
			TermId: cmd.lastLogEntryID.TermID,
			Index:  cmd.lastLogEntryID.Index,
		},
	}
	rpcResponse, rpcErr := rpcClient.RequestVote(rpcContext, grantVoteRequest)
	if rpcErr != nil {
		return &clientRequestVoteReply{
			voteGranted: false,
			votingErr:   rpcErr,
		}
	}
	return &clientRequestVoteReply{
		voteGranted: rpcResponse.GetVoteGranted(),
		votingErr:   nil,
	}
}

// handleHeartbeat sends the heartbeat to the remote node and returns whether the remote node accepts
// the authority of this node as the leader. If the remote leader does not accept the authority then
// this node should step down as leader and make way for the election for the next term
func (rpcc *RealRaftProtobufClient) handleHeartbeat(state *raftProtocolClientState, cmd *clientHeartbeat) *clientHeartbeatReply {
	if statusErr := rpcc.checkOperationalStatus(state); statusErr != nil {
		return &clientHeartbeatReply{
			acceptedAsLeader: true,
			heartbeatErr:     statusErr,
		}
	}
	rpcClient, clientConnErr := rpcc.getRPCClient(state)
	if clientConnErr != nil {
		return &clientHeartbeatReply{
			acceptedAsLeader: true,
			heartbeatErr:     clientConnErr,
		}
	}
	rpcContext, rpcCancelFunc := rpcc.getRPCCallContext()
	defer rpcCancelFunc()
	heartbeatRequest := &raftpb.HeartbeatRequest{
		SenderInfo: &raftpb.NodeInfo{
			NodeId: rpcc.CurrentNodeID,
			TermId: cmd.currentTermID,
		},
		LatestCommitIndex: cmd.maxCommittedIndex,
	}
	rpcResponse, rpcErr := rpcClient.Heartbeat(rpcContext, heartbeatRequest)
	if rpcErr != nil {
		return &clientHeartbeatReply{
			acceptedAsLeader: true,
			heartbeatErr:     rpcErr,
		}
	}
	return &clientHeartbeatReply{
		acceptedAsLeader: rpcResponse.GetAcceptAsLeader(),
		heartbeatErr:     nil,
	}
}

// handleAppendEntry sends the request to the remote node to append an entry to its log at a particular index
// with the given previous index and termID (to maintain order ot log entries). If the remote node succeeds in
// replicating, then it returns true else false. If there is any error in between then it is returned.
func (rpcc *RealRaftProtobufClient) handleAppendEntry(state *raftProtocolClientState, cmd *clientAppendEntry) *clientAppendEntryReply {
	if statusErr := rpcc.checkOperationalStatus(state); statusErr != nil {
		return &clientAppendEntryReply{
			entryAppended: false,
			appendErr:     statusErr,
		}
	}
	rpcClient, clientConnErr := rpcc.getRPCClient(state)
	if clientConnErr != nil {
		return &clientAppendEntryReply{
			entryAppended: false,
			appendErr:     clientConnErr,
		}
	}
	rpcContext, rpcCancelFunc := rpcc.getRPCCallContext()
	defer rpcCancelFunc()
	appendEntryRequest := &raftpb.AppendEntryRequest{
		SenderInfo: &raftpb.NodeInfo{
			NodeId: rpcc.CurrentNodeID,
			TermId: cmd.currentTermID,
		},
		CurEntry:   ConvertEntryToProtobuf(cmd.entry),
		PrevTermId: cmd.prevEntryID.TermID,
		CurEntryMetadata: &raftpb.OpEntryMetadata{
			TermId: cmd.entry.GetTermID(),
			Index:  cmd.index,
		},
	}
	rpcResponse, rpcErr := rpcClient.AppendEntry(rpcContext, appendEntryRequest)
	if rpcErr != nil {
		return &clientAppendEntryReply{
			entryAppended: false,
			appendErr:     rpcErr,
		}
	}
	return &clientAppendEntryReply{
		entryAppended: rpcResponse.GetAppended(),
		appendErr:     nil,
	}
}

// handleReconnect attempts to reobtain the connection to the remote node
func (rpcc *RealRaftProtobufClient) handleReconnect(state *raftProtocolClientState, cmd *clientReconnect) error {
	if statusErr := rpcc.checkOperationalStatus(state); statusErr != nil {
		return statusErr
	}
	logrus.WithFields(logrus.Fields{
		logfield.Component: raftProtocolClient,
		logfield.Event:     "RECONNECT",
	}).Infof("attempting to reconnect to %s", state.remoteNodeInfo.ID)
	state.ClientConn = nil
	rpcc.getConnection(state)
	return nil
}

// getRPCClient returns a new client RPC stub.
func (rpcc *RealRaftProtobufClient) getRPCClient(state *raftProtocolClientState) (raftpb.RaftProtocolClient, error) {
	conn, err := rpcc.getConnection(state)
	if err != nil {
		return nil, err
	}
	return raftpb.NewRaftProtocolClient(conn), nil
}

// getConnection tries to obtain connection to the remote node if it does not exist and caches it. If the connection
// is already there in the cache then it is returned
func (rpcc *RealRaftProtobufClient) getConnection(state *raftProtocolClientState) (*grpc.ClientConn, error) {
	if state.ClientConn != nil {
		return state.ClientConn, nil
	}
	var (
		connErr  error
		grpcConn *grpc.ClientConn
	)
	remoteNodeAddress := state.remoteNodeInfo.RPCURL
	for attempt := uint32(1); attempt <= rpcc.MaxConnectionRetryAttempts; attempt++ {
		// NOTE: WithInsecure, as the name says is insecure. TODO: Change this and
		// add certificates to enable secure communication
		grpcConn, connErr = grpc.Dial(remoteNodeAddress,
			grpc.WithInsecure(),
			grpc.WithKeepaliveParams(keepalive.ClientParameters{PermitWithoutStream: true}),
		)
		if connErr != nil {
			logrus.WithFields(logrus.Fields{
				logfield.Component: raftProtocolClient,
				logfield.Event:     "GET-CONN-DIAL",
			}).Warnf("attempt #%d to obtain connection to %s failed",
				attempt, state.remoteNodeInfo.ID)
			<-time.After(time.Duration(attempt) * time.Second)
		} else {
			break
		}
	}
	if connErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: connErr.Error(),
			logfield.Component:   raftProtocolClient,
			logfield.Event:       "GET-CONN",
		}).Errorf("cannot obtain connection to %s", state.remoteNodeInfo.ID)
		return nil, connErr
	}
	state.ClientConn = grpcConn
	return state.ClientConn, nil
}

// getRPCCallContext returns the context for the RPC call containing cancellation,
// deadline and other information needed to make RPC call.
func (rpcc *RealRaftProtobufClient) getRPCCallContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), time.Duration(rpcc.RPCTimeoutInMillis)*time.Millisecond)
}

// tryReconnect sends clientReconnect message to the sub-component handling outgoing
// communication for the given nodeID.
func (rpcc *RealRaftProtobufClient) tryReconnect(nodeID string) error {
	nodeCmdChannel, isPresent := rpcc.getNode(nodeID)
	if !isPresent {
		return &cluster.MemberWithGivenIDDoesNotExistError{NodeID: nodeID}
	}
	errChan := make(chan error)
	nodeCmdChannel <- &clientReconnect{errChan: errChan}
	return <-errChan
}

// checkOperationalStatus checks if the component is operational for the given remote node and returns appropriate
// error messages if it is not started or is already destroyed.
func (rpcc *RealRaftProtobufClient) checkOperationalStatus(state *raftProtocolClientState) error {
	if state.isDestroyed {
		return raftProtocolClientIsDestroyedError
	}
	if !state.isStarted {
		return raftProtocolClientIsNotStartedError
	}
	return nil
}
