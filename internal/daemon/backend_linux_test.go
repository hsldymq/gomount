//go:build linux

package daemon

import (
	"testing"

	"github.com/rclone/rclone/fs/config/configmap"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

func TestBackendConfigWithDefaultsIncludesS3ChunkSize(t *testing.T) {
	mapper, err := backendConfigWithDefaults("s3", configmap.Simple{"provider": "Alibaba"})
	if err != nil {
		t.Fatalf("backendConfigWithDefaults() error = %v", err)
	}
	if got, ok := mapper.Get("chunk_size"); !ok || got != "5Mi" {
		t.Fatalf("chunk_size = %q, %v; want rclone default 5Mi", got, ok)
	}
	if got, ok := mapper.Get("provider"); !ok || got != "Alibaba" {
		t.Fatalf("provider override = %q, %v", got, ok)
	}
}

func TestS3BackendValuesResolveProvidersAndRustFSDefaults(t *testing.T) {
	values := s3BackendValues(daemonapi.Source{Provider: "rustfs", Endpoint: "https://rustfs.example.com"})
	if values["provider"] != "Other" || values["region"] != "us-east-1" || values["force_path_style"] != "true" {
		t.Fatalf("unexpected RustFS values: %+v", values)
	}

	values = s3BackendValues(daemonapi.Source{Provider: "Cloudflare"})
	if values["provider"] != "Cloudflare" {
		t.Fatalf("provider was not passed through: %+v", values)
	}
}
