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

package node

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/datastore"
	"github.com/su225/raft/node/election"
	"github.com/su225/raft/node/heartbeat"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/replication"
	"github.com/su225/raft/node/rest"
	"github.com/su225/raft/node/rpc"
	"github.com/su225/raft/node/rpc/server"
	"github.com/su225/raft/node/state"
)

var context = "CONTEXT"

// ContextLifecycleError represents the errors that occur during lifecycle
// events of the node context. This can be thought of as the consolidated
// error message derived from errors in various components
type ContextLifecycleError struct {
	Errors []error
}

func (e *ContextLifecycleError) Error() string {
	errorMessages := make([]string, 0)
	for _, err := range e.Errors {
		errorMessages = append(errorMessages, err.Error())
	}
	return strings.Join(errorMessages, "\n")
}

// Context represents the holder for all the components
// running as part of this Raft node.
type Context struct {
	// RealRaftProbufServer is an implementation of RaftProtocolServer
	// It is responsible for receiving all incoming protocol-related
	// messages from other nodes and taking appropriate action
	*server.RealRaftProtobufServer

	// RealRaftProtobufClient is responsible for handling all outgoing
	// communication from this node
	rpc.RaftProtobufClient

	// MembershipManager is responsible for managing cluster membership
	// and keeping track of information about other nodes in the cluster
	cluster.MembershipManager

	// EntryPersistence is responsible for persisting and retrieving
	// entries written to write-ahead log
	log.EntryPersistence

	// MetadataPersistence is responsible for persisting and retrieving
	// metadata of write-ahead log
	log.MetadataPersistence

	// WriteAheadLogManager is responsible for managing write-ahead log
	// entries, metadata and persisting them to disk
	log.WriteAheadLogManager

	// RaftStateManager is responsible for managing state related to
	// raft node safely in concurrent environments. It is also responsible
	// for recovery of state across restarts/crashes
	state.RaftStateManager

	// RaftStatePersistence is responsible for persisting and retrieving
	// raft-related state data.
	state.RaftStatePersistence

	// Voter is responsible for deciding whether to grant or reject vote
	*state.Voter

	// LeaderElectionAlgorithm is responsible for actually executing the
	// leader election algorithm. In case of raft it would be becoming
	// candidate, requesting votes and upgrading to leader if necessary
	election.LeaderElectionAlgorithm

	// LeaderElectionManager is responsible for running election timeout,
	// starting and declaring results of leader election
	election.LeaderElectionManager

	// LeaderHeartbeatController is responsible for controlling the sending
	// of heartbeat to remote nodes to establish authority as leader
	heartbeat.LeaderHeartbeatController

	// APIServer receives key-value store operation requests from the user.
	// If the current node is the leader then it is appended to the log and
	// it tries to replicate it to other nodes. If it is not a leader then
	// it forwards to the leader node if it is known. Otherwise error is
	// returned informing the same.
	*rest.APIServer

	// DataStore represents the API through which the distributed key-value
	// store is accessed. All the complexities of distributing are hidden
	// beneath this component.
	datastore.DataStore

	// EntryReplicationController is responsible for handling replication of
	// entries across the cluster when the node is elected as leader
	replication.EntryReplicationController

	// SnapshotPersistence is responsible for handling persistence of key-value
	// pairs as part of the snapshot
	log.SnapshotPersistence

	// SnapshotMetadataPersistence is responsible for handling persistence of
	// snapshot related metadata like snapshot index, epoch etc
	log.SnapshotMetadataPersistence

	// SnapshotHandler is responsible for handling snapshot
	log.SnapshotHandler

	// EntryGarbageCollector is responsible for cleaning up entries
	// once their snapshot is taken
	EntryGarbageCollector log.GarbageCollector

	// SnapshotGarbageCollector is responsible for cleaning up snapshots
	// which are not longer used. 
	SnapshotGarbageCollector log.GarbageCollector
}

// NewContext creates a new node context and returns it
// It wires up all the components before returning.
func NewContext(config *Config) *Context {
	joiner := getJoiner(config)
	currentNodeInfo := cluster.NodeInfo{
		ID:     config.NodeID,
		APIURL: fmt.Sprintf(":%d", config.APIPort),
		RPCURL: fmt.Sprintf(":%d", config.RPCPort),
	}
	membershipManager := cluster.NewRealMembershipManager(currentNodeInfo, joiner)

	realRaftProtobufClient := rpc.NewRealRaftProtobufClient(
		membershipManager,
		config.NodeID,
		config.MaxConnectionRetryAttempts,
		uint64(config.RPCTimeoutInMillis),
	)

	entryPersistence := log.NewFileBasedEntryPersistence(config.WriteAheadLogEntryPath)
	metadataPersistence := log.NewFileBasedMetadataPersistence(config.WriteAheadLogMetadataPath)
	writeAheadLogManager := log.NewWriteAheadLogManagerImpl(
		entryPersistence,
		metadataPersistence,
		nil, // SnapshotHandler filled later
	)
	snapshotPersistence := log.NewSimpleFileBasedSnapshotPersistence(config.SnapshotPath)
	snapshotMetadataPersistence := log.NewSimpleFileBasedSnapshotMetadataPersistence(config.SnapshotPath)
	snapshotHandler := log.NewRealSnapshotHandler(
		writeAheadLogManager,
		snapshotPersistence,
		snapshotMetadataPersistence,
		nil, // EntryGarbageCollector filled later	
	)
	entryGarbageCollector := log.NewRealEntryGarbageCollector(
		snapshotHandler,
		entryPersistence,
	)
	writeAheadLogManager.SnapshotHandler = snapshotHandler
	snapshotHandler.EntryGarbageCollector = entryGarbageCollector

	raftStatePersistence := state.NewFileBasedRaftStatePersistence(config.RaftStatePath)
	raftStateManager := state.NewRealRaftStateManager(
		config.NodeID,
		raftStatePersistence,
	)
	voter := state.NewVoter(
		raftStateManager,
		writeAheadLogManager,
	)
	leaderElectionAlgo := election.NewRaftLeaderElectionAlgorithm(
		config.NodeID,
		realRaftProtobufClient,
		raftStateManager,
		writeAheadLogManager,
		membershipManager,
	)
	leaderElectionManager := election.NewRealLeaderElectionManager(
		uint64(config.ElectionTimeoutInMillis),
		leaderElectionAlgo,
	)
	realRaftProtobufServer := server.NewRealRaftProtobufServer(
		config.RPCPort,
		voter,
		raftStateManager,
		leaderElectionManager,
	)
	leaderHeartbeatController := heartbeat.NewRealLeaderHeartbeatController(
		config.NodeID,
		config.HeartbeatIntervalInMillis,
		realRaftProtobufClient,
		raftStateManager,
		writeAheadLogManager,
		membershipManager,
	)
	replicationController := replication.NewRealEntryReplicationController(
		realRaftProtobufClient,
		writeAheadLogManager,
		config.NodeID,
		membershipManager,
		snapshotHandler,
	)

	raftStateManager.RegisterSubscription(leaderElectionManager)
	raftStateManager.RegisterSubscription(leaderHeartbeatController)
	raftStateManager.RegisterSubscription(replicationController)

	dataStore := datastore.NewRaftKeyValueStore(
		replicationController,
		raftStateManager,
		writeAheadLogManager,
		snapshotHandler,
	)

	apiServer := rest.NewAPIServer(
		config.APIPort,
		config.NodeID,
		config.APITimeoutInMillis,
		config.APIFwdTimeoutInMillis,
		raftStateManager,
		membershipManager,
		dataStore,
	)

	return &Context{
		RealRaftProtobufServer:      realRaftProtobufServer,
		RaftProtobufClient:          realRaftProtobufClient,
		MembershipManager:           membershipManager,
		EntryPersistence:            entryPersistence,
		MetadataPersistence:         metadataPersistence,
		WriteAheadLogManager:        writeAheadLogManager,
		RaftStateManager:            raftStateManager,
		RaftStatePersistence:        raftStatePersistence,
		Voter:                       voter,
		LeaderElectionAlgorithm:     leaderElectionAlgo,
		LeaderElectionManager:       leaderElectionManager,
		LeaderHeartbeatController:   leaderHeartbeatController,
		APIServer:                   apiServer,
		DataStore:                   dataStore,
		EntryReplicationController:  replicationController,
		SnapshotPersistence:         snapshotPersistence,
		SnapshotMetadataPersistence: snapshotMetadataPersistence,
		SnapshotHandler:             snapshotHandler,
		EntryGarbageCollector:       entryGarbageCollector,
	}
}

// Start starts various node context components. If the
// operation is not successful then it returns error
func (ctx *Context) Start() error {
	if membershipMgrStartErr := ctx.MembershipManager.Start(); membershipMgrStartErr != nil {
		return membershipMgrStartErr
	}
	if protobufClientStartErr := ctx.RaftProtobufClient.Start(); protobufClientStartErr != nil {
		return protobufClientStartErr
	}
	if writeAheadLogMgrStartErr := ctx.WriteAheadLogManager.Start(); writeAheadLogMgrStartErr != nil {
		return writeAheadLogMgrStartErr
	}
	if leaderElectionMgrErr := ctx.LeaderElectionManager.Start(); leaderElectionMgrErr != nil {
		return leaderElectionMgrErr
	}
	if heartbeatCtrlErr := ctx.LeaderHeartbeatController.Start(); heartbeatCtrlErr != nil {
		return heartbeatCtrlErr
	}
	if replicationCtrlErr := ctx.EntryReplicationController.Start(); replicationCtrlErr != nil {
		return replicationCtrlErr
	}
	if stateMgrStartErr := ctx.RaftStateManager.Start(); stateMgrStartErr != nil {
		return stateMgrStartErr
	}
	if probufStartErr := ctx.RealRaftProtobufServer.Start(); probufStartErr != nil {
		return probufStartErr
	}
	if restStartErr := ctx.APIServer.Start(); restStartErr != nil {
		return restStartErr
	}
	return nil
}

// Destroy destroys all components so that they become
// non-operational and the resources can be cleaned up
// This allows for graceful shutdown of each of the
// component in the node.
func (ctx *Context) Destroy() error {
	contextErrorMessage := &ContextLifecycleError{Errors: []error{}}
	if protobufDestroyErr := ctx.RealRaftProtobufServer.Destroy(); protobufDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, protobufDestroyErr)
	}
	if restDestroyErr := ctx.APIServer.Destroy(); restDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, restDestroyErr)
	}
	if heartbeatDestroyErr := ctx.LeaderHeartbeatController.Destroy(); heartbeatDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, heartbeatDestroyErr)
	}
	if leaderElectionMgrDestroyErr := ctx.LeaderElectionManager.Destroy(); leaderElectionMgrDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, leaderElectionMgrDestroyErr)
	}
	if replicationCtrlDestroyErr := ctx.EntryReplicationController.Destroy(); replicationCtrlDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, replicationCtrlDestroyErr)
	}
	if stateMgrDestroyErr := ctx.RaftStateManager.Destroy(); stateMgrDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, stateMgrDestroyErr)
	}
	if writeAheadLogMgrDestroyErr := ctx.WriteAheadLogManager.Destroy(); writeAheadLogMgrDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, writeAheadLogMgrDestroyErr)
	}
	if protobufClientDestroyErr := ctx.RaftProtobufClient.Destroy(); protobufClientDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, protobufClientDestroyErr)
	}
	if membershipMgrDestroyErr := ctx.MembershipManager.Destroy(); membershipMgrDestroyErr != nil {
		contextErrorMessage.Errors = append(contextErrorMessage.Errors, membershipMgrDestroyErr)
	}
	logrus.WithFields(logrus.Fields{
		logfield.ErrorReason: contextErrorMessage.Error(),
		logfield.Component:   context,
		logfield.Event:       "DESTROY",
	}).Errorln("error while destroying components")
	if len(contextErrorMessage.Errors) > 0 {
		return contextErrorMessage
	}
	return nil
}

func getJoiner(config *Config) cluster.Joiner {
	if config.JoinMode == KubernetesJoinMode {
		panic("k8s-mode is not yet implemented")
	}
	return cluster.NewStaticFileBasedJoiner(config.ClusterConfigPath)
}
