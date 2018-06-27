package libstream

import (
	"net/http"
	"strconv"
	"encoding/json"
	"io"
	"github.com/gorilla/mux"
	"time"
)

type StreamServer struct {
	Address   string
	APIPrefix string
	RootToken string

	Timer  *time.Timer
	Router *mux.Router
}

type ServerConfig struct {
	address   string
	rootToken string
	apiPrefix string
}

func NewServer(config ServerConfig) *StreamServer {
	if config.apiPrefix == "" {
		config.apiPrefix = "/api/v1/"
	}
	if config.address == "" {
		config.address = "/"
	}
	if config.rootToken == "" {
		config.rootToken = "!csdf!25"
	}

	server := &StreamServer{
		Address:   config.address,
		RootToken: config.rootToken,
		APIPrefix: config.apiPrefix,
	}

	server.SetupRouter()
	return server
}

func (s *StreamServer) SetupRouter() {
	s.Router = mux.NewRouter()
	//s.Router.PathPrefix(s.APIPrefix)
	Logger.Debugf(`API endpoint "%s"`, s.APIPrefix)
}

func (s *StreamServer) Run() {
	Logger.Infof(`Stream server started on "%s"`, s.Address)
	s.Router.PathPrefix("/s").HandlerFunc(s.ShowAllStreams).Methods("GET")
	s.Router.PathPrefix("/run").HandlerFunc(s.StartNewStream).Methods("GET")
	s.Router.PathPrefix("/activate/{id}").HandlerFunc(s.ActivateStream).Methods("PATCH")
	s.Router.PathPrefix("/interrupt/{id}").HandlerFunc(s.InterruptStream).Methods("PATCH")
	s.Router.PathPrefix("/finish/{id}").HandlerFunc(s.FinishStream).Methods("PATCH")
	http.ListenAndServe(s.Address, s.Router)
}

//show all
func (s *StreamServer) ShowAllStreams(w http.ResponseWriter, r *http.Request) {

	pn, _ := strconv.Atoi(r.URL.Query().Get("page[number]"))
	ps, _ := strconv.Atoi(r.URL.Query().Get("page[size]"))

	if allStream, success := SelectAll(pn, ps); success {
		allStreamJSON, _ := json.Marshal(allStream)
		w.WriteHeader(http.StatusOK)
		w.Write(allStreamJSON)
	} else {
		w.WriteHeader(http.StatusNoContent)
		io.WriteString(w, "Parameters under the limit")
	}
}

//start new -- created
func (s *StreamServer) StartNewStream(w http.ResponseWriter, r *http.Request) {
	stream := NewStream()
	Logger.Debug(`New stream created with uuid `, stream.ID)
	if InsertToDB(stream) {
		streamJSON, _ := json.Marshal(s)
		w.WriteHeader(http.StatusCreated)
		w.Write(streamJSON)
	}
}

//set active
func (s *StreamServer) ActivateStream(w http.ResponseWriter, r *http.Request) {
	stream := mux.Vars(r)["id"]
	s.UpdateStream(w, stream, "a")
	//s.Timer.Stop()
}

//set interrupted
func (s *StreamServer) InterruptStream(w http.ResponseWriter, r *http.Request) {
	stream := mux.Vars(r)["id"]
	s.UpdateStream(w, stream, "i")

	go s.finishByTimer(w, stream)
}

//set finished
func (s *StreamServer) FinishStream(w http.ResponseWriter, r *http.Request) {
	stream := mux.Vars(r)["id"]
	s.UpdateStream(w, stream, "f")
}

func (s *StreamServer) UpdateStream(w http.ResponseWriter, streamID string, status string) {

	if sDB, check := CheckFromDB(streamID); check {
		if name, success := sDB.UpdateStatus(status); success {
			w.WriteHeader(http.StatusOK)
			resultString := "Stream status update on " + name
			UpdateRow(sDB)
			io.WriteString(w, resultString)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, "Stream update - error - check stream ID or Status name")
	}
}

func (s *StreamServer) finishByTimer(w http.ResponseWriter, streamID string) {
	<-s.Timer.C
	s.UpdateStream(w, streamID, "f")
}