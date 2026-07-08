//go:build linux

package smb

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hsldymq/gomount/internal/config"
)

func (d *Driver) NeedsSudo() bool {
	return true
}

func (d *Driver) buildMountCommand(entry *config.MountEntry, credsFile, addr string, port int) *exec.Cmd {
	smbAddr := fmt.Sprintf("//%s/%s", addr, entry.SMB.ShareName)

	options := fmt.Sprintf("credentials=%s,port=%d,file_mode=0755,dir_mode=0755,uid=%d,gid=%d",
		credsFile,
		port,
		os.Getuid(),
		os.Getgid(),
	)

	args := []string{
		smbAddr,
		entry.MountDirPath,
		"-o", options,
	}

	return exec.Command("mount.cifs", args...)
}

func (d *Driver) buildUnmountCommand(entry *config.MountEntry) *exec.Cmd {
	return exec.Command("umount", entry.MountDirPath)
}
