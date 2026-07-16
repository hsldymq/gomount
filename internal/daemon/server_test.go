package daemon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

func TestServerHealth(t *testing.T) {
	server := NewServer(NewSessionManager(&fakeMounter{}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got daemonapi.HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if !got.OK || got.PID != os.Getpid() || got.MountedSessions != 0 || len(got.ManagedTypes) != 2 || got.ManagedTypes[0] != "aliyun_oss" || got.ManagedTypes[1] != "webdav" {
		t.Fatalf("unexpected health: %+v", got)
	}
}

func TestServerShutdownUnmountsSessionsAndCallsShutdown(t *testing.T) {
	called := false
	mgr := NewSessionManager(&fakeMounter{})
	mgr.Mount(daemonapi.EntrySnapshot{Name: "docs", Type: "webdav", MountDirPath: "/mnt/docs", Source: daemonapi.Source{URL: "https://cloud.example.com/dav"}})
	server := NewServerWithShutdown(mgr, func() { called = true })
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/shutdown", nil)

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got daemonapi.ShutdownResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode shutdown: %v", err)
	}
	if !got.OK || got.Unmounted != 1 || len(got.Errors) != 0 || !called {
		t.Fatalf("unexpected shutdown response called=%v: %+v", called, got)
	}
}

func TestServerShutdownDoesNotStopWhenUnmountFails(t *testing.T) {
	called := false
	mgr := NewSessionManager(&fakeMounter{unmountErr: os.ErrPermission})
	mgr.Mount(daemonapi.EntrySnapshot{Name: "docs", Type: "webdav", MountDirPath: "/mnt/docs", Source: daemonapi.Source{URL: "https://cloud.example.com/dav"}})
	server := NewServerWithShutdown(mgr, func() { called = true })
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/shutdown", nil)

	server.ServeHTTP(rec, req)

	var got daemonapi.ShutdownResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode shutdown: %v", err)
	}
	if got.OK || got.Unmounted != 0 || len(got.Errors) != 1 || called {
		t.Fatalf("expected failed shutdown without callback called=%v: %+v", called, got)
	}
}

func TestServerStatusBatchReturnsManagedAndUnmanaged(t *testing.T) {
	server := NewServer(NewSessionManager(&fakeMounter{}))
	body := daemonapi.BatchRequest{Entries: []daemonapi.EntrySnapshot{
		{Name: "nas", Type: "smb", MountDirPath: "/mnt/nas"},
		{Name: "docs", Type: "webdav", MountDirPath: "/mnt/docs", Source: daemonapi.Source{URL: "https://cloud.example.com/dav"}},
	}}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/status", bytes.NewReader(data))

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got daemonapi.StatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	if len(got.Statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %+v", got)
	}
	if got.Statuses[0].Managed {
		t.Fatalf("expected smb unmanaged, got %+v", got.Statuses[0])
	}
	if !got.Statuses[1].Managed || got.Statuses[1].Mounted {
		t.Fatalf("expected webdav managed and unmounted, got %+v", got.Statuses[1])
	}
}

func TestServerMountBatch(t *testing.T) {
	server := NewServer(NewSessionManager(&fakeMounter{}))
	body := daemonapi.BatchRequest{Entries: []daemonapi.EntrySnapshot{{
		Name:         "docs",
		Type:         "webdav",
		MountDirPath: "/mnt/docs",
		Source:       daemonapi.Source{URL: "https://cloud.example.com/dav", Path: "/docs"},
	}}}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/mount", bytes.NewReader(data))

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got daemonapi.BatchResponse
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode mount: %v", err)
	}
	if len(got.Results) != 1 || !got.Results[0].Success || !got.Results[0].Mounted {
		t.Fatalf("unexpected mount response: %+v", got)
	}
}
