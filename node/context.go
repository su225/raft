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

	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/log"
	"github.com/su225/raft/node/rpc"
)

// Context represents the holder for all the components
// running as part of this Raft node.
type Context struct {
	// RealRaftProbufServer is an implementation of RaftProtocolServer
	// It is responsible for receiving all incoming protocol-related
	// messages from other nodes and taking appropriate action
	*rpc.RealRaftProtobufServer

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
}

// NewContext creates a new node context and returns it
// It wires up all the components before returning.
func NewContext(config *Config) *Context {
	realRaftProtobufServer := rpc.NewRealRaftProtobufServer(config.RPCPort)

	joiner := getJoiner(config)
	currentNodeInfo := cluster.NodeInfo{
		ID:     config.NodeID,
		APIURL: fmt.Sprintf(":%d", config.APIPort),
		RPCURL: fmt.Sprintf(":%d", config.RPCPort),
	}
	membershipManager := cluster.NewRealMembershipManager(currentNodeInfo, joiner)

	entryPersistence := log.NewFileBasedEntryPersistence(config.WriteAheadLogEntryPath)
	metadataPersistence := log.NewFileBasedMetadataPersistence(config.WriteAheadLogMetadataPath)
	writeAheadLogManager := log.NewWriteAheadLogManagerImpl(
		entryPersistence,
		metadataPersistence,
	)

	return &Context{
		RealRaftProtobufServer: realRaftProtobufServer,
		MembershipManager:      membershipManager,
		EntryPersistence:       entryPersistence,
		MetadataPersistence:    metadataPersistence,
		WriteAheadLogManager:   writeAheadLogManager,
	}
}

// Start starts various node context components. If the
// operation is not successful then it returns error
func (ctx *Context) Start() error {
	if probufStartErr := ctx.RealRaftProtobufServer.Start(); probufStartErr != nil {
		return probufStartErr
	}
	if membershipMgrStartErr := ctx.MembershipManager.Start(); membershipMgrStartErr != nil {
		return membershipMgrStartErr
	}
	if writeAheadLogMgrStartErr := ctx.WriteAheadLogManager.Start(); writeAheadLogMgrStartErr != nil {
		return writeAheadLogMgrStartErr
	}
	return nil
}

// Destroy destroys all components so that they become
// non-operational and the resources can be cleaned up
// This allows for graceful shutdown of each of the
// component in the node.
func (ctx *Context) Destroy() error {
	if writeAheadLogMgrDestroyErr := ctx.WriteAheadLogManager.Destroy(); writeAheadLogMgrDestroyErr != nil {
		return writeAheadLogMgrDestroyErr
	}
	if membershipMgrDestroyErr := ctx.MembershipManager.Destroy(); membershipMgrDestroyErr != nil {
		return membershipMgrDestroyErr
	}
	if protobufDestroyErr := ctx.RealRaftProtobufServer.Destroy(); protobufDestroyErr != nil {
		return protobufDestroyErr
	}
	return nil
}

func getJoiner(config *Config) cluster.Joiner {
	if config.JoinMode == KubernetesJoinMode {
		panic("k8s-mode is not yet implemented")
	}
	return cluster.NewStaticFileBasedJoiner(config.ClusterConfigPath)
}
