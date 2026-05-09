package webdav

import (
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestDriver_Type(t *testing.T) {
	d := NewDriver(nil)
	if d.Type() != "webdav" {
		t.Errorf("expected type 'webdav', got '%s'", d.Type())
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

func TestDriver_NeedsSudo(t *testing.T) {
	d := NewDriver(nil)
	if d.NeedsSudo() {
		t.Error("webdav driver via rclone should not need sudo")
	}
}
