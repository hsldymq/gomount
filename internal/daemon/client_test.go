package daemon

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

func TestUnixClientHealth(t *testing.T) {
	socketPath := tempSocketPath(t)
	server := &http.Server{Handler: NewServer(NewSessionManager(&fakeMounter{}))}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	go server.Serve(ln)
	t.Cleanup(func() { _ = server.Close() })

	client := NewClient(socketPath)
	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if !health.OK || len(health.ManagedTypes) != 1 || health.ManagedTypes[0] != "webdav" {
		t.Fatalf("unexpected health: %+v", health)
	}
}

func TestUnixClientMount(t *testing.T) {
	socketPath := tempSocketPath(t)
	server := &http.Server{Handler: NewServer(NewSessionManager(&fakeMounter{}))}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	go server.Serve(ln)
	t.Cleanup(func() { _ = server.Close() })

	client := NewClient(socketPath)
	resp, err := client.Mount(context.Background(), []daemonapi.EntrySnapshot{{
		Name:         "docs",
		Type:         "webdav",
		MountDirPath: "/mnt/docs",
		Source:       daemonapi.Source{URL: "https://cloud.example.com/dav"},
	}})
	if err != nil {
		t.Fatalf("Mount returned error: %v", err)
	}
	if len(resp.Results) != 1 || !resp.Results[0].Success || !resp.Results[0].Mounted {
		t.Fatalf("unexpected mount response: %+v", resp)
	}
}

func TestUnixClientShutdown(t *testing.T) {
	socketPath := tempSocketPath(t)
	server := &http.Server{Handler: NewServerWithShutdown(NewSessionManager(&fakeMounter{}), func() {})}
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("listen unix: %v", err)
	}
	go server.Serve(ln)
	t.Cleanup(func() { _ = server.Close() })

	client := NewClient(socketPath)
	resp, err := client.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown returned error: %v", err)
	}
	if !resp.OK {
		t.Fatalf("unexpected shutdown response: %+v", resp)
	}
}

func tempSocketPath(t *testing.T) string {
	t.Helper()
	return t.TempDir() + "/gomount.sock"
}
