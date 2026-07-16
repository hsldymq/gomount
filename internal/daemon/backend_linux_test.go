//go:build linux

package daemon

import (
	"testing"

	"github.com/rclone/rclone/fs/config/configmap"
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
