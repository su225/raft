package replication

import (
	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/log"
)

// EntryReplicationController is responsible for controlling and
// coordinating the replication of log entries to the majority of
// nodes in the Raft cluster.
type EntryReplicationController interface {
	// ReplicateEntry replicates the given entry at the given entryID
	// to at least majority of the nodes in the cluster. If there is
	// an error during replication then it is returned. Replication
	// is still successful if there are errors in minority of them
	// since all it needs is the entry to be present in majority.
	ReplicateEntry(entryID log.EntryID, entry log.Entry) error

	// ComponentLifecycle makes the replication controller a component
	// with Start() and Destroy() lifecycle methods.
	common.ComponentLifecycle

	// Pausable makes the component pausable. That is - it can start
	// and stop operations without becoming non-functional
	common.Pausable
}

// RealEntryReplicationController is the implementation of replication
// controller which talks to other nodes in the effort of replicating
// a given entry to their logs. This must be active only when the node
// is the leader in the cluster
type RealEntryReplicationController struct {
}

// NewRealEntryReplicationController creates a new instance of real entry
// replication controller and returns the same
func NewRealEntryReplicationController() *RealEntryReplicationController {
	return &RealEntryReplicationController{}
}

// Start starts the component. It makes the component operational
func (r *RealEntryReplicationController) Start() error {
	return nil
}

// Destroy makes the component non-operational
func (r *RealEntryReplicationController) Destroy() error {
	return nil
}

// Pause pauses the component's operation, but unlike destroy it
// does not make the component non-operational
func (r *RealEntryReplicationController) Pause() error {
	return nil
}

// Resume resumes component's operation.
func (r *RealEntryReplicationController) Resume() error {
	return nil
}

// ReplicateEntry tries to replicate the given entry at the given entryID
// and returns nil if replication is successful or false otherwise.
func (r *RealEntryReplicationController) ReplicateEntry(entryID log.EntryID, entry log.Entry) error {
	return nil
}
