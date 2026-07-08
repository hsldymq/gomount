//go:build darwin

package smb

import (
	"strings"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestDriver_DarwinBuildMountCommandUsesMountSMBFS(t *testing.T) {
	d := NewDriver()
	entry := &config.MountEntry{
		SMB:          &config.SMBConfig{Addr: "nas.local", ShareName: "shared folder", Username: "user", Password: "p@ss"},
		MountDirPath: "/Users/me/Mounts/nas",
	}

	cmd := d.buildMountCommand(entry, "/tmp/ignored", entry.SMB.Addr, entry.SMB.GetPort())
	if !strings.HasSuffix(cmd.Path, "mount_smbfs") {
		t.Fatalf("expected mount_smbfs, got %s", cmd.Path)
	}
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "//user:p%40ss@nas.local/shared%20folder") {
		t.Fatalf("expected encoded SMB URL with password, got %s", args)
	}
	if !strings.Contains(args, "/Users/me/Mounts/nas") {
		t.Fatalf("expected mount path, got %s", args)
	}
	if d.NeedsSudo() {
		t.Fatal("darwin SMB should not require sudo by default")
	}
}

func TestDriver_DarwinBuildUnmountCommandUsesUmount(t *testing.T) {
	d := NewDriver()
	entry := &config.MountEntry{MountDirPath: "/Users/me/Mounts/nas"}

	cmd := d.buildUnmountCommand(entry)
	if !strings.HasSuffix(cmd.Path, "umount") {
		t.Fatalf("expected umount, got %s", cmd.Path)
	}
	if strings.Join(cmd.Args, " ") != "umount /Users/me/Mounts/nas" {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
}
