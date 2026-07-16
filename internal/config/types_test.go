package config

import (
	"strings"
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

func TestMountEntry_ValidateDriverConfigAcceptsWebDAVURL(t *testing.T) {
	entry := MountEntry{
		Name: "cloud",
		Type: "webdav",
		WebDAV: &WebDAVConfig{
			URL:      "https://cloud.example.com/dav",
			Username: "user",
			Password: "pass",
			Path:     "/docs",
		},
	}

	if err := entry.ValidateDriverConfig(); err != nil {
		t.Fatalf("expected webdav url config to be accepted, got %v", err)
	}
}

func TestMountEntry_ValidateDriverConfigRejectsWebDAVWithoutURL(t *testing.T) {
	entry := MountEntry{
		Name:   "cloud",
		Type:   "webdav",
		WebDAV: &WebDAVConfig{Path: "/docs"},
	}

	err := entry.ValidateDriverConfig()
	if err == nil {
		t.Fatal("expected webdav without url to be rejected")
	}
	if !strings.Contains(err.Error(), "webdav.url is required") {
		t.Fatalf("expected missing url error, got %v", err)
	}
}

func TestMountEntry_ValidateDriverConfigRejectsWebDAVURLUserinfo(t *testing.T) {
	entry := MountEntry{
		Name: "cloud",
		Type: "webdav",
		WebDAV: &WebDAVConfig{
			URL: "https://user:secret@cloud.example.com/dav",
		},
	}

	err := entry.ValidateDriverConfig()
	if err == nil {
		t.Fatal("expected webdav url with userinfo to be rejected")
	}
	if !strings.Contains(err.Error(), "webdav.url must not include credentials") {
		t.Fatalf("expected userinfo error, got %v", err)
	}
}

func TestMountEntryValidateDriverConfigAcceptsOSS(t *testing.T) {
	entry := MountEntry{Name: "archive", Type: "aliyun_oss", AliyunOSS: &AliyunOSSConfig{
		Bucket: "my-bucket", Endpoint: "oss-cn-hangzhou.aliyuncs.com", AccessKeyID: "id", AccessKeySecret: "secret",
	}}
	if err := entry.ValidateDriverConfig(); err != nil {
		t.Fatalf("expected oss config to be accepted, got %v", err)
	}
}

func TestMountEntryValidateDriverConfigRejectsIncompleteOSS(t *testing.T) {
	entry := MountEntry{Name: "archive", Type: "aliyun_oss", AliyunOSS: &AliyunOSSConfig{Bucket: "my-bucket"}}
	err := entry.ValidateDriverConfig()
	if err == nil || !strings.Contains(err.Error(), "aliyun_oss.endpoint is required") {
		t.Fatalf("expected missing endpoint error, got %v", err)
	}
}

func TestMountEntryValidateDriverConfigRejectsLegacyOSSName(t *testing.T) {
	entry := MountEntry{Name: "archive", Type: "oss"}
	err := entry.ValidateDriverConfig()
	if err == nil || !strings.Contains(err.Error(), "unknown type 'oss'") {
		t.Fatalf("expected legacy oss type to be rejected, got %v", err)
	}
}
