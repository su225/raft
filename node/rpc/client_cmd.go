package rpc

import "github.com/su225/raft/node/log"

type protocolClientCommand interface {
	IsProtocolClientCommand() bool
}

type startClient struct {
	protocolClientCommand
	errChan chan error
}

type destroyClient struct {
	protocolClientCommand
	errChan chan error
}

type clientRequestVote struct {
	protocolClientCommand
	currentTermID  uint64
	lastLogEntryID log.EntryID
	replyChan      chan *clientRequestVoteReply
}

type clientRequestVoteReply struct {
	voteGranted bool
	votingErr   error
}

type clientHeartbeat struct {
	protocolClientCommand
	currentTermID     uint64
	maxCommittedIndex uint64
	replyChan         chan *clientHeartbeatReply
}

type clientHeartbeatReply struct {
	acceptedAsLeader bool
	heartbeatErr     error
}

type clientAppendEntry struct {
	protocolClientCommand
	currentTermID uint64
	index         uint64
	entry         log.Entry
	prevEntryID   log.EntryID
	replyChan     chan *clientAppendEntryReply
}

type clientAppendEntryReply struct {
	entryAppended bool
	appendErr     error
}

type clientReconnect struct {
	protocolClientCommand
	errChan chan error
}
