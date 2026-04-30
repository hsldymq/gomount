package webdav

import (
	"os"
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
		confFile      string
		expectURL     string
		expectOptions []string
	}{
		{
			name: "basic webdav",
			entry: &config.MountEntry{
				Name:         "test",
				WebDAV:       &config.WebDAVConfig{URL: "https://cloud.example.com/dav"},
				MountDirPath: "/mnt/webdav",
			},
			confFile:      "",
			expectURL:     "https://cloud.example.com/dav",
			expectOptions: []string{},
		},
		{
			name: "webdav with conf",
			entry: &config.MountEntry{
				Name:         "test",
				WebDAV:       &config.WebDAVConfig{URL: "https://cloud.example.com/dav", Username: "user", Password: "secret"},
				MountDirPath: "/mnt/webdav",
			},
			confFile:      "/tmp/test.conf",
			expectURL:     "https://cloud.example.com/dav",
			expectOptions: []string{"conf=/tmp/test.conf"},
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
			confFile:      "",
			expectURL:     "https://cloud.example.com/dav",
			expectOptions: []string{"file_mode=0644", "dir_mode=0755"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := d.buildMountCommand(tt.entry, tt.confFile)

			if cmd == nil {
				t.Fatal("expected command, got nil")
			}

			if !strings.HasSuffix(cmd.Path, "mount.davfs") {
				t.Errorf("expected path to end with 'mount.davfs', got '%s'", cmd.Path)
			}

			argsStr := strings.Join(cmd.Args, " ")

			if !strings.Contains(argsStr, tt.expectURL) {
				t.Errorf("expected URL '%s' in args, got: %s", tt.expectURL, argsStr)
			}

			if !strings.Contains(argsStr, tt.entry.MountDirPath) {
				t.Errorf("expected mount path '%s' in args, got: %s", tt.entry.MountDirPath, argsStr)
			}

			for _, opt := range tt.expectOptions {
				if !strings.Contains(argsStr, opt) {
					t.Errorf("expected option containing '%s' in args, got: %s", opt, argsStr)
				}
			}
		})
	}
}

func TestDriver_createCredentialFiles(t *testing.T) {
	d := NewDriver()

	t.Run("no credentials", func(t *testing.T) {
		entry := &config.MountEntry{
			WebDAV: &config.WebDAVConfig{URL: "https://example.com/dav"},
		}
		confFile, secretsPath, cleanup, err := d.createCredentialFiles(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if confFile != "" {
			t.Error("expected empty confFile when no credentials")
		}
		if secretsPath != "" {
			t.Error("expected empty secretsPath when no credentials")
		}
		if cleanup != nil {
			t.Error("expected nil cleanup when no credentials")
		}
	})

	t.Run("with credentials", func(t *testing.T) {
		entry := &config.MountEntry{
			WebDAV: &config.WebDAVConfig{
				URL:      "https://example.com/dav",
				Username: "testuser",
				Password: "testpass",
			},
		}
		confFile, _, cleanup, err := d.createCredentialFiles(entry)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer cleanup()

		if confFile == "" {
			t.Fatal("expected conf file path")
		}

		if _, err := os.Stat(confFile); os.IsNotExist(err) {
			t.Error("conf file was not created")
		}

		confContent, err := os.ReadFile(confFile)
		if err != nil {
			t.Fatalf("failed to read conf file: %v", err)
		}
		if !strings.Contains(string(confContent), "secrets ") {
			t.Errorf("conf file should contain secrets path, got: %s", string(confContent))
		}
	})
}
