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

import "github.com/su225/raft/node/data"

type snapshotHandlerCommand interface {
	IsSnapshotHandlerCommand() bool
}

type snapshotHandlerStart struct {
	snapshotHandlerCommand
	errorChan chan error
}

type snapshotHandlerDestroy struct {
	snapshotHandlerCommand
	errorChan chan error
}

type snapshotHandlerRecover struct {
	snapshotHandlerCommand
	errorChan chan error
}

type snapshotHandlerFreeze struct {
	snapshotHandlerCommand
	errorChan chan error
}

type snapshotHandlerUnfreeze struct {
	snapshotHandlerCommand
	errorChan chan error
}

type runSnapshotBuilder struct {
	snapshotHandlerCommand
	errorChan chan error
}

type stopSnapshotBuilder struct {
	snapshotHandlerCommand
	errorChan chan error
}

type terminateSnapshotBuilder struct {
	snapshotHandlerCommand
	errorChan chan error
}

type addKVPair struct {
	snapshotHandlerCommand
	epoch      uint64
	key, value string
	errorChan  chan error
}

type removeKVPair struct {
	snapshotHandlerCommand
	epoch     uint64
	key       string
	errorChan chan error
}

type getKVPair struct {
	snapshotHandlerCommand
	epoch     uint64
	key       string
	replyChan chan *getKVPairReply
}

type getKVPairReply struct {
	key, value string
	err        error
}

type forEachKeyValuePair struct {
	snapshotHandlerCommand
	epoch         uint64
	transformFunc func(data.KVPair) error
	errorChan     chan error
}

type createEpoch struct {
	snapshotHandlerCommand
	epoch     uint64
	errorChan chan error
}

type deleteEpoch struct {
	snapshotHandlerCommand
	epoch     uint64
	errorChan chan error
}

type getSnapshotMetadata struct {
	snapshotHandlerCommand
	replyChan chan SnapshotMetadata
}

type setSnapshotMetadata struct {
	snapshotHandlerCommand
	metadata  SnapshotMetadata
	errorChan chan error
}
