//go:build darwin

package smb

import (
	"net"
	"net/url"
	"os/exec"
	"strconv"

	"github.com/hsldymq/gomount/internal/config"
)

func (d *Driver) NeedsSudo() bool {
	return false
}

func (d *Driver) buildMountCommand(entry *config.MountEntry, _ string, addr string, port int) *exec.Cmd {
	host := addr
	if port != 0 && port != 445 {
		host = net.JoinHostPort(addr, strconv.Itoa(port))
	}

	user := url.User(entry.SMB.Username)
	if entry.SMB.Password != "" {
		user = url.UserPassword(entry.SMB.Username, entry.SMB.Password)
	}

	smbURL := "//" + user.String() + "@" + host + "/" + url.PathEscape(entry.SMB.ShareName)
	return exec.Command("mount_smbfs", smbURL, entry.MountDirPath)
}

func (d *Driver) buildUnmountCommand(entry *config.MountEntry) *exec.Cmd {
	return exec.Command("umount", entry.MountDirPath)
}
