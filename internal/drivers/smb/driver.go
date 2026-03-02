package smb

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hsldymq/smb_mount/internal/config"
	"github.com/hsldymq/smb_mount/internal/drivers"
	"github.com/hsldymq/smb_mount/internal/mount"
)

// Driver SMB驱动实现
type Driver struct{}

// NewDriver 创建新的SMB驱动
func NewDriver() *Driver {
	return &Driver{}
}

// Type 返回驱动类型
func (d *Driver) Type() string {
	return "smb"
}

// Mount 执行SMB挂载
func (d *Driver) Mount(ctx context.Context, entry *config.MountEntry) error {
	// 检查是否已挂载
	mounted, err := mount.CheckStatus(entry.MountDirPath)
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("failed to check mount status: %w", err),
		}
	}
	if mounted {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("already mounted at %s", entry.MountDirPath),
		}
	}

	// 创建挂载目录
	if err := os.MkdirAll(entry.MountDirPath, 0755); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("failed to create mount directory: %w", err),
		}
	}

	// 创建凭据文件
	credsFile, err := d.createCredentialFile(entry)
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    err,
		}
	}
	defer os.Remove(credsFile)

	// 构建并执行挂载命令
	cmd := d.buildMountCommand(entry, credsFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("mount.cifs failed: %w\nOutput: %s", err, string(output)),
		}
	}

	return nil
}

// Unmount 执行卸载
func (d *Driver) Unmount(ctx context.Context, entry *config.MountEntry) error {
	cmd := exec.CommandContext(ctx, "umount", entry.MountDirPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "unmount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("umount failed: %w\nOutput: %s", err, string(output)),
		}
	}
	return nil
}

// Status 检查挂载状态
func (d *Driver) Status(ctx context.Context, entry *config.MountEntry) (*drivers.MountStatus, error) {
	mounted, err := mount.CheckStatus(entry.MountDirPath)
	if err != nil {
		return nil, &drivers.DriverError{
			Driver: d.Type(),
			Op:     "status",
			Entry:  entry.Name,
			Err:    fmt.Errorf("failed to check mount status: %w", err),
		}
	}

	status := &drivers.MountStatus{
		Mounted: mounted,
		Details: map[string]string{
			"type": "smb",
			"addr": entry.SMBAddr,
		},
	}

	if mounted {
		status.Message = fmt.Sprintf("Mounted at %s", entry.MountDirPath)
		status.Details["share"] = entry.ShareName
	} else {
		status.Message = "Not mounted"
	}

	return status, nil
}

// Validate 验证配置
func (d *Driver) Validate(entry *config.MountEntry) error {
	if entry.SMBAddr == "" {
		return fmt.Errorf("smb_addr is required for SMB driver")
	}
	if entry.ShareName == "" {
		return fmt.Errorf("share_name is required for SMB driver")
	}
	if entry.Username == "" {
		return fmt.Errorf("username is required for SMB driver")
	}
	return nil
}

// createCredentialFile 创建临时凭据文件
func (d *Driver) createCredentialFile(entry *config.MountEntry) (string, error) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "smb_mount_creds_*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create credentials file: %w", err)
	}
	defer tmpFile.Close()

	// 设置权限为仅所有者可读写
	if err := tmpFile.Chmod(0600); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to set credentials file permissions: %w", err)
	}

	// 写入凭据
	content := fmt.Sprintf("username=%s\npassword=%s\ndomain=%s\n",
		entry.Username,
		entry.Password,
		"", // domain支持可后续添加
	)

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write credentials: %w", err)
	}

	return tmpFile.Name(), nil
}

// buildMountCommand 构建mount.cifs命令
func (d *Driver) buildMountCommand(entry *config.MountEntry, credsFile string) *exec.Cmd {
	// 构建SMB地址
	smbAddr := fmt.Sprintf("//%s:%d/%s", entry.SMBAddr, entry.GetSMBPort(), entry.ShareName)

	// 构建挂载选项
	options := fmt.Sprintf("credentials=%s,file_mode=0755,dir_mode=0755,uid=%d,gid=%d",
		credsFile,
		os.Getuid(),
		os.Getgid(),
	)

	// 构建命令
	args := []string{
		smbAddr,
		entry.MountDirPath,
		"-o", options,
	}

	return exec.Command("mount.cifs", args...)
}
