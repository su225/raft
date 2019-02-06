package log

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

type getCurrentEpoch struct {
	snapshotHandlerCommand
	replyChan chan uint64
}

type setCurrentEpoch struct {
	snapshotHandlerCommand
	epoch     uint64
	errorChan chan error
}

type getCurrentSnapshotIndex struct {
	snapshotHandlerCommand
	replyChan chan uint64
}
