package sshfs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	"github.com/hsldymq/gomount/internal/mount"
)

// Driver SSHFS驱动实现
type Driver struct{}

// NewDriver 创建新的SSHFS驱动
func NewDriver() *Driver {
	return &Driver{}
}

// Type 返回驱动类型
func (d *Driver) Type() string {
	return "sshfs"
}

// Mount 执行SSHFS挂载
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

	// 构建并执行挂载命令
	cmd := d.buildMountCommand(entry)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("sshfs failed: %w\nOutput: %s", err, string(output)),
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
			"type":       "sshfs",
			"host":       entry.SSH.Host,
			"remotePath": entry.RemotePath,
		},
	}

	if mounted {
		status.Message = fmt.Sprintf("Mounted at %s", entry.MountDirPath)
	} else {
		status.Message = "Not mounted"
	}

	return status, nil
}

// Validate 验证配置
func (d *Driver) Validate(entry *config.MountEntry) error {
	if entry.SSH == nil {
		return fmt.Errorf("ssh config is required for SSHFS driver")
	}
	if entry.SSH.Host == "" {
		return fmt.Errorf("ssh.host is required for SSHFS driver")
	}
	if entry.SSH.User == "" {
		return fmt.Errorf("ssh.user is required for SSHFS driver")
	}
	if entry.RemotePath == "" {
		return fmt.Errorf("remote_path is required for SSHFS driver")
	}
	return nil
}

// buildMountCommand 构建sshfs命令
func (d *Driver) buildMountCommand(entry *config.MountEntry) *exec.Cmd {
	// 构建远程路径
	remotePath := fmt.Sprintf("%s@%s:%s",
		entry.SSH.User,
		entry.SSH.Host,
		entry.RemotePath,
	)

	// 构建选项
	var options []string

	// 端口配置
	if entry.SSH.GetPort() != 22 {
		options = append(options, fmt.Sprintf("port=%d", entry.SSH.GetPort()))
	}

	// 密钥认证
	if entry.SSH.KeyFile != "" {
		options = append(options, fmt.Sprintf("IdentityFile=%s", entry.SSH.KeyFile))
	}

	// 默认选项
	options = append(options,
		"follow_symlinks",
		"cache_timeout=600",
		" reconnect",
	)

	// 构建命令
	args := []string{
		"-o", strings.Join(options, ","),
		remotePath,
		entry.MountDirPath,
	}

	return exec.Command("sshfs", args...)
}
