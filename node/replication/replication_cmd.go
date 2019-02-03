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
	entry     log.Entry
	errorChan chan error
}
