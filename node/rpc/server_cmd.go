package rpc

import (
	"github.com/su225/raft/pb"
)

type startServer struct {
	protocolServerCommand
	RPCPort uint32
	errChan chan error
}

type destroyServer struct {
	protocolServerCommand
	errChan chan error
}

type requestVoteRequest struct {
	protocolServerCommand
	*raftpb.GrantVoteRequest
	resChan chan requestVoteReply
}

type requestVoteReply struct {
	*raftpb.GrantVoteReply
	RequestVoteError error
}

type appendEntryRequest struct {
	protocolServerCommand
	*raftpb.AppendEntryRequest
	resChan chan appendEntryReply
}

type appendEntryReply struct {
	*raftpb.AppendEntryReply
	AppendEntryError error
}

type heartbeatRequest struct {
	protocolServerCommand
	*raftpb.HeartbeatRequest
	resChan chan heartbeatReply
}

type heartbeatReply struct {
	*raftpb.HeartbeatReply
	HeartbeatError error
}
