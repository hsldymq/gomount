package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	BaseURL string
	client  *http.Client
}

func NewClient(port int) *Client {
	return &Client{
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d/api/v1", port),
		client:  &http.Client{},
	}
}

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (c *Client) Health() (*HealthResponse, error) {
	data, err := c.doRequest("GET", "/health", nil)
	if err != nil {
		return nil, err
	}
	var resp HealthResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Mount(name string) (*MountResponse, error) {
	data, err := c.doRequest("POST", "/mount", MountRequest{Name: name})
	if err != nil {
		return nil, err
	}
	var resp MountResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Unmount(name string) (*MountResponse, error) {
	data, err := c.doRequest("POST", "/unmount", UnmountRequest{Name: name})
	if err != nil {
		return nil, err
	}
	var resp MountResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) UnmountAll() (*MountResponse, error) {
	data, err := c.doRequest("POST", "/unmount-all", nil)
	if err != nil {
		return nil, err
	}
	var resp MountResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) List() (*ListResponse, error) {
	data, err := c.doRequest("GET", "/list", nil)
	if err != nil {
		return nil, err
	}
	var resp ListResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *Client) Shutdown() error {
	_, err := c.doRequest("POST", "/shutdown", nil)
	return err
}
