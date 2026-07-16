package daemon

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

type Server struct {
	sessions *SessionManager
	mux      *http.ServeMux
	shutdown func()
}

func NewServer(sessions *SessionManager) *Server {
	return NewServerWithShutdown(sessions, nil)
}

func NewServerWithShutdown(sessions *SessionManager, shutdown func()) *Server {
	s := &Server{sessions: sessions, mux: http.NewServeMux(), shutdown: shutdown}
	s.mux.HandleFunc("/v1/health", s.handleHealth)
	s.mux.HandleFunc("/v1/status", s.handleStatus)
	s.mux.HandleFunc("/v1/mount", s.handleMount)
	s.mux.HandleFunc("/v1/unmount", s.handleUnmount)
	s.mux.HandleFunc("/v1/shutdown", s.handleShutdown)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	writeJSON(w, http.StatusOK, daemonapi.HealthResponse{
		OK:              true,
		Version:         "dev",
		PID:             os.Getpid(),
		ManagedTypes:    s.sessions.ManagedTypes(),
		MountedSessions: s.sessions.MountedSessionCount(),
	})
}

func (s *Server) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	unmounted, errs := s.sessions.UnmountAll()
	resp := daemonapi.ShutdownResponse{OK: len(errs) == 0, Unmounted: unmounted, Errors: errs}
	writeJSON(w, http.StatusOK, resp)
	if resp.OK && s.shutdown != nil {
		s.shutdown()
	}
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req daemonapi.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err.Error())
		return
	}
	resp := daemonapi.StatusResponse{Statuses: make([]daemonapi.OperationResult, 0, len(req.Entries))}
	for _, entry := range req.Entries {
		resp.Statuses = append(resp.Statuses, s.sessions.Status(entry))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleMount(w http.ResponseWriter, r *http.Request) {
	s.handleOperation(w, r, s.sessions.Mount)
}

func (s *Server) handleUnmount(w http.ResponseWriter, r *http.Request) {
	s.handleOperation(w, r, s.sessions.Unmount)
}

func (s *Server) handleOperation(w http.ResponseWriter, r *http.Request, op func(daemonapi.EntrySnapshot) daemonapi.OperationResult) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed", "")
		return
	}
	var req daemonapi.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON request", err.Error())
		return
	}
	resp := daemonapi.BatchResponse{Results: make([]daemonapi.OperationResult, 0, len(req.Entries))}
	for _, entry := range req.Entries {
		resp.Results = append(resp.Results, op(entry))
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message, detail string) {
	writeJSON(w, status, struct {
		Error daemonapi.ErrorPayload `json:"error"`
	}{Error: daemonapi.ErrorPayload{Code: code, Message: message, Detail: detail}})
}
