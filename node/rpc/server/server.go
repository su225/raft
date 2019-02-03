// Copyright (c) 2019 Suchith J N

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package server

import (
	"context"
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	. "github.com/su225/raft/logfield"
	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/election"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/rpc"
	"github.com/su225/raft/node/state"
	"github.com/su225/raft/pb"
	"google.golang.org/grpc"
)

const rpcServer = "RPC-SERVER"

var rpcServerNotStartedError = &common.ComponentHasNotStartedError{ComponentName: rpcServer}
var rpcServerIsDestroyedError = &common.ComponentIsDestroyedError{ComponentName: rpcServer}

// RealRaftProtobufServer is responsible for handling all
// incoming communication to this node. In other words, all
// incoming messages must come here
type RealRaftProtobufServer struct {
	// RPCPort is the port on which the protocol server
	// listens to. This must be different from API Port which
	// is the port API-server listens to
	RPCPort uint32

	// Voter is responsible for making voting decisions
	// on behalf of the node
	*state.Voter

	// RaftStateManager is responsible for managing state related to raft
	state.RaftStateManager

	// LeaderElectionManager is needed to reset the election timeout
	election.LeaderElectionManager

	// commandChannel is used to provide various commands. This
	// is where operations actually happen
	commandChannel chan protocolServerCommand

	// The component has lifecycle - it can be started and destroyed
	common.ComponentLifecycle
}

// NewRealRaftProtobufServer creates a new instance of RealRaftProtocolServer
// But this method does not start the server. In other words, server won't be
// listening to incoming messages at the given port.
func NewRealRaftProtobufServer(
	rpcPort uint32,
	voter *state.Voter,
	raftStateMgr state.RaftStateManager,
	electionMgr election.LeaderElectionManager,
) *RealRaftProtobufServer {
	return &RealRaftProtobufServer{
		RPCPort:               rpcPort,
		Voter:                 voter,
		RaftStateManager:      raftStateMgr,
		LeaderElectionManager: electionMgr,
		commandChannel:        make(chan protocolServerCommand),
	}
}

// Start brings up the server so that it listens at the port specified by
// RPCPort and starts accepting connections and incoming protobuf messages.
func (rpcs *RealRaftProtobufServer) Start() error {
	go rpcs.loop()
	startupErrChan := make(chan error)
	rpcs.commandChannel <- &startServer{
		rpcPort: rpcs.RPCPort,
		errChan: startupErrChan,
	}
	return <-startupErrChan
}

// Destroy brings down the server so that other nodes can no longer
// connect to this node. The component becomes non-operational and
// this function is irreversible.
func (rpcs *RealRaftProtobufServer) Destroy() error {
	destroyErrChan := make(chan error)
	rpcs.commandChannel <- &destroyServer{
		errChan: destroyErrChan,
	}
	return <-destroyErrChan
}

// RequestVote decides if this node should grant vote to the remote node for the
// term based on certain criteria like the length of the log. If there is any
// failure in the component or network then it must be assumed that the node is
// not in a position to grant vote and is taken to be 'false'
func (rpcs *RealRaftProtobufServer) RequestVote(context context.Context, request *raftpb.GrantVoteRequest) (*raftpb.GrantVoteReply, error) {
	requestVoteResChan := make(chan *requestVoteReply)
	rpcs.commandChannel <- &requestVoteRequest{
		GrantVoteRequest: request,
		resChan:          requestVoteResChan,
	}
	result := <-requestVoteResChan
	return result.GrantVoteReply, result.requestVoteError
}

// AppendEntry checks if it is possible to append entry to the log. If it is then
// it appends entry to the log at the given index in the given term. If there are
// any failures then the same will be returned and the client must retry.
func (rpcs *RealRaftProtobufServer) AppendEntry(context context.Context, request *raftpb.AppendEntryRequest) (*raftpb.AppendEntryReply, error) {
	appendEntryResChan := make(chan *appendEntryReply)
	rpcs.commandChannel <- &appendEntryRequest{
		AppendEntryRequest: request,
		resChan:            appendEntryResChan,
	}
	result := <-appendEntryResChan
	return result.AppendEntryReply, result.appendEntryError
}

// Heartbeat tries to update maximum committed index obtained from the leader. If
// a heartbeat is received when the node is not a follower then it is not considered
// based on certain conditions arount TermID. In reply, it tells the node sending
// heartbeat if it accepts the node as the leader
func (rpcs *RealRaftProtobufServer) Heartbeat(context context.Context, request *raftpb.HeartbeatRequest) (*raftpb.HeartbeatReply, error) {
	heartbeatResChan := make(chan *heartbeatReply)
	rpcs.commandChannel <- &heartbeatRequest{
		HeartbeatRequest: request,
		resChan:          heartbeatResChan,
	}
	result := <-heartbeatResChan
	return result.HeartbeatReply, result.heartbeatError
}

// InstallSnapshot tries to obtain snapshot from the leader and applies it so that
// the write-ahead log can be fast-forwarded and older entries can be cleaned up. The
// snapshot transfer must be atomic. In other words, on failure, snapshot being
// transferred must be discarded.
func (rpcs *RealRaftProtobufServer) InstallSnapshot(raftpb.RaftProtocol_InstallSnapshotServer) error {
	return nil
}

type raftProtocolServerState struct {
	isStarted   bool
	isDestroyed bool
	server      *grpc.Server
}

// loop listens to various commands and returns results if necessary. If the component
// is destroyed then all commands are no-op
func (rpcs *RealRaftProtobufServer) loop() {
	state := &raftProtocolServerState{
		isStarted:   false,
		isDestroyed: false,
		server:      nil,
	}
	for {
		cmd := <-rpcs.commandChannel
		switch serverCmd := cmd.(type) {
		case *startServer:
			serverCmd.errChan <- rpcs.handleStartServer(state, serverCmd)
		case *destroyServer:
			serverCmd.errChan <- rpcs.handleDestroyServer(state, serverCmd)
		case *requestVoteRequest:
			serverCmd.resChan <- rpcs.handleRequestVote(state, serverCmd)
		case *appendEntryRequest:
			serverCmd.resChan <- rpcs.handleAppendEntry(state, serverCmd)
		case *heartbeatRequest:
			serverCmd.resChan <- rpcs.handleHeartbeat(state, serverCmd)
		}
	}
}

// handleStartServer starts the server if it not destroyed or already started. If there is an error while
// starting then it is returned. Otherwise nil is returned. This operation is idempotent.
func (rpcs *RealRaftProtobufServer) handleStartServer(state *raftProtocolServerState, cmd *startServer) error {
	if state.isDestroyed {
		return rpcServerIsDestroyedError
	}
	if state.isStarted {
		return nil
	}

	// Setup listener on which the server listens
	rpcServerAddress := fmt.Sprintf(":%d", cmd.rpcPort)
	rpcListener, rpcListenerErr := net.Listen("tcp", rpcServerAddress)
	if rpcListenerErr != nil {
		return rpcListenerErr
	}

	// start GRPC server and register. TODO: There is a race condition
	// between Serve and destroy where GracefulStop can be called before
	// Serve is complete. Figure out how to signal the start of the server
	state.server = grpc.NewServer()
	raftpb.RegisterRaftProtocolServer(state.server, rpcs)
	go state.server.Serve(rpcListener)

	state.isStarted = true
	logrus.WithFields(logrus.Fields{
		Component: rpcServer,
		Event:     "START",
	}).Infof("starting RPC server at %s", rpcServerAddress)
	return nil
}

// handleDestroyServer gracefully shuts down the RPC server if it is not already destroyed.
// If it is already destroyed then this is a no-op. This operation is idempotent
func (rpcs *RealRaftProtobufServer) handleDestroyServer(state *raftProtocolServerState, cmd *destroyServer) error {
	if state.isDestroyed {
		return nil
	}
	state.server.GracefulStop()
	state.isDestroyed = true
	logrus.WithFields(logrus.Fields{
		Component: rpcServer,
		Event:     "DESTROY",
	}).Infof("destroyed RPC server")
	return nil
}

// handleRequestVote handles request vote message where the remote node asks for vote for a given term. Here
// the decision to grant or deny vote is made. If there is an error then it is communicated and vote is denied
// If the node is in higher term and its log is at least as long as the current log then vote is granted if and
// only if the node has not already voted in this term
func (rpcs *RealRaftProtobufServer) handleRequestVote(state *raftProtocolServerState, cmd *requestVoteRequest) *requestVoteReply {
	remoteNodeID := cmd.GrantVoteRequest.GetSenderInfo().GetNodeId()
	remoteTermID := cmd.GrantVoteRequest.GetSenderInfo().GetTermId()
	remoteTailIDPb := cmd.GrantVoteRequest.GetLastEntryMetadata()
	remoteTailID := log.EntryID{
		TermID: remoteTailIDPb.TermId,
		Index:  remoteTailIDPb.Index,
	}

	logrus.WithFields(logrus.Fields{
		Component: rpcServer,
		Event:     "RECV-REQUEST-VOTE",
	}).Debugf("received request-vote message from (%s,%d)", remoteNodeID, remoteTermID)

	voteGranted, votingErr := rpcs.Voter.DecideVote(remoteNodeID, remoteTermID, remoteTailID)
	currentRaftState := rpcs.GetRaftState()
	rpcs.LeaderElectionManager.ResetTimeout()
	if voteGranted {
		logrus.WithFields(logrus.Fields{
			Component: rpcServer,
			Event:     "GRANT-VOTE",
		}).Debugf("granted vote in term #%d to %s", remoteTermID, remoteNodeID)
	}
	return &requestVoteReply{
		requestVoteError: votingErr,
		GrantVoteReply: &raftpb.GrantVoteReply{
			SenderInfo:  rpcs.getProtobufHeader(&currentRaftState),
			VoteGranted: voteGranted,
		},
	}
}

// handleAppendEntry handles append entry request from mostly cluster leader. If the operation can be performed
// preserving all safety properties specified in Raft paper then it will be successful, otherwise it will fail.
// If the remote node has lower term ID then the entry is not accepted. If the remote node doesn't have the same
// term ID as the current node then the request is ignored. If the remote node is in higher term ID its heartbeat
// should force this node to become its follower anyways.
func (rpcs *RealRaftProtobufServer) handleAppendEntry(state *raftProtocolServerState, cmd *appendEntryRequest) *appendEntryReply {
	remoteNodeID := cmd.AppendEntryRequest.GetSenderInfo().GetNodeId()
	remoteTermID := cmd.AppendEntryRequest.GetSenderInfo().GetTermId()
	currentRaftState := rpcs.RaftStateManager.GetRaftState()
	currentTermID := currentRaftState.CurrentTermID

	reply := &appendEntryReply{
		appendEntryError: nil,
		AppendEntryReply: &raftpb.AppendEntryReply{
			SenderInfo: rpcs.getProtobufHeader(&currentRaftState),
			Appended:   false,
		},
	}

	logrus.WithFields(logrus.Fields{
		Component: rpcServer,
		Event:     "RECV-APPEND-ENTRY",
	}).Debugf("received append-entry message from (%d,%d)", remoteNodeID, remoteTermID)

	if currentTermID > remoteTermID {
		return reply
	}

	// If termID is less than or equal to the remote Term ID then first downgrade to the
	// status of follower accepting the remote node as leader. If there is any error in
	// this process then don't write entry to the log and return error
	if currentTermID <= remoteTermID {
		if opErr := rpcs.RaftStateManager.DowngradeToFollower(remoteNodeID, remoteTermID); opErr != nil {
			logrus.WithFields(logrus.Fields{
				ErrorReason: opErr.Error(),
				Component:   rpcServer,
				Event:       "DOWNGRADE-TO-FOLLOWER",
			}).Errorf("error while trying to step down as follower")
			reply.Appended = false
			reply.appendEntryError = opErr
			return reply
		}
		prevEntryID := log.EntryID{
			TermID: cmd.AppendEntryRequest.GetPrevTermId(),
			Index:  cmd.AppendEntryRequest.GetCurEntryMetadata().Index - 1,
		}
		curIndex := cmd.AppendEntryRequest.GetCurEntryMetadata().GetIndex()
		entry := rpc.ConvertProtobufToEntry(cmd.AppendEntryRequest.GetCurEntry())
		_, writeErr := rpcs.WriteAheadLogManager.WriteEntryAfter(prevEntryID, curIndex, entry)
		if writeErr != nil {
			logrus.WithFields(logrus.Fields{
				ErrorReason: writeErr.Error(),
				Component:   rpcServer,
				Event:       "WRITE-ENTRY",
			}).Errorf("error while writing entry #%d", curIndex)
			reply.appendEntryError = writeErr
			reply.Appended = false
			return reply
		} else {
			reply.Appended = true
		}
	}
	return reply
}

// handleHeartbeat handles heartbeat from the remote node which claims to be the leader. If the remote node is
// in a higher term then this node should accept it as the leader. If the remote node is in the lower term then
// it should not accept it as the leader. If the remote node has the same term as the current node and the node
// is a candidate or a follower without a leader then leader is updated. This operation might update the maximum
// committed index in the write-ahead log if this node accepts the remote as leader.
func (rpcs *RealRaftProtobufServer) handleHeartbeat(serverState *raftProtocolServerState, cmd *heartbeatRequest) *heartbeatReply {
	remoteNodeID := cmd.HeartbeatRequest.GetSenderInfo().GetNodeId()
	remoteTermID := cmd.HeartbeatRequest.GetSenderInfo().GetTermId()
	currentRaftState := rpcs.RaftStateManager.GetRaftState()
	reply := &heartbeatReply{
		heartbeatError: nil,
		HeartbeatReply: &raftpb.HeartbeatReply{
			SenderInfo:     rpcs.getProtobufHeader(&currentRaftState),
			AcceptAsLeader: true,
		},
	}
	if currentRaftState.CurrentTermID > remoteTermID {
		reply.HeartbeatReply.AcceptAsLeader = false
		return reply
	}

	// Try to accept authority of the remote node as leader.
	// If there is an issue then complain and reject authority
	if (currentRaftState.CurrentRole != state.RoleLeader && currentRaftState.CurrentTermID == remoteTermID) ||
		(currentRaftState.CurrentTermID < remoteTermID) {
		if opErr := rpcs.RaftStateManager.DowngradeToFollower(remoteNodeID, remoteTermID); opErr != nil {
			logrus.WithFields(logrus.Fields{
				ErrorReason: opErr.Error(),
				Component:   rpcServer,
				Event:       "DOWNGRADE-TO-FOLLOWER",
			}).Errorf("error while trying to step down as follower")
			reply.heartbeatError = opErr
			reply.AcceptAsLeader = true
			return reply
		}
	}
	if resetErr := rpcs.resetElectionTimeout(); resetErr != nil {
		reply.heartbeatError = resetErr
		return reply
	}
	return reply
}

func (rpcs *RealRaftProtobufServer) resetElectionTimeout() error {
	if resetErr := rpcs.LeaderElectionManager.ResetTimeout(); resetErr != nil {
		logrus.WithFields(logrus.Fields{
			ErrorReason: resetErr.Error(),
			Component:   rpcServer,
			Event:       "RESET-TIMEOUT",
		}).Errorf("error while election timeout reset")
		return resetErr
	}
	return nil
}

func (rpcs *RealRaftProtobufServer) getProtobufHeader(curState *state.RaftState) *raftpb.NodeInfo {
	return &raftpb.NodeInfo{
		NodeId: curState.CurrentNodeID,
		TermId: curState.CurrentTermID,
	}
}
