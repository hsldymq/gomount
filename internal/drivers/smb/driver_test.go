package smb

import (
	"os"
	"strings"
	"testing"

	"github.com/hsldymq/smb_mount/internal/config"
)

func TestDriver_Type(t *testing.T) {
	d := NewDriver()
	if d.Type() != "smb" {
		t.Errorf("expected type 'smb', got '%s'", d.Type())
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
				Name:      "test",
				SMBAddr:   "192.168.1.100",
				ShareName: "shared",
				Username:  "user",
			},
			wantErr: false,
		},
		{
			name: "missing smb_addr",
			entry: &config.MountEntry{
				Name:      "test",
				ShareName: "shared",
				Username:  "user",
			},
			wantErr: true,
		},
		{
			name: "missing share_name",
			entry: &config.MountEntry{
				Name:     "test",
				SMBAddr:  "192.168.1.100",
				Username: "user",
			},
			wantErr: true,
		},
		{
			name: "missing username",
			entry: &config.MountEntry{
				Name:      "test",
				SMBAddr:   "192.168.1.100",
				ShareName: "shared",
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
	entry := &config.MountEntry{
		Name:            "test",
		SMBAddr:         "192.168.1.100",
		SMBPort:         445,
		ShareName:       "shared",
		Username:        "user",
		Password:        "pass",
		MountDirPath: "/mnt/test",
	}

	cmd := d.buildMountCommand(entry, "/tmp/creds.txt")

	if cmd == nil {
		t.Fatal("expected command, got nil")
	}

	// 验证命令路径以mount.cifs结尾
	if !strings.HasSuffix(cmd.Path, "mount.cifs") {
		t.Errorf("expected path to end with 'mount.cifs', got '%s'", cmd.Path)
	}

	// 验证参数包含必要元素
	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "credentials=/tmp/creds.txt") {
		t.Error("expected credentials in args")
	}
	if !strings.Contains(argsStr, "/mnt/test") {
		t.Error("expected mount path in args")
	}
}

func TestDriver_createCredentialFile(t *testing.T) {
	d := NewDriver()
	entry := &config.MountEntry{
		Username: "testuser",
		Password: "testpass",
	}

	path, err := d.createCredentialFile(entry)
	if err != nil {
		t.Fatalf("createCredentialFile() error = %v", err)
	}
	defer func() {
		// 清理测试文件
		if path != "" {
			_ = os.Remove(path)
		}
	}()

	// 验证文件存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("credential file was not created")
	}

	// 验证文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read credential file: %v", err)
	}

	expected := "username=testuser\npassword=testpass\ndomain=\n"
	if string(content) != expected {
		t.Errorf("credential file content mismatch\nexpected: %s\ngot: %s", expected, string(content))
	}
}
