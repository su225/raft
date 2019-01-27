package state

import "errors"

// InMemoryRaftStatePersistence represents in-memory persistence.
// This MUST be used only FOR TESTING PURPOSES.
type InMemoryRaftStatePersistence struct {
	ShouldSucceed bool
	*RaftDurableState
}

var errStatePersistence = errors.New("state-persistence error")

// NewInMemoryRaftStatePersistence creates a new instance of in-memory
// raft state persistence. "succeed" indicates whether to fail the operations
// and "state" is the initial state of the persistence
func NewInMemoryRaftStatePersistence(succeed bool, state *RaftDurableState) *InMemoryRaftStatePersistence {
	return &InMemoryRaftStatePersistence{
		ShouldSucceed:    succeed,
		RaftDurableState: state,
	}
}

// PersistRaftState stores the given state if calls are supposed to be successful.
// Otherwise they return a generic state persistence-related error
func (p *InMemoryRaftStatePersistence) PersistRaftState(state *RaftDurableState) error {
	if !p.ShouldSucceed {
		return errStatePersistence
	}
	p.RaftDurableState = state
	return nil
}

// RetrieveRaftState returns the stored state if calls are supposed to be successful.
// Otherwise they return a generic state persistence-related error
func (p *InMemoryRaftStatePersistence) RetrieveRaftState() (*RaftDurableState, error) {
	if !p.ShouldSucceed {
		return nil, errStatePersistence
	}
	return p.RaftDurableState, nil
}
