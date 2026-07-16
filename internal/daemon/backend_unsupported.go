//go:build !linux && !darwin

package daemon

import (
	"fmt"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

type RcloneMounter struct{}

func NewRcloneMounter() Mounter {
	return &RcloneMounter{}
}

func SupportedManagedTypes() []string {
	return nil
}

func (m *RcloneMounter) Mount(entry daemonapi.EntrySnapshot) (MountSession, error) {
	return nil, fmt.Errorf("embedded rclone managed mounts are not supported on this platform")
}
