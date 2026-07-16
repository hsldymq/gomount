package tui

import (
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestEntryAddrWebDAV(t *testing.T) {
	entry := config.MountEntry{
		Type:   "webdav",
		WebDAV: &config.WebDAVConfig{URL: "https://cloud.example.com/dav", Path: "/team"},
	}

	if got := entryAddr(entry); got != "https://cloud.example.com/dav:/team" {
		t.Fatalf("entryAddr() = %q", got)
	}
}

func TestEntryAddrWebDAVRedactsURLUserinfo(t *testing.T) {
	entry := config.MountEntry{
		Type:   "webdav",
		WebDAV: &config.WebDAVConfig{URL: "https://user:secret@cloud.example.com/dav", Path: "/team"},
	}

	if got := entryAddr(entry); got != "https://xxxxx@cloud.example.com/dav:/team" {
		t.Fatalf("entryAddr() = %q", got)
	}
}
