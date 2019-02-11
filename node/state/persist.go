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

import (
	"encoding/json"
	"io/ioutil"
)

// RaftStatePersistence defines the operations that must be
// supported by implementation wishing to persist/retrieve
// raft-state information across crashes and restarts
type RaftStatePersistence interface {
	// PersistRaftState persists the durable part of raft
	// state. If there is an error in the process then it
	// is returned, otherwise nil
	PersistRaftState(state *RaftDurableState) error

	// RetrieveMetadata retrieves durable part of raft state
	// If there is an error in the process then nil metadata
	// is returned along with the error.
	RetrieveRaftState() (*RaftDurableState, error)
}

// FileBasedRaftStatePersistence persists raft-state to
// a simple file in JSON format. The path of the state
// file must be specified
type FileBasedRaftStatePersistence struct {
	StateFilePath string
}

// NewFileBasedRaftStatePersistence creates a new instance of File-based raft-state
// persistence where stateFilePath specifies the path to the state file.
func NewFileBasedRaftStatePersistence(stateFilePath string) *FileBasedRaftStatePersistence {
	return &FileBasedRaftStatePersistence{StateFilePath: stateFilePath}
}

// PersistRaftState persists raft state to the given file. If there are any
// errors during the process it is returned
func (p *FileBasedRaftStatePersistence) PersistRaftState(state *RaftDurableState) error {
	marshalBytes, marshalErr := json.Marshal(state)
	if marshalErr != nil {
		return marshalErr
	}
	return ioutil.WriteFile(p.StateFilePath, marshalBytes, 0600)
}

// RetrieveRaftState retrieves raft state from the speicified file. If there
// are any errors during the process then it is returned.
func (p *FileBasedRaftStatePersistence) RetrieveRaftState() (*RaftDurableState, error) {
	stateBytes, readErr := ioutil.ReadFile(p.StateFilePath)
	if readErr != nil {
		return nil, readErr
	}
	var state RaftDurableState
	if unmarshalErr := json.Unmarshal(stateBytes, &state); unmarshalErr != nil {
		return nil, unmarshalErr
	}
	return &state, nil
}
