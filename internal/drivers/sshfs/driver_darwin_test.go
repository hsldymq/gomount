//go:build darwin

package sshfs

import (
	"strings"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestDriver_DarwinBuildUnmountCommandUsesUmount(t *testing.T) {
	d := NewDriver()
	entry := &config.MountEntry{MountDirPath: "/Users/me/Mounts/dev"}

	cmd := d.buildUnmountCommand(entry)
	if !strings.HasSuffix(cmd.Path, "umount") {
		t.Fatalf("expected umount, got %s", cmd.Path)
	}
	if strings.Join(cmd.Args, " ") != "umount /Users/me/Mounts/dev" {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
}
