package replication

import (
	"sync"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/common"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/rpc"
	"github.com/su225/raft/node/state"
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

const replicationCtrl = "REPLICATION"

var replicationCtrlIsDestroyedErr = &common.ComponentIsDestroyedError{ComponentName: replicationCtrl}

// RealEntryReplicationController is the implementation of replication
// controller which talks to other nodes in the effort of replicating
// a given entry to their logs. This must be active only when the node
// is the leader in the cluster
type RealEntryReplicationController struct {
	chanSelectorMutex sync.RWMutex
	commandChannel    map[string]chan replicationControllerCommand
	listenerChannel   chan state.RaftStateManagerEvent

	// Client is required to send messages to other nodes
	// requesting them to add entry to their logs
	rpc.RaftProtobufClient

	// WriteAheadLogManager is required to fetch entries
	// with given index.
	log.WriteAheadLogManager

	// SnapshotHandler is required to freeze/unfreeze snapshot
	// during replication
	log.SnapshotHandler

	// CurrentNodeID and MembershipManager are required to
	// know each member and send them entries
	CurrentNodeID string
	cluster.MembershipManager
}

// NewRealEntryReplicationController creates a new instance of real entry
// replication controller and returns the same
func NewRealEntryReplicationController(
	rpcClient rpc.RaftProtobufClient,
	writeAheadLogMgr log.WriteAheadLogManager,
	currentNodeID string,
	membershipMgr cluster.MembershipManager,
	snapshotHandler log.SnapshotHandler,
) *RealEntryReplicationController {
	return &RealEntryReplicationController{
		chanSelectorMutex:    sync.RWMutex{},
		commandChannel:       make(map[string]chan replicationControllerCommand),
		listenerChannel:      make(chan state.RaftStateManagerEvent),
		RaftProtobufClient:   rpcClient,
		WriteAheadLogManager: writeAheadLogMgr,
		SnapshotHandler:      snapshotHandler,
		CurrentNodeID:        currentNodeID,
		MembershipManager:    membershipMgr,
	}
}

// Start starts the component. It makes the component operational
func (r *RealEntryReplicationController) Start() error {
	go r.listener()
	for _, node := range r.MembershipManager.GetAllNodes() {
		if node.ID == r.CurrentNodeID {
			continue
		}
		r.addChannel(node.ID)
		go r.loop(node)
	}
	return nil
}

type controllerMessageType uint8

const (
	destroy controllerMessageType = iota
	pause
	resume
)

func getControllerMessageString(msgType controllerMessageType) string {
	switch msgType {
	case destroy:
		return "destroy"
	case pause:
		return "pause"
	case resume:
		return "resume"
	}
	return ""
}

// Destroy makes the component non-operational
func (r *RealEntryReplicationController) Destroy() error {
	return r.sendControllerMessage(destroy)
}

// Pause pauses the component's operation, but unlike destroy it
// does not make the component non-operational
func (r *RealEntryReplicationController) Pause() error {
	return r.sendControllerMessage(pause)
}

// Resume resumes component's operation.
func (r *RealEntryReplicationController) Resume() error {
	return r.sendControllerMessage(resume)
}

func (r *RealEntryReplicationController) sendControllerMessage(msgType controllerMessageType) error {
	nodeCount := len(r.commandChannel)
	opWaitGroup := sync.WaitGroup{}
	opWaitGroup.Add(nodeCount)
	for nodeID, nodeChan := range r.commandChannel {
		go func(nodeID string, nodeChan chan<- replicationControllerCommand) {
			defer opWaitGroup.Done()
			errorChan := make(chan error)
			switch msgType {
			case destroy:
				nodeChan <- &replicationControllerDestroy{errorChan: errorChan}
			case pause:
				nodeChan <- &replicationControllerPause{errorChan: errorChan}
			case resume:
				nodeChan <- &replicationControllerResume{errorChan: errorChan}
			}
			if err := <-errorChan; err != nil {
				msgTypeStr := getControllerMessageString(msgType)
				logrus.WithFields(logrus.Fields{
					logfield.ErrorReason: err.Error(),
					logfield.Component:   replicationCtrl,
					logfield.Event:       msgTypeStr,
				}).Errorf("error during %s for %s", msgTypeStr, nodeID)
			}
		}(nodeID, nodeChan)
	}
	opWaitGroup.Wait()
	return nil
}

type matchIndexCollector struct {
	idxMutex sync.Mutex
	indices  []uint64
}

func newMatchIndexCollector() *matchIndexCollector {
	return &matchIndexCollector{
		idxMutex: sync.Mutex{},
		indices:  make([]uint64, 0),
	}
}

func (m *matchIndexCollector) add(idx uint64) {
	m.idxMutex.Lock()
	defer m.idxMutex.Unlock()
	m.indices = append(m.indices, idx)
}

func (m *matchIndexCollector) canCommit(idx uint64, needed int32) bool {
	m.idxMutex.Lock()
	defer m.idxMutex.Unlock()
	count := int32(1)
	for _, i := range m.indices {
		if i == idx {
			count++
		}
	}
	return count >= needed
}

// ReplicateEntry tries to replicate the given entry at the given entryID
// and returns nil if replication is successful or false otherwise.
func (r *RealEntryReplicationController) ReplicateEntry(entryID log.EntryID, entry log.Entry) error {
	totalNodes := int32(len(r.commandChannel) + 1)
	majorityCount := int32(totalNodes/2 + 1)

	proceed, proceedSignal := make(chan struct{}), sync.Once{}
	replicationCount, nodesReached := int32(1), int32(1)
	matchIndexCollector := newMatchIndexCollector()

	for nodeID, nodeChan := range r.commandChannel {
		go func(nodeID string, nodeChan chan<- replicationControllerCommand) {
			defer func() {
				updatedNodesReached := atomic.AddInt32(&nodesReached, 1)
				if updatedNodesReached == totalNodes {
					proceedSignal.Do(func() { proceed <- struct{}{} })
				}
			}()
			replyChan := make(chan *replicateEntryReply)
			nodeChan <- &replicateEntry{
				entryID:   entryID,
				replyChan: replyChan,
			}
			reply := <-replyChan
			if reply.replicationErr != nil {
				logrus.WithFields(logrus.Fields{
					logfield.ErrorReason: reply.replicationErr.Error(),
					logfield.Component:   replicationCtrl,
					logfield.Event:       "REPLICATION",
				}).Errorf("error while replicating entry (%d,%d) to %s",
					entryID.TermID, entryID.Index, nodeID)
				return
			}
			matchIndexCollector.add(reply.matchIndex)
			if updatedReplicationCount := atomic.AddInt32(&replicationCount, 1); updatedReplicationCount >= majorityCount {
				proceedSignal.Do(func() { proceed <- struct{}{} })
			}
		}(nodeID, nodeChan)
	}
	<-proceed
	if atomic.LoadInt32(&replicationCount) < majorityCount {
		return &EntryCannotBeCommittedError{Index: entryID.Index}
	}
	if !matchIndexCollector.canCommit(entryID.Index, majorityCount) {
		return &EntryCannotBeCommittedError{Index: entryID.Index}
	}
	if _, updateErr := r.WriteAheadLogManager.UpdateMaxCommittedIndex(entryID.Index); updateErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: updateErr.Error(),
			logfield.Component:   replicationCtrl,
			logfield.Event:       "UPDATE-COMMIT-IDX",
		}).Errorf("error while updating committed index")
		return updateErr
	}
	return nil
}

func (r *RealEntryReplicationController) getChannel(nodeID string) (chan replicationControllerCommand, bool) {
	r.chanSelectorMutex.RLock()
	defer r.chanSelectorMutex.RUnlock()
	if nodeChan, isPresent := r.commandChannel[nodeID]; isPresent {
		return nodeChan, true
	}
	return nil, false
}

func (r *RealEntryReplicationController) addChannel(nodeID string) {
	r.chanSelectorMutex.Lock()
	defer r.chanSelectorMutex.Unlock()
	r.commandChannel[nodeID] = make(chan replicationControllerCommand)
}

type replicationControllerState struct {
	isStarted   bool
	isPaused    bool
	isDestroyed bool

	remoteNodeInfo cluster.NodeInfo
	matchIndex     uint64
}

func (r *RealEntryReplicationController) loop(nodeInfo cluster.NodeInfo) {
	state := &replicationControllerState{
		isStarted:   true,
		isPaused:    true,
		isDestroyed: false,

		remoteNodeInfo: nodeInfo,
		matchIndex:     0,
	}
	cmdChan, isPresent := r.getChannel(nodeInfo.ID)
	if !isPresent {
		return
	}
	for {
		cmd := <-cmdChan
		switch c := cmd.(type) {
		case *replicationControllerDestroy:
			c.errorChan <- r.handleReplicationControllerDestroy(state, c)
		case *replicationControllerPause:
			c.errorChan <- r.handleReplicationControllerPause(state, c)
		case *replicationControllerResume:
			c.errorChan <- r.handleReplicationControllerResume(state, c)
		case *replicateEntry:
			c.replyChan <- r.handleReplicateEntry(state, c)
		}
	}
}

func (r *RealEntryReplicationController) handleReplicationControllerDestroy(state *replicationControllerState, cmd *replicationControllerDestroy) error {
	if state.isDestroyed {
		return nil
	}
	state.isPaused = true
	state.isDestroyed = true
	return nil
}

func (r *RealEntryReplicationController) handleReplicationControllerPause(state *replicationControllerState, cmd *replicationControllerPause) error {
	if state.isDestroyed {
		return replicationCtrlIsDestroyedErr
	}
	state.isPaused = true
	return nil
}

func (r *RealEntryReplicationController) handleReplicationControllerResume(state *replicationControllerState, cmd *replicationControllerResume) error {
	if state.isDestroyed {
		return replicationCtrlIsDestroyedErr
	}
	state.matchIndex = 0
	state.isPaused = false
	return nil
}

func (r *RealEntryReplicationController) handleReplicateEntry(state *replicationControllerState, cmd *replicateEntry) *replicateEntryReply {
	if state.isDestroyed {
		return &replicateEntryReply{
			matchIndex:     state.matchIndex,
			replicationErr: replicationCtrlIsDestroyedErr,
		}
	}
	if cmd.entryID.Index <= state.matchIndex {
		return nil
	}

	// Freezing snapshot handler during replication also freezes
	// entry garbage collector which means entries are not deleted
	// thereby initiating unnecessary snapshot transfers
	r.SnapshotHandler.Freeze()
	defer r.SnapshotHandler.Unfreeze()

	curTermID := cmd.entryID.TermID
	curEntryIndex := cmd.entryID.Index
	initMatchIndex := state.matchIndex

	snapshotMetadata := r.SnapshotHandler.GetSnapshotMetadata()
	snapshotIndex := snapshotMetadata.Index

	// Try to transfer the current entry, if it is not possible then
	// try transferring the previous entry until initial match index
	// is hit (it does not make sense to go less than that since from
	// previous rounds it is confirmed that logs match until that point)
	//
	// If current snapshot index is hit in the middle then just transfer
	// the snapshot there by fast forwarding the log to that point. After
	// that send entries from the snapshot index to the end
	replicateIndex := curEntryIndex
	for replicateIndex > initMatchIndex && replicateIndex <= curEntryIndex {
		logrus.Debugf("nodeID=%s, replicateIndex=%d", state.remoteNodeInfo.ID, replicateIndex)
		if replicateIndex == snapshotIndex {
			matchIndex, transferErr := r.doTransferSnapshot(state, curTermID)
			if transferErr != nil {
				return &replicateEntryReply{
					matchIndex:     state.matchIndex,
					replicationErr: transferErr,
				}
			}
			replicateIndex = matchIndex + 1
			continue
		}
		replicationSuccessful, replicationErr := r.doReplicateEntry(state, curTermID, replicateIndex)
		if replicationErr != nil {
			matchIndex, transferErr := r.doTransferSnapshot(state, curTermID)
			if transferErr != nil {
				return &replicateEntryReply{
					matchIndex:     state.matchIndex,
					replicationErr: transferErr,
				}
			}
			replicateIndex = matchIndex + 1
			continue
		}
		if replicationSuccessful {
			if state.matchIndex < replicateIndex {
				state.matchIndex = replicateIndex
			}
			replicateIndex++
		} else {
			replicateIndex--
		}
	}

	return &replicateEntryReply{
		matchIndex:     state.matchIndex,
		replicationErr: nil,
	}
}

func (r *RealEntryReplicationController) doTransferSnapshot(state *replicationControllerState, curTermID uint64) (uint64, error) {
	matchIndex, transferErr := r.RaftProtobufClient.InstallSnapshot(curTermID, state.remoteNodeInfo.ID)
	if transferErr != nil {
		return 0, transferErr
	}
	if matchIndex > state.matchIndex {
		state.matchIndex = matchIndex
	}
	return matchIndex, nil
}

func (r *RealEntryReplicationController) doReplicateEntry(state *replicationControllerState, curTermID, entryIndex uint64) (bool, error) {
	remoteNodeID := state.remoteNodeInfo.ID
	logrus.WithFields(logrus.Fields{
		logfield.Component: replicationCtrl,
		logfield.Event:     "REPLICATION",
	}).Debugf("replicating entry #%d to %s", entryIndex, remoteNodeID)

	curEntry, prevEntry, fetchErr := r.getPreviousAndCurrentEntries(entryIndex)
	if fetchErr != nil {
		return false, fetchErr
	}
	prevEntryID := log.EntryID{TermID: prevEntry.GetTermID(), Index: entryIndex - 1}

	appendedSuccessfully, appendErr := r.RaftProtobufClient.AppendEntry(curTermID, remoteNodeID, prevEntryID, entryIndex, curEntry)
	if appendErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: appendErr.Error(),
			logfield.Component:   replicationCtrl,
			logfield.Event:       "APPEND-RPC",
		}).Errorf("error while replicating entry #%d to %s", entryIndex, remoteNodeID)
		return false, appendErr
	}
	return appendedSuccessfully, nil
}

func (r *RealEntryReplicationController) getPreviousAndCurrentEntries(index uint64) (log.Entry, log.Entry, error) {
	curEntry, fetchErr := r.getEntryAtIndexOrFail(index)
	prevEntry, fetchErr := r.getEntryAtIndexOrFail(index)
	return curEntry, prevEntry, fetchErr
}

func (r *RealEntryReplicationController) getEntryAtIndexOrFail(index uint64) (log.Entry, error) {
	entry, fetchErr := r.WriteAheadLogManager.GetEntry(index)
	if fetchErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: fetchErr.Error(),
			logfield.Component:   replicationCtrl,
			logfield.Event:       "ENTRY-FETCH",
		}).Errorf("error while fetching entry #%d", index)
		return nil, fetchErr
	}
	return entry, nil
}

type replicationControllerListenerState struct {
	isDestroyed bool
}

func (r *RealEntryReplicationController) listener() {
	lisState := &replicationControllerListenerState{
		isDestroyed: false,
	}
	for {
		event := <-r.listenerChannel
		switch ev := event.(type) {
		case *state.UpgradeToLeaderEvent:
			r.onUpgradeToLeader(lisState, ev)
		case *state.BecomeCandidateEvent:
			r.onBecomeCandidate(lisState, ev)
		case *state.DowngradeToFollowerEvent:
			r.onDowngradeToFollower(lisState, ev)
		}
	}
}

// NotifyChannel is needed to subscribe to events from RaftStateManager
func (r *RealEntryReplicationController) NotifyChannel() chan<- state.RaftStateManagerEvent {
	return r.listenerChannel
}

func (r *RealEntryReplicationController) onUpgradeToLeader(state *replicationControllerListenerState, ev *state.UpgradeToLeaderEvent) error {
	r.Resume()
	logrus.WithFields(logrus.Fields{
		logfield.Component: replicationCtrl,
		logfield.Event:     "UPGRADE-TO-LEADER",
	}).Debugf("Leader now - start replication controller")
	return nil
}

func (r *RealEntryReplicationController) onBecomeCandidate(state *replicationControllerListenerState, ev *state.BecomeCandidateEvent) error {
	return r.Pause()
}

func (r *RealEntryReplicationController) onDowngradeToFollower(state *replicationControllerListenerState, ev *state.DowngradeToFollowerEvent) error {
	return r.Pause()
}
