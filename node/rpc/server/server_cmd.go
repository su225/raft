package server

import (
	"github.com/su225/raft/pb"
)

type protocolServerCommand interface {
	isProtocolServerCommand() bool
}

type startServer struct {
	protocolServerCommand
	rpcPort uint32
	errChan chan error
}

type destroyServer struct {
	protocolServerCommand
	errChan chan error
}

type requestVoteRequest struct {
	protocolServerCommand
	*raftpb.GrantVoteRequest
	resChan chan *requestVoteReply
}

type requestVoteReply struct {
	*raftpb.GrantVoteReply
	requestVoteError error
}

type appendEntryRequest struct {
	protocolServerCommand
	*raftpb.AppendEntryRequest
	resChan chan *appendEntryReply
}

type appendEntryReply struct {
	*raftpb.AppendEntryReply
	appendEntryError error
}

type heartbeatRequest struct {
	protocolServerCommand
	*raftpb.HeartbeatRequest
	resChan chan *heartbeatReply
}

type heartbeatReply struct {
	*raftpb.HeartbeatReply
	heartbeatError error
}
