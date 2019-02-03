package replication

import "github.com/su225/raft/node/log"

type replicationControllerCommand interface {
	IsReplicationControllerCommand() bool
}

type replicationControllerDestroy struct {
	replicationControllerCommand
	errorChan chan error
}

type replicationControllerPause struct {
	replicationControllerCommand
	errorChan chan error
}

type replicationControllerResume struct {
	replicationControllerCommand
	errorChan chan error
}

type replicateEntry struct {
	replicationControllerCommand
	entryID   log.EntryID
	replyChan chan *replicateEntryReply
}

type replicateEntryReply struct {
	matchIndex     uint64
	replicationErr error
}
