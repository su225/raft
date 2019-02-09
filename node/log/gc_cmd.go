package log

import "github.com/su225/raft/node/common"

type commandType uint8

const (
	start commandType = iota
	destroy
	pause
	resume
	freeze
	unfreeze
)

// GarbageCollector is responsible for cleaning
// up the entries which are already part of some snapshot
// or snapshots which are no longer used (previous epoch)
type GarbageCollector interface {
	common.ComponentLifecycle
	common.Pausable
	common.Freezable
}

type gcCommand struct {
	cmd     commandType
	errChan chan error
}
