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

package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node"
)

const mainComponent = "MAIN"

func main() {
	setLogFormatter()

	nodeConfig, configErr := parseConfig()
	if configErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: configErr.Error(),
			logfield.Component:   mainComponent,
			logfield.Event:       "PARSE-CMD-ARGS",
		}).Errorf("error while parsing command-line arguments")
		return
	}
	configValidationErrs := node.Validate(nodeConfig)
	if len(configValidationErrs) > 0 {
		for _, err := range configValidationErrs {
			logrus.WithFields(logrus.Fields{
				logfield.ErrorReason: err.Error(),
				logfield.Component:   mainComponent,
				logfield.Event:       "VALIDATE-CONFIG",
			}).Errorf("config validation error")
		}
		return
	}

	logrus.WithFields(logrus.Fields{
		logfield.Component: mainComponent,
		logfield.Event:     "CONFIG",
	}).Infof("NodeID=%s, APIPort=%d, RPCPort=%d",
		nodeConfig.NodeID,
		nodeConfig.APIPort,
		nodeConfig.RPCPort)

	nodeContext := node.NewContext(nodeConfig)
	if startupErr := nodeContext.Start(); startupErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: startupErr.Error(),
			logfield.Component:   mainComponent,
			logfield.Event:       "START-CTX",
		}).Error("error while starting node ops")
		nodeContext.Destroy()
		os.Exit(-1)
	}
	setupGracefulShutdown(nodeContext)
}

// setLogFormatter sets up some options for formatting log entries.
// For now debug, colors are enabled and it is plain text formatter.
func setLogFormatter() {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	logrus.SetLevel(logrus.DebugLevel)
}

// parseConfig parses command-line arguments and builds the configuration
// If there are any errors then it is returned
func parseConfig() (*node.Config, error) {
	configFile := flag.String("config-file", "",
		"JSON configuration file. When this is specified all other arguments are ignored")

	nodeID := flag.String("id", "",
		"ID of the node")

	apiPort := flag.Uint("api-port", 6666,
		"Port used by api-server")

	rpcPort := flag.Uint("rpc-port", 6667,
		"Port used by rpc-server")

	writeAheadLogEntryPath := flag.String("log-entry-path", ".",
		"Path where where write-ahead log entries are stored")

	writeAheadLogMetadataPath := flag.String("log-metadata-path", ".",
		"Path where write-ahead log metadata is stored")

	raftStatePath := flag.String("raft-state-path", ".",
		"Path where the state of the raft node is stored")

	joinMode := flag.String("join-mode", "cluster-file",
		"Specifies the mode of peer discovery (allowed values: cluster-file, k8s)")

	clusterConfigPath := flag.String("cluster-config-path", ".",
		`Path where cluster configuration is stored. This is helpful for peer discovery.
		 The cluster configuration format is dependent on join-mode argument.`)

	electionTimeout := flag.Int64("election-timeout", 2000,
		`Election timeout is interval for which the current node waits for heartbeat from
		 the leader. If heartbeat is not received within this interval, then leader
		 election is triggered`)

	heartbeatInterval := flag.Int64("heartbeat", 500,
		`Heartbeat interval is the interval between which the leader node sends heartbeats
		 to all other nodes in the cluster. This must be much lesser than election timeout
		 so that elections are not triggered too frequently`)

	rpcTimeout := flag.Int64("rpc-timeout", 1000,
		`RPC Timeout is the time within which the RPC call to another node must get a
		 reply. Otherwise it will be timed-out. But the connection might be maintained
		 to the remote node if possible`)

	apiTimeout := flag.Int64("api-timeout", 2000,
		`API Timeout is the time within which the API call must complete. Otherwise
		 something like Gateway timed out or internal server error are returned`)

	apiFwdTimeout := flag.Int64("api-fwd-timeout", 1500,
		`API Forward timeout is the timeout within which the response for the forwarded
		 request must be returned. Otherwise it will be a gateway timeout or internal
		 server error (or something related)`)

	maxConnectionRetryAttempts := flag.Uint64("max-conn-retry-attempts", 5,
		`Maximum connection retry attempts is the maximum number of times the current node
		 tries to connect to some specified remote node before giving up`)

	snapshotPath := flag.String("snapshot-path", ".",
		`Snapshot path is the directory where snapshot information and related metadata is kept`)

	flag.Parse()
	isConfigFileSpecified := len(*configFile) > 0
	if isConfigFileSpecified {
		return parseConfigurationFile(*configFile)
	}

	config := &node.Config{
		NodeID:                     *nodeID,
		APIPort:                    uint32(*apiPort),
		RPCPort:                    uint32(*rpcPort),
		WriteAheadLogEntryPath:     *writeAheadLogEntryPath,
		WriteAheadLogMetadataPath:  *writeAheadLogMetadataPath,
		RaftStatePath:              *raftStatePath,
		JoinMode:                   *joinMode,
		ClusterConfigPath:          *clusterConfigPath,
		ElectionTimeoutInMillis:    *electionTimeout,
		HeartbeatIntervalInMillis:  *heartbeatInterval,
		RPCTimeoutInMillis:         *rpcTimeout,
		APITimeoutInMillis:         *apiTimeout,
		APIFwdTimeoutInMillis:      *apiFwdTimeout,
		MaxConnectionRetryAttempts: uint32(*maxConnectionRetryAttempts),
		SnapshotPath:               *snapshotPath,
	}
	return config, nil
}

// parseConfigurationFile parses the configuration file and gets configuration.
// If there is any error then it is returned
func parseConfigurationFile(configFile string) (*node.Config, error) {
	configBytes, readErr := ioutil.ReadFile(configFile)
	if readErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: readErr.Error(),
			logfield.Component:   mainComponent,
			logfield.Event:       "READ-CONFIG-FILE",
		}).Error("error while reading config file")
		return nil, readErr
	}
	var config node.Config
	if unmarshalErr := json.Unmarshal(configBytes, &config); unmarshalErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: unmarshalErr.Error(),
			logfield.Component:   mainComponent,
			logfield.Event:       "JSON-PARSE-CONFIG-FILE",
		}).Error("error while parsing config file")
	}
	return &config, nil
}

func setupGracefulShutdown(context *node.Context) {
	gracefulStop := make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	receivedSignal := <-gracefulStop
	logrus.WithFields(logrus.Fields{
		logfield.Component: mainComponent,
		logfield.Event:     "SHUTDOWN",
	}).Infof("Received signal %d. Shutting down", receivedSignal)
	context.Destroy()
}
