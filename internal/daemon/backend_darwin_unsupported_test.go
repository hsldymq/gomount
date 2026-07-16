//go:build darwin && (!cgo || !cmount)

package daemon

import (
	"strings"
	"testing"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

func TestDarwinUnsupportedMounterExplainsCmountBuild(t *testing.T) {
	_, err := NewRcloneMounter().Mount(daemonapi.EntrySnapshot{Type: "aliyun_oss"})
	if err == nil || !strings.Contains(err.Error(), "-tags cmount") {
		t.Fatalf("expected actionable cmount error, got %v", err)
	}
	if len(SupportedManagedTypes()) != 2 {
		t.Fatalf("expected cloud managed types to remain visible, got %v", SupportedManagedTypes())
	}
}
