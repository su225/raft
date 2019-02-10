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

type writeAheadLogManagerCommand interface {
	isWriteAheadLogManagerCommand() bool
}

type startWriteAheadLogManager struct {
	writeAheadLogManagerCommand
	errChan chan error
}

type destroyWriteAheadLogManager struct {
	writeAheadLogManagerCommand
	errChan chan error
}

type recoverWriteAheadLogMetadata struct {
	writeAheadLogManagerCommand
	errChan chan error
}

type updateMaxCommittedIndex struct {
	writeAheadLogManagerCommand
	index     uint64
	replyChan chan *updateMaxCommittedIndexReply
}

type updateMaxCommittedIndexReply struct {
	updatedIndex uint64
	updateErr    error
}

type getMetadata struct {
	writeAheadLogManagerCommand
	replyChan chan *getMetadataReply
}

type getMetadataReply struct {
	metadata     WriteAheadLogMetadata
	retrievalErr error
}

type forceSetMetadata struct {
	writeAheadLogManagerCommand
	metadata WriteAheadLogMetadata
	errChan  chan error
}

type appendEntry struct {
	writeAheadLogManagerCommand
	entry     Entry
	replyChan chan *writeEntryReply
}

type writeEntry struct {
	writeAheadLogManagerCommand
	index     uint64
	entry     Entry
	replyChan chan *writeEntryReply
}

type writeEntryAfter struct {
	writeEntry
	beforeEntryID EntryID
}

type writeEntryReply struct {
	tailLogEntryID EntryID
	appendErr      error
}

type getEntry struct {
	writeAheadLogManagerCommand
	index     uint64
	replyChan chan *getEntryReply
}

type getEntryReply struct {
	entry        Entry
	retrievalErr error
}
