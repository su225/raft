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
