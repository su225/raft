package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/su225/raft/logfield"
	"github.com/su225/raft/node/cluster"
	"github.com/su225/raft/node/datastore"
	"github.com/su225/raft/node/state"
)

const apiServer = "API-SERVER"

// APIServer represents the REST server which is
// responsible for receiving requests from the user
type APIServer struct {
	// Port on which the API server listens
	// It must be different from RPC port - the
	// one used to receive protocol messages
	APIPort uint32

	// CurrentNodeID is the identifier of the node
	// on which the server is running
	CurrentNodeID string

	// APITimeout and APIForwardTimeout represent
	// timeout related to handling user requests and
	// the forwarded ones
	APITimeout        int64
	APIForwardTimeout int64

	// RaftStateManager is needed to lookup the current
	// role of the node. If it is not the leader and the
	// leader is known then it is forwarded to the leader.
	// Otherwise it will return error with 5XX indicating
	// some error on the server side. The cluster will be
	// down during leader election.
	state.RaftStateManager

	// MembershipManager is needed to lookup the URL of
	// the API server of the node in case request is
	// forwarded to it.
	cluster.MembershipManager

	// Server represents the HTTP server
	*http.Server

	// DataStore provides the API to access Raft backed
	// key-value storage
	datastore.DataStore
}

// NewAPIServer creates a new instance of APIServer and returns it.
// Note that it does not start listening on specified port yet.
func NewAPIServer(
	apiPort uint32,
	currentNodeID string,
	apiTimeout int64,
	apiFwdTimeout int64,
	stateMgr state.RaftStateManager,
	membershipMgr cluster.MembershipManager,
	dataStore datastore.DataStore,
) *APIServer {
	return &APIServer{
		APIPort:           apiPort,
		CurrentNodeID:     currentNodeID,
		APITimeout:        apiTimeout,
		APIForwardTimeout: apiFwdTimeout,
		RaftStateManager:  stateMgr,
		MembershipManager: membershipMgr,
		DataStore:         dataStore,
	}
}

// Start starts the API server listening at the given port for
// given methods. If there is an issue while starting it is returned
func (s *APIServer) Start() error {
	r := s.setupRESTRoutes()
	s.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.APIPort),
		Handler: r,
	}
	go func() {
		logrus.WithFields(logrus.Fields{
			logfield.Component: apiServer,
			logfield.Event:     "START-REST-SERVER",
		}).Debugf("starting REST server at %s", s.Server.Addr)

		if err := s.Server.ListenAndServe(); err != http.ErrServerClosed {
			logrus.WithFields(logrus.Fields{
				logfield.ErrorReason: err.Error(),
				logfield.Component:   apiServer,
				logfield.Event:       "LISTEN-AND-SERVE",
			}).Errorf("error while starting/shutting down REST server")
		}
	}()
	return nil
}

// setupRESTRoutes sets up the routes for the REST server.
func (s *APIServer) setupRESTRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/v1/data", s.addOrUpdateKey).Methods("POST")
	r.HandleFunc("/v1/data/{key}", s.getKey).Methods("GET")
	r.HandleFunc("/v1/data/{key}", s.deleteKey).Methods("DELETE")
	return r
}

// Destroy kills the HTTP server within the given timeout. If it doesn't
// happen then it returns the error complaining the same.
func (s *APIServer) Destroy() error {
	shutdownCtx, shutdownCancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancelFunc()
	logrus.WithFields(logrus.Fields{
		logfield.Component: apiServer,
		logfield.Event:     "DESTROY",
	}).Debugf("shutting down REST server")
	if shutdownErr := s.Server.Shutdown(shutdownCtx); shutdownErr != nil {
		return shutdownErr
	}
	return nil
}

type handleOrForwardAction uint8

const (
	handle handleOrForwardAction = iota
	forward
	abort
)

// addOrUpdateKey adds a new key-value pair or updates the existing one. If the current
// node is not the leader then it is forwarded to the leader node if it knows it or returns
// internal server error. If the operation is successful then it returns OK. If the body is
// not in the proper format then it returns Bad Request.
func (s *APIServer) addOrUpdateKey(w http.ResponseWriter, r *http.Request) {
	s.handleRequest(w, r, "UPSERT-KEY", "upsert", s.handleUpsertRequest)
}

// getKey tries to retrieve the value of key-value pair. If it exists the value is returned.
// If it does not then 404 - Not found is returned. If there is some error on the server side
// where the leader is not known to this node then Internal server error is returned.
func (s *APIServer) getKey(w http.ResponseWriter, r *http.Request) {
	s.handleRequest(w, r, "GET-KEY", "get", s.handleGetRequest)
}

// deleteKey tries to delete a key-value pair given the key. If the operation is successful, that
// is the key exists and is deleted or the key does not exist and hence a no-op then 200 OK is
// returned. If there is an issue on the server side then Internal Server Error is returned.
func (s *APIServer) deleteKey(w http.ResponseWriter, r *http.Request) {
	s.handleRequest(w, r, "DELETE-KEY", "delete", s.handleDeleteRequest)
}

// handleUpsertRequest handles the upsert request.
func (s *APIServer) handleUpsertRequest(w http.ResponseWriter, r *http.Request) {
	kvBytes, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: err.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "UPSERT-READ-BODY",
		}).Errorf("error while reading request body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var kvPair datastore.KVPair
	if unmarshalErr := json.Unmarshal(kvBytes, &kvPair); unmarshalErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: unmarshalErr.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "UPSERT-BODY-UNMARSHAL",
		}).Errorf("error while unmarshaling request [JSON] body")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if putErr := s.DataStore.PutData(kvPair.Key, kvPair.Value); putErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: putErr.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "UPSERT-DS-PUT",
		}).Errorf("error while upserting data into the store")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleDeleteRequest handles the get request.
func (s *APIServer) handleGetRequest(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	key, keyPresent := pathParams["key"]
	if !keyPresent {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	value, retrieveErr := s.DataStore.GetData(key)
	if retrieveErr != nil {
		// TODO: Examine the type of retrieveErr and write
		// responses accordingly. If the error is due to the
		// fact that the given key doesn't exist then it must
		// 404 - StatusNotFound instead of 500 - StatusInternalServerError
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	kvPair := datastore.KVPair{Key: key, Value: value}
	marshaledKVPair, marshalErr := json.Marshal(kvPair)
	if marshalErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: marshalErr.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "GET-MARSHAL-KV",
		}).Errorf("error while marshaling key-value pair")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Write(marshaledKVPair)
	w.WriteHeader(http.StatusOK)
}

// handleDeleteRequest handles the delete request
func (s *APIServer) handleDeleteRequest(w http.ResponseWriter, r *http.Request) {
	pathParams := mux.Vars(r)
	key, keyPresent := pathParams["key"]
	if !keyPresent {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	delErr := s.DataStore.DeleteData(key)
	if delErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: delErr.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "DEL-KV-PAIR",
		}).Errorf("error while deleting key")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleRequest provides the standard way of handling incoming request. First
// it logs certain attributes of the request, then determines if the request should
// be forwarded, aborted or handled at this node and performs appropriate operations
func (s *APIServer) handleRequest(
	w http.ResponseWriter, r *http.Request,
	eventName, requestName string,
	handlerFunc func(w http.ResponseWriter, r *http.Request),
) {
	logrus.WithFields(logrus.Fields{
		logfield.Component:          apiServer,
		logfield.Event:              eventName,
		logfield.RequesterIPAddress: r.RemoteAddr,
		logfield.RESTMethod:         r.Method,
		logfield.RequestURI:         r.RequestURI,
	}).Infof("received %s request", requestName)

	currentRaftState := s.RaftStateManager.GetRaftState()
	switch s.determineHandlingAction(&currentRaftState) {
	case abort:
		w.WriteHeader(http.StatusInternalServerError)
	case forward:
		s.forwardRequest(currentRaftState.CurrentLeader, w, r)
	case handle:
		handlerFunc(w, r)
	}
}

// determineHandlingAction determines whether the request must be handled at this node or
// should be forwarded to some other node or should simply return because this node cannot
// handle and it doesn't know where to forward.
func (s *APIServer) determineHandlingAction(currentRaftState *state.RaftState) handleOrForwardAction {
	if currentRaftState.CurrentRole != state.RoleLeader {
		if currentRaftState.CurrentLeader == "" {
			return abort
		}
		return forward
	}
	return handle
}

// forwardRequest forwards the request to the leader node
func (s *APIServer) forwardRequest(leaderNodeID string, w http.ResponseWriter, r *http.Request) {
	leaderNodeInfo, lookupErr := s.MembershipManager.GetNode(leaderNodeID)
	if lookupErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: lookupErr.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "PROXY-FWD",
		}).Errorf("error while forwarding to %s", leaderNodeID)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	// TODO: This is hacky. Find a better way of doing it
	forwardURL := fmt.Sprintf("http://%s%s", leaderNodeInfo.APIURL, r.URL.Path)
	logrus.WithFields(logrus.Fields{
		logfield.Component: apiServer,
		logfield.Event:     "PROXY-FWD",
	}).Debugf("forwarding request to %s at %s", leaderNodeID, forwardURL)

	fwdRequest, requestErr := http.NewRequest(r.Method, forwardURL, r.Body)
	if requestErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: requestErr.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "PROXY-REQ-CREATE",
		}).Errorf("error while creating request to forward")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Copy headers from source request to the request to be forwarded
	srcHeaders, dstHeaders := r.Header, fwdRequest.Header
	s.copyHTTPRequestHeaders(&dstHeaders, &srcHeaders)
	fwdRequest.Header.Set("X-Leader-Forward-From", s.CurrentNodeID)

	// Now send the request to the remote node and obtain the response
	httpClient := http.Client{
		Timeout: time.Duration(s.APIForwardTimeout) * time.Millisecond,
	}
	response, respErr := httpClient.Do(fwdRequest)
	if respErr != nil {
		logrus.WithFields(logrus.Fields{
			logfield.ErrorReason: respErr.Error(),
			logfield.Component:   apiServer,
			logfield.Event:       "FWD-REQUEST",
		}).Errorf("error while sending request or getting response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	s.replyFromResponse(w, response)
}

// replyFromResponse sends the response to the client derived from the upstream response
func (s *APIServer) replyFromResponse(w http.ResponseWriter, response *http.Response) {
	srcHeaders, dstHeaders := response.Header, w.Header()
	s.copyHTTPRequestHeaders(&dstHeaders, &srcHeaders)
	w.WriteHeader(response.StatusCode)
	io.Copy(w, response.Body)
}

// copyHTTPRequestHeaders copies HTTP request headers from source (src) to destination (dst)
func (s *APIServer) copyHTTPRequestHeaders(dst *http.Header, src *http.Header) {
	for headerKey, headerVals := range *src {
		for _, headerVal := range headerVals {
			dst.Set(headerKey, headerVal)
		}
	}
}
