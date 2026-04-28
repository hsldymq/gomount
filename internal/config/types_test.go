package config

import (
	"testing"
)

func TestMountEntry_GetMountPath(t *testing.T) {
	tests := []struct {
		name         string
		entry        MountEntry
		expectedPath string
	}{
		{
			name: "with mount_dir_path",
			entry: MountEntry{
				Name:         "test",
				MountDirPath: "/custom/path",
			},
			expectedPath: "/custom/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.entry.GetMountPath()
			if path != tt.expectedPath {
				t.Errorf("expected path '%s', got '%s'", tt.expectedPath, path)
			}
		})
	}
}

func TestSMBConfig_GetPort(t *testing.T) {
	tests := []struct {
		name         string
		port         int
		expectedPort int
	}{
		{"default", 0, 445},
		{"custom", 1445, 1445},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SMBConfig{Port: tt.port}
			if got := cfg.GetPort(); got != tt.expectedPort {
				t.Errorf("GetPort() = %v, want %v", got, tt.expectedPort)
			}
		})
	}
}

func TestMountEntry_HasPassword(t *testing.T) {
	tests := []struct {
		name     string
		entry    MountEntry
		expected bool
	}{
		{"has smb password", MountEntry{SMB: &SMBConfig{Password: "secret"}}, true},
		{"empty smb password", MountEntry{SMB: &SMBConfig{Password: ""}}, false},
		{"no smb config", MountEntry{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.entry.HasPassword(); got != tt.expected {
				t.Errorf("HasPassword() = %v, want %v", got, tt.expected)
			}
		})
	}
}
