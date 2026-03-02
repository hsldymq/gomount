package drivers

import (
	"context"
	"fmt"

	"github.com/hsldymq/smb_mount/internal/config"
)

// Driver 驱动接口，所有挂载驱动必须实现
type Driver interface {
	// Type 返回驱动类型标识
	// 例如: "smb", "sshfs", "tunnel-smb", "webdav"
	Type() string

	// Mount 执行挂载操作
	// ctx: 上下文，用于取消和超时控制
	// entry: 挂载条目配置
	Mount(ctx context.Context, entry *config.MountEntry) error

	// Unmount 执行卸载操作
	// ctx: 上下文
	// entry: 挂载条目配置
	Unmount(ctx context.Context, entry *config.MountEntry) error

	// Status 检查挂载状态
	// 返回挂载状态详情
	Status(ctx context.Context, entry *config.MountEntry) (*MountStatus, error)

	// Validate 验证配置是否有效
	// 在加载配置时调用，提前发现配置错误
	Validate(entry *config.MountEntry) error
}

// MountStatus 挂载状态信息
type MountStatus struct {
	// Mounted 是否已挂载
	Mounted bool

	// Message 状态描述信息
	Message string

	// Details 驱动特定的详细信息
	// 例如：SMB可能包含服务器地址，SSHFS可能包含连接状态
	Details map[string]string
}

// DriverRegistry 驱动注册表
type DriverRegistry struct {
	drivers map[string]Driver
}

// NewRegistry 创建新的驱动注册表
func NewRegistry() *DriverRegistry {
	return &DriverRegistry{
		drivers: make(map[string]Driver),
	}
}

// Register 注册驱动
func (r *DriverRegistry) Register(d Driver) {
	r.drivers[d.Type()] = d
}

// Get 根据类型获取驱动
func (r *DriverRegistry) Get(driverType string) (Driver, bool) {
	d, ok := r.drivers[driverType]
	return d, ok
}

// Detect 自动检测并返回合适的驱动
// 根据MountEntry的字段自动推断驱动类型
func (r *DriverRegistry) Detect(entry *config.MountEntry) (Driver, error) {
	// 1. 优先使用显式声明的type字段
	if entry.Type != "" {
		if d, ok := r.drivers[entry.Type]; ok {
			return d, nil
		}
		return nil, fmt.Errorf("unknown driver type: %s", entry.Type)
	}

	// 2. 向后兼容：根据字段自动推断

	// SSHFS: 有SSH配置且有远程路径
	if entry.SSH != nil && entry.RemotePath != "" {
		if d, ok := r.drivers["sshfs"]; ok {
			return d, nil
		}
	}

	// WebDAV: 有WebDAV配置
	if entry.WebDAV != nil && entry.WebDAV.URL != "" {
		if d, ok := r.drivers["webdav"]; ok {
			return d, nil
		}
	}

	// SMB: 有SMB地址（向后兼容旧配置）
	if entry.SMBAddr != "" {
		if d, ok := r.drivers["smb"]; ok {
			return d, nil
		}
	}

	return nil, fmt.Errorf("cannot determine driver for entry: %s", entry.Name)
}

// List 返回所有已注册的驱动类型
func (r *DriverRegistry) List() []string {
	types := make([]string, 0, len(r.drivers))
	for t := range r.drivers {
		types = append(types, t)
	}
	return types
}

// DriverError 驱动操作错误
type DriverError struct {
	// Driver 驱动类型
	Driver string

	// Op 操作类型: mount, unmount, status
	Op string

	// Entry 挂载条目名称
	Entry string

	// Err 原始错误
	Err error
}

func (e *DriverError) Error() string {
	return fmt.Sprintf("%s driver %s %s: %v", e.Driver, e.Op, e.Entry, e.Err)
}

func (e *DriverError) Unwrap() error {
	return e.Err
}
