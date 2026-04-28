package webdav

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	"github.com/hsldymq/gomount/internal/status"
)

// Driver WebDAV驱动实现
type Driver struct{}

// NewDriver 创建新的WebDAV驱动
func NewDriver() *Driver {
	return &Driver{}
}

// Type 返回驱动类型
func (d *Driver) Type() string {
	return "webdav"
}

// Mount 执行WebDAV挂载
func (d *Driver) Mount(ctx context.Context, entry *config.MountEntry) error {
	// 检查是否已挂载
	mounted, err := status.CheckStatus(entry.MountDirPath)
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
			Err:    fmt.Errorf("mount.davfs failed: %w\nOutput: %s", err, string(output)),
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
	mounted, err := status.CheckStatus(entry.MountDirPath)
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
			"type": "webdav",
			"url":  entry.WebDAV.URL,
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
	if entry.WebDAV == nil {
		return fmt.Errorf("webdav config is required for webdav driver")
	}
	if entry.WebDAV.URL == "" {
		return fmt.Errorf("webdav.url is required for webdav driver")
	}
	return nil
}

// buildMountCommand 构建mount.davfs命令
func (d *Driver) buildMountCommand(entry *config.MountEntry) *exec.Cmd {
	var options []string

	// 文件权限选项
	options = append(options,
		fmt.Sprintf("uid=%d", os.Getuid()),
		fmt.Sprintf("gid=%d", os.Getgid()),
	)

	// 从entry.Options获取其他选项
	if entry.Options != nil {
		if fileMode, ok := entry.Options["file_mode"]; ok {
			options = append(options, fmt.Sprintf("file_mode=%v", fileMode))
		}
		if dirMode, ok := entry.Options["dir_mode"]; ok {
			options = append(options, fmt.Sprintf("dir_mode=%v", dirMode))
		}
	}

	// 构建命令
	args := []string{
		entry.WebDAV.URL,
		entry.MountDirPath,
	}

	if len(options) > 0 {
		args = append(args, "-o", strings.Join(options, ","))
	}

	cmd := exec.Command("mount.davfs", args...)

	// 设置环境变量（用户名密码）
	env := os.Environ()
	if entry.WebDAV.Username != "" {
		env = append(env, fmt.Sprintf("WEBDAV_USERNAME=%s", entry.WebDAV.Username))
	}
	if entry.WebDAV.Password != "" {
		env = append(env, fmt.Sprintf("WEBDAV_PASSWORD=%s", entry.WebDAV.Password))
	}
	cmd.Env = env

	return cmd
}
