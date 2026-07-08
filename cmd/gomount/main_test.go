package main

import (
	"runtime"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	sshfsDriver "github.com/hsldymq/gomount/internal/drivers/sshfs"
)

func TestEntriesNeedSudoReturnsFalseWhenSelectedDriversDoNotNeedSudo(t *testing.T) {
	entry := &config.MountEntry{Name: "dev", Type: "sshfs", SSHFS: &config.SSHFSConfig{Host: "dev", RemotePath: "/src"}}
	mgr := drivers.NewManager(&config.Config{Mounts: []config.MountEntry{*entry}})
	mgr.RegisterDriver(sshfsDriver.NewDriver())

	need, err := entriesNeedSudo(mgr, []*config.MountEntry{entry})
	if err != nil {
		t.Fatalf("entriesNeedSudo returned error: %v", err)
	}
	if need {
		t.Fatal("expected sshfs-only selection not to need sudo")
	}
}

func TestEntriesNeedSudoReturnsTrueWhenAnySelectedDriverNeedsSudo(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific SMB sudo behavior")
	}

	entry := &config.MountEntry{Name: "nas", Type: "smb", SMB: &config.SMBConfig{Addr: "nas", ShareName: "data", Username: "me"}}
	mgr := createDriverManager(&config.Config{Mounts: []config.MountEntry{*entry}})

	need, err := entriesNeedSudo(mgr, []*config.MountEntry{entry})
	if err != nil {
		t.Fatalf("entriesNeedSudo returned error: %v", err)
	}
	if !need {
		t.Fatal("expected smb selection to need sudo on Linux")
	}
}
