package webdav

import (
	"strings"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestDriver_Type(t *testing.T) {
	d := NewDriver()
	if d.Type() != "webdav" {
		t.Errorf("expected type 'webdav', got '%s'", d.Type())
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
			name: "valid entry",
			entry: &config.MountEntry{
				Name:   "test",
				WebDAV: &config.WebDAVConfig{URL: "https://example.com/dav"},
			},
			wantErr: false,
		},
		{
			name: "valid entry with auth",
			entry: &config.MountEntry{
				Name:   "test",
				WebDAV: &config.WebDAVConfig{URL: "https://example.com/dav", Username: "user", Password: "pass"},
			},
			wantErr: false,
		},
		{
			name: "missing webdav config",
			entry: &config.MountEntry{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "missing webdav url",
			entry: &config.MountEntry{
				Name:   "test",
				WebDAV: &config.WebDAVConfig{},
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
		expectURL     string
		expectOptions []string
		expectEnvVars []string
	}{
		{
			name: "basic webdav",
			entry: &config.MountEntry{
				Name:         "test",
				WebDAV:       &config.WebDAVConfig{URL: "https://cloud.example.com/dav"},
				MountDirPath: "/mnt/webdav",
			},
			expectURL:     "https://cloud.example.com/dav",
			expectOptions: []string{},
			expectEnvVars: []string{},
		},
		{
			name: "webdav with auth",
			entry: &config.MountEntry{
				Name:         "test",
				WebDAV:       &config.WebDAVConfig{URL: "https://cloud.example.com/dav", Username: "user", Password: "secret"},
				MountDirPath: "/mnt/webdav",
			},
			expectURL:     "https://cloud.example.com/dav",
			expectOptions: []string{},
			expectEnvVars: []string{"WEBDAV_USERNAME=user", "WEBDAV_PASSWORD=secret"},
		},
		{
			name: "webdav with custom options",
			entry: &config.MountEntry{
				Name:         "test",
				WebDAV:       &config.WebDAVConfig{URL: "https://cloud.example.com/dav"},
				MountDirPath: "/mnt/webdav",
				Options: map[string]interface{}{
					"file_mode": "0644",
					"dir_mode":  "0755",
				},
			},
			expectURL:     "https://cloud.example.com/dav",
			expectOptions: []string{"file_mode=0644", "dir_mode=0755"},
			expectEnvVars: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := d.buildMountCommand(tt.entry)

			if cmd == nil {
				t.Fatal("expected command, got nil")
			}

			// 验证命令路径
			if !strings.HasSuffix(cmd.Path, "mount.davfs") {
				t.Errorf("expected path to end with 'mount.davfs', got '%s'", cmd.Path)
			}

			// 验证参数
			argsStr := strings.Join(cmd.Args, " ")

			// 检查URL
			if !strings.Contains(argsStr, tt.expectURL) {
				t.Errorf("expected URL '%s' in args, got: %s", tt.expectURL, argsStr)
			}

			// 检查挂载路径
			if !strings.Contains(argsStr, tt.entry.MountDirPath) {
				t.Errorf("expected mount path '%s' in args, got: %s", tt.entry.MountDirPath, argsStr)
			}

			// 检查选项
			for _, opt := range tt.expectOptions {
				if !strings.Contains(argsStr, opt) {
					t.Errorf("expected option containing '%s' in args, got: %s", opt, argsStr)
				}
			}

			// 检查环境变量
			envStr := strings.Join(cmd.Env, " ")
			for _, env := range tt.expectEnvVars {
				if !strings.Contains(envStr, env) {
					t.Errorf("expected env var '%s' in environment, got: %s", env, envStr)
				}
			}
		})
	}
}
