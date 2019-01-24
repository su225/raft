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
