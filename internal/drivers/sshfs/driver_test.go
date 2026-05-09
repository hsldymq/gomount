package sshfs

import (
	"strings"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestDriver_Type(t *testing.T) {
	d := NewDriver(nil)
	if d.Type() != "sshfs" {
		t.Errorf("expected type 'sshfs', got '%s'", d.Type())
	}
}

func TestDriver_Validate(t *testing.T) {
	d := NewDriver(nil)

	tests := []struct {
		name    string
		entry   *config.MountEntry
		wantErr bool
	}{
		{
			name: "valid entry",
			entry: &config.MountEntry{
				Name:  "test",
				SSHFS: &config.SSHFSConfig{Host: "example.com", RemotePath: "/home/user"},
			},
			wantErr: false,
		},
		{
			name: "valid entry with ssh config alias",
			entry: &config.MountEntry{
				Name:  "test",
				SSHFS: &config.SSHFSConfig{Host: "my-server", RemotePath: "/home/user"},
			},
			wantErr: false,
		},
		{
			name: "missing sshfs config",
			entry: &config.MountEntry{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "missing sshfs host",
			entry: &config.MountEntry{
				Name:  "test",
				SSHFS: &config.SSHFSConfig{RemotePath: "/home/user"},
			},
			wantErr: true,
		},
		{
			name: "missing remote path",
			entry: &config.MountEntry{
				Name:  "test",
				SSHFS: &config.SSHFSConfig{Host: "example.com"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := d.Validate(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDriver_buildMountCommand(t *testing.T) {
	d := NewDriver(nil)

	tests := []struct {
		name         string
		entry        *config.MountEntry
		expectRemote string
		expectOpts   []string
	}{
		{
			name: "basic sshfs",
			entry: &config.MountEntry{
				Name:         "test",
				SSHFS:        &config.SSHFSConfig{Host: "example.com", RemotePath: "/home/user/data"},
				MountDirPath: "/mnt/test",
			},
			expectRemote: "example.com:/home/user/data",
			expectOpts:   []string{"follow_symlinks", "cache_timeout=600"},
		},
		{
			name: "ssh config alias",
			entry: &config.MountEntry{
				Name:         "test",
				SSHFS:        &config.SSHFSConfig{Host: "my-server", RemotePath: "/data"},
				MountDirPath: "/mnt/test",
			},
			expectRemote: "my-server:/data",
			expectOpts:   []string{"follow_symlinks", "cache_timeout=600"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := d.buildMountCommand(tt.entry)

			if cmd == nil {
				t.Fatal("expected command, got nil")
			}

			if !strings.HasSuffix(cmd.Path, "sshfs") {
				t.Errorf("expected path to end with 'sshfs', got '%s'", cmd.Path)
			}

			argsStr := strings.Join(cmd.Args, " ")

			if !strings.Contains(argsStr, tt.expectRemote) {
				t.Errorf("expected remote path '%s' in args, got: %s", tt.expectRemote, argsStr)
			}

			if !strings.Contains(argsStr, tt.entry.MountDirPath) {
				t.Errorf("expected mount path '%s' in args, got: %s", tt.entry.MountDirPath, argsStr)
			}

			for _, opt := range tt.expectOpts {
				if !strings.Contains(argsStr, opt) {
					t.Errorf("expected option '%s' in args, got: %s", opt, argsStr)
				}
			}
		})
	}
}
