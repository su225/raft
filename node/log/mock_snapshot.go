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

package log

import (
	"errors"

	"github.com/su225/raft/node/data"
)

// MockSnapshotHandler is the mock of snapshot handler
// which must be used only for TESTING PURPOSES
type MockSnapshotHandler struct {
	ShouldSucceed bool
	SnapshotMetadata
}

// NewMockSnapshotHandler creates and returns a new instance
// of MockSnapshotHandler with the given snapshot metadata.
func NewMockSnapshotHandler(
	shouldSucceed bool,
	metadata SnapshotMetadata,
) *MockSnapshotHandler {
	return &MockSnapshotHandler{
		ShouldSucceed:    false,
		SnapshotMetadata: metadata,
	}
}

// GetDefaultMockSnapshotHandler creates mock snapshot handler with default settings
func GetDefaultMockSnapshotHandler() *MockSnapshotHandler {
	return NewMockSnapshotHandler(true, SnapshotMetadata{Epoch: 1, EntryID: EntryID{}})
}

var errSnapshotHandler = errors.New("snapshot handler error")

// Start does nothing
func (ms *MockSnapshotHandler) Start() error { return nil }

// Destroy does nothing
func (ms *MockSnapshotHandler) Destroy() error { return nil }

// Recover does nothing
func (ms *MockSnapshotHandler) Recover() error { return nil }

// Freeze does nothing
func (ms *MockSnapshotHandler) Freeze() error { return nil }

// Unfreeze does nothing
func (ms *MockSnapshotHandler) Unfreeze() error { return nil }

// RunSnapshotBuilder does nothing
func (ms *MockSnapshotHandler) RunSnapshotBuilder() error { return nil }

// StopSnapshotBuilder does nothing
func (ms *MockSnapshotHandler) StopSnapshotBuilder() error { return nil }

// AddKeyValuePair returns error if shouldSucceed is false
func (ms *MockSnapshotHandler) AddKeyValuePair(epoch uint64, key, value string) error {
	if !ms.ShouldSucceed {
		return errSnapshotHandler
	}
	return nil
}

// RemoveKeyValuePair returns error if shouldSucceed is false
func (ms *MockSnapshotHandler) RemoveKeyValuePair(epoch uint64, key string) error {
	if !ms.ShouldSucceed {
		return errSnapshotHandler
	}
	return nil
}

// GetKeyValuePair gets the key-value pair if shouldSucceed is true
func (ms *MockSnapshotHandler) GetKeyValuePair(epoch uint64, key string) (string, error) {
	if !ms.ShouldSucceed {
		return "", errSnapshotHandler
	}
	return "a", nil
}

// ForEachKeyValuePair does nothing if shouldSucceed is true
func (ms *MockSnapshotHandler) ForEachKeyValuePair(epoch uint64, f func(data.KVPair) error) error {
	if !ms.ShouldSucceed {
		return errSnapshotHandler
	}
	return nil
}

// CreateEpoch returns error if shouldSucceed is false, otherwise does nothing
func (ms *MockSnapshotHandler) CreateEpoch(epochID uint64) error {
	if !ms.ShouldSucceed {
		return errSnapshotHandler
	}
	return nil
}

// DeleteEpoch returns error if shouldSucceed is false, otherwise does nothing
func (ms *MockSnapshotHandler) DeleteEpoch(epochID uint64) error {
	if !ms.ShouldSucceed {
		return errSnapshotHandler
	}
	return nil
}

// SetSnapshotMetadata sets the snapshot metadata
func (ms *MockSnapshotHandler) SetSnapshotMetadata(metadata SnapshotMetadata) error {
	if !ms.ShouldSucceed {
		return errSnapshotHandler
	}
	ms.SnapshotMetadata = metadata
	return nil
}

// GetSnapshotMetadata returns the snapshot metadata
func (ms *MockSnapshotHandler) GetSnapshotMetadata() SnapshotMetadata {
	return ms.SnapshotMetadata
}
