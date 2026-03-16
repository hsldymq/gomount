package sshfs

import (
	"strings"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestDriver_Type(t *testing.T) {
	d := NewDriver()
	if d.Type() != "sshfs" {
		t.Errorf("expected type 'sshfs', got '%s'", d.Type())
	}
}

func TestDriver_Validate(t *testing.T) {
	d := NewDriver()

	tests := []struct {
		name    string
		entry   *config.MountEntry
		wantErr bool
	}{
		{
			name: "valid entry with key",
			entry: &config.MountEntry{
				Name:       "test",
				SSH:        &config.SSHConfig{Host: "example.com", User: "user"},
				RemotePath: "/home/user",
			},
			wantErr: false,
		},
		{
			name: "valid entry with password",
			entry: &config.MountEntry{
				Name:       "test",
				SSH:        &config.SSHConfig{Host: "example.com", User: "user", Password: "pass"},
				RemotePath: "/home/user",
			},
			wantErr: false,
		},
		{
			name: "missing ssh config",
			entry: &config.MountEntry{
				Name:       "test",
				RemotePath: "/home/user",
			},
			wantErr: true,
		},
		{
			name: "missing ssh host",
			entry: &config.MountEntry{
				Name:       "test",
				SSH:        &config.SSHConfig{User: "user"},
				RemotePath: "/home/user",
			},
			wantErr: true,
		},
		{
			name: "missing ssh user",
			entry: &config.MountEntry{
				Name:       "test",
				SSH:        &config.SSHConfig{Host: "example.com"},
				RemotePath: "/home/user",
			},
			wantErr: true,
		},
		{
			name: "missing remote path",
			entry: &config.MountEntry{
				Name: "test",
				SSH:  &config.SSHConfig{Host: "example.com", User: "user"},
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
	d := NewDriver()

	tests := []struct {
		name          string
		entry         *config.MountEntry
		expectRemote  string
		expectOptions []string
	}{
		{
			name: "basic sshfs",
			entry: &config.MountEntry{
				Name:         "test",
				SSH:          &config.SSHConfig{Host: "example.com", User: "user"},
				RemotePath:   "/home/user/data",
				MountDirPath: "/mnt/test",
			},
			expectRemote:  "user@example.com:/home/user/data",
			expectOptions: []string{"follow_symlinks", "cache_timeout=600", " reconnect"},
		},
		{
			name: "sshfs with custom port",
			entry: &config.MountEntry{
				Name:         "test",
				SSH:          &config.SSHConfig{Host: "example.com", User: "user", Port: 2222},
				RemotePath:   "/data",
				MountDirPath: "/mnt/test",
			},
			expectRemote:  "user@example.com:/data",
			expectOptions: []string{"port=2222", "follow_symlinks", "cache_timeout=600", " reconnect"},
		},
		{
			name: "sshfs with key file",
			entry: &config.MountEntry{
				Name:         "test",
				SSH:          &config.SSHConfig{Host: "example.com", User: "user", KeyFile: "~/.ssh/id_rsa"},
				RemotePath:   "/data",
				MountDirPath: "/mnt/test",
			},
			expectRemote:  "user@example.com:/data",
			expectOptions: []string{"IdentityFile=~/.ssh/id_rsa", "follow_symlinks", "cache_timeout=600", " reconnect"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := d.buildMountCommand(tt.entry)

			if cmd == nil {
				t.Fatal("expected command, got nil")
			}

			// 验证命令路径
			if !strings.HasSuffix(cmd.Path, "sshfs") {
				t.Errorf("expected path to end with 'sshfs', got '%s'", cmd.Path)
			}

			// 验证参数
			argsStr := strings.Join(cmd.Args, " ")

			// 检查远程路径
			if !strings.Contains(argsStr, tt.expectRemote) {
				t.Errorf("expected remote path '%s' in args, got: %s", tt.expectRemote, argsStr)
			}

			// 检查挂载路径
			if !strings.Contains(argsStr, tt.entry.MountDirPath) {
				t.Errorf("expected mount path '%s' in args, got: %s", tt.entry.MountDirPath, argsStr)
			}

			// 检查选项
			for _, opt := range tt.expectOptions {
				if !strings.Contains(argsStr, opt) {
					t.Errorf("expected option '%s' in args, got: %s", opt, argsStr)
				}
			}
		})
	}
}
