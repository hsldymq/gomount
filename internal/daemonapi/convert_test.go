package daemonapi

import (
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestFromMountEntryConvertsWebDAV(t *testing.T) {
	entry := config.MountEntry{
		Name:         "docs",
		Type:         "webdav",
		MountDirPath: "/mnt/docs",
		WebDAV:       &config.WebDAVConfig{URL: "https://cloud.example.com/dav", Username: "user", Password: "pass", Path: "/team"},
		Options:      map[string]interface{}{"vfs_cache_mode": "writes"},
	}

	snapshot, ok := FromMountEntry(&entry)
	if !ok {
		t.Fatal("expected webdav entry to convert")
	}
	if snapshot.Name != "docs" || snapshot.Type != "webdav" || snapshot.Source.URL != "https://cloud.example.com/dav" || snapshot.Source.Username != "user" || snapshot.Source.Password != "pass" || snapshot.Source.Path != "/team" {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.Options["vfs_cache_mode"] != "writes" {
		t.Fatalf("options not preserved: %+v", snapshot.Options)
	}
}

func TestFromMountEntryReturnsUnmanagedSnapshotForSMB(t *testing.T) {
	entry := config.MountEntry{Name: "nas", Type: "smb", MountDirPath: "/mnt/nas"}

	snapshot, ok := FromMountEntry(&entry)
	if ok {
		t.Fatal("expected smb to be marked unmanaged")
	}
	if snapshot.Name != "nas" || snapshot.Type != "smb" || snapshot.MountDirPath != "/mnt/nas" {
		t.Fatalf("unexpected unmanaged snapshot: %+v", snapshot)
	}
}
