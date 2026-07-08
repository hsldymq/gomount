//go:build darwin

package status

import "testing"

func TestDarwinMountLineMatches(t *testing.T) {
	line := "//user@nas.local/shared on /Users/me/Mounts/nas (smbfs, nodev, nosuid, mounted by me)"

	if !darwinMountLineMatches(line, "/Users/me/Mounts/nas") {
		t.Fatal("expected mount line to match mount path")
	}
	if darwinMountLineMatches(line, "/Users/me/Mounts/other") {
		t.Fatal("expected different mount path not to match")
	}
}
