//go:build linux

package sshfs

import (
	"os/exec"

	"github.com/hsldymq/gomount/internal/config"
)

func (d *Driver) buildUnmountCommand(entry *config.MountEntry) *exec.Cmd {
	return exec.Command("fusermount", "-u", entry.MountDirPath)
}
