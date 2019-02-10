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
