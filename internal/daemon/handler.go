package daemon

import (
	"encoding/json"
	"net/http"
)

func writeJSON(w http.ResponseWriter, statusCode int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

func handleMount(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req MountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, MountResponse{Success: false, Message: err.Error()})
			return
		}

		ctx := r.Context()
		msg, err := h.Mount(ctx, req.Name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, MountResponse{Success: false, Message: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, MountResponse{Success: true, Message: msg})
	}
}

func handleUnmount(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req UnmountRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, MountResponse{Success: false, Message: err.Error()})
			return
		}

		ctx := r.Context()
		msg, err := h.Unmount(ctx, req.Name)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, MountResponse{Success: false, Message: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, MountResponse{Success: true, Message: msg})
	}
}

func handleUnmountAll(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.UnmountAll()
		writeJSON(w, http.StatusOK, MountResponse{Success: true})
	}
}

func handleList(h *Handlers) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries := h.List()
		writeJSON(w, http.StatusOK, ListResponse{Mounts: entries})
	}
}

func handleShutdown(srv *http.Server) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, ShutdownResponse{Success: true})
		go func() {
			srv.Close()
		}()
	}
}
