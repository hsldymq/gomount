//go:build darwin && (!cgo || !cmount)

package daemon

import (
	"fmt"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

type RcloneMounter struct{}

func NewRcloneMounter() Mounter { return &RcloneMounter{} }

// Keep managed types visible so a regular macOS build returns an actionable
// build error instead of silently treating cloud mounts as unmanaged.
func SupportedManagedTypes() []string { return daemonapi.ManagedTypes() }

func (m *RcloneMounter) Mount(entry daemonapi.EntrySnapshot) (MountSession, error) {
	return nil, fmt.Errorf("macOS cloud mounts require macFUSE and a binary built with CGO_ENABLED=1 and -tags cmount")
}
