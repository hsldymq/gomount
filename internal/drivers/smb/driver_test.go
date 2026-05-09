package smb

import (
	"os"
	"strings"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

func TestDriver_Type(t *testing.T) {
	d := NewDriver(nil)
	if d.Type() != "smb" {
		t.Errorf("expected type 'smb', got '%s'", d.Type())
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
				Name: "test",
				SMB:  &config.SMBConfig{Addr: "192.168.1.100", ShareName: "shared", Username: "user"},
			},
			wantErr: false,
		},
		{
			name: "missing smb config",
			entry: &config.MountEntry{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "missing smb addr",
			entry: &config.MountEntry{
				Name: "test",
				SMB:  &config.SMBConfig{ShareName: "shared", Username: "user"},
			},
			wantErr: true,
		},
		{
			name: "missing share_name",
			entry: &config.MountEntry{
				Name: "test",
				SMB:  &config.SMBConfig{Addr: "192.168.1.100", Username: "user"},
			},
			wantErr: true,
		},
		{
			name: "missing username",
			entry: &config.MountEntry{
				Name: "test",
				SMB:  &config.SMBConfig{Addr: "192.168.1.100", ShareName: "shared"},
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
	entry := &config.MountEntry{
		Name:         "test",
		SMB:          &config.SMBConfig{Addr: "192.168.1.100", Port: 445, ShareName: "shared", Username: "user", Password: "pass"},
		MountDirPath: "/mnt/test",
	}

	cmd := d.buildMountCommand(entry, "/tmp/creds.txt", entry.SMB.Addr, entry.SMB.GetPort())

	if cmd == nil {
		t.Fatal("expected command, got nil")
	}

	if !strings.HasSuffix(cmd.Path, "mount.cifs") {
		t.Errorf("expected path to end with 'mount.cifs', got '%s'", cmd.Path)
	}

	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "credentials=/tmp/creds.txt") {
		t.Error("expected credentials in args")
	}
	if !strings.Contains(argsStr, "/mnt/test") {
		t.Error("expected mount path in args")
	}
}

func TestDriver_createCredentialFile(t *testing.T) {
	d := NewDriver(nil)
	entry := &config.MountEntry{
		SMB: &config.SMBConfig{Username: "testuser", Password: "testpass"},
	}

	path, err := d.createCredentialFile(entry)
	if err != nil {
		t.Fatalf("createCredentialFile() error = %v", err)
	}
	defer func() {
		if path != "" {
			_ = os.Remove(path)
		}
	}()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("credential file was not created")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read credential file: %v", err)
	}

	expected := "username=testuser\npassword=testpass\ndomain=\n"
	if string(content) != expected {
		t.Errorf("credential file content mismatch\nexpected: %s\ngot: %s", expected, string(content))
	}
}
