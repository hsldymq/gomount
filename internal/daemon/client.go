package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

type Client struct {
	socketPath string
	httpClient *http.Client
}

func NewClient(socketPath string) *Client {
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &Client{
		socketPath: socketPath,
		httpClient: &http.Client{Transport: transport, Timeout: 30 * time.Second},
	}
}

func DefaultSocketPath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gomount.sock"), nil
}

func DefaultLogPath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "gomount-daemon.log"), nil
}

func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "gomount")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func (c *Client) Health(ctx context.Context) (*daemonapi.HealthResponse, error) {
	var out daemonapi.HealthResponse
	if err := c.get(ctx, "/v1/health", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Mount(ctx context.Context, entries []daemonapi.EntrySnapshot) (*daemonapi.BatchResponse, error) {
	var out daemonapi.BatchResponse
	if err := c.post(ctx, "/v1/mount", daemonapi.BatchRequest{Entries: entries}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Unmount(ctx context.Context, entries []daemonapi.EntrySnapshot) (*daemonapi.BatchResponse, error) {
	var out daemonapi.BatchResponse
	if err := c.post(ctx, "/v1/unmount", daemonapi.BatchRequest{Entries: entries}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Status(ctx context.Context, entries []daemonapi.EntrySnapshot) (*daemonapi.StatusResponse, error) {
	var out daemonapi.StatusResponse
	if err := c.post(ctx, "/v1/status", daemonapi.BatchRequest{Entries: entries}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Shutdown(ctx context.Context) (*daemonapi.ShutdownResponse, error) {
	var out daemonapi.ShutdownResponse
	if err := c.post(ctx, "/v1/shutdown", struct{}{}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://unix"+path, nil)
	if err != nil {
		return err
	}
	return c.do(req, out)
}

func (c *Client) post(ctx context.Context, path string, in any, out any) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unix"+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("daemon returned HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
