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
		{
			name: "with absolute path",
			entry: MountEntry{
				Name:         "test",
				MountDirPath: "/mnt/test-mount",
			},
			expectedPath: "/mnt/test-mount",
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

func TestMountEntry_GetSMBPort(t *testing.T) {
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
			entry := MountEntry{SMBPort: tt.port}
			if got := entry.GetSMBPort(); got != tt.expectedPort {
				t.Errorf("GetSMBPort() = %v, want %v", got, tt.expectedPort)
			}
		})
	}
}

func TestSSHConfig_GetPort(t *testing.T) {
	tests := []struct {
		name         string
		port         int
		expectedPort int
	}{
		{"default", 0, 22},
		{"custom", 2222, 2222},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SSHConfig{Port: tt.port}
			if got := cfg.GetPort(); got != tt.expectedPort {
				t.Errorf("GetPort() = %v, want %v", got, tt.expectedPort)
			}
		})
	}
}

func TestSSHConfig_GetKeepaliveInterval(t *testing.T) {
	tests := []struct {
		name             string
		interval         int
		expectedInterval int
	}{
		{"default", 0, 30},
		{"custom", 60, 60},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SSHConfig{KeepaliveInterval: tt.interval}
			if got := cfg.GetKeepaliveInterval(); got != tt.expectedInterval {
				t.Errorf("GetKeepaliveInterval() = %v, want %v", got, tt.expectedInterval)
			}
		})
	}
}

func TestSSHConfig_GetKeepaliveCountMax(t *testing.T) {
	tests := []struct {
		name        string
		countMax    int
		expectedMax int
	}{
		{"default", 0, 3},
		{"custom", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := SSHConfig{KeepaliveCountMax: tt.countMax}
			if got := cfg.GetKeepaliveCountMax(); got != tt.expectedMax {
				t.Errorf("GetKeepaliveCountMax() = %v, want %v", got, tt.expectedMax)
			}
		})
	}
}


func TestMountEntry_HasPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{"has password", "secret", true},
		{"empty password", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := MountEntry{Password: tt.password}
			if got := entry.HasPassword(); got != tt.expected {
				t.Errorf("HasPassword() = %v, want %v", got, tt.expected)
			}
		})
	}
}
