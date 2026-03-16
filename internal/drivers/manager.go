package drivers

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/hsldymq/gomount/internal/config"
)

// Manager 驱动管理器
type Manager struct {
	registry *DriverRegistry
	config   *config.Config
	mu       sync.RWMutex
}

// NewManager 创建新的驱动管理器
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		registry: NewRegistry(),
		config:   cfg,
	}
}

// RegisterDriver 注册驱动
func (m *Manager) RegisterDriver(d Driver) {
	m.registry.Register(d)
}

// GetDriver 根据类型获取驱动
func (m *Manager) GetDriver(driverType string) (Driver, error) {
	d, ok := m.registry.Get(driverType)
	if !ok {
		return nil, fmt.Errorf("driver not found: %s", driverType)
	}
	return d, nil
}

// DetectDriver 自动检测条目对应的驱动
func (m *Manager) DetectDriver(entry *config.MountEntry) (Driver, error) {
	return m.registry.Detect(entry)
}

// Mount 挂载指定条目
func (m *Manager) Mount(ctx context.Context, entryName string) error {
	entry, err := m.findEntry(entryName)
	if err != nil {
		return err
	}

	// 确保挂载路径已解析
	entry.GetMountPath()

	// 自动检测驱动
	driver, err := m.DetectDriver(entry)
	if err != nil {
		return fmt.Errorf("cannot detect driver for %s: %w", entryName, err)
	}

	// 验证配置
	if err := driver.Validate(entry); err != nil {
		return fmt.Errorf("validation failed for %s: %w", entryName, err)
	}

	// 执行挂载
	return driver.Mount(ctx, entry)
}

// Unmount 卸载指定条目
func (m *Manager) Unmount(ctx context.Context, entryName string) error {
	entry, err := m.findEntry(entryName)
	if err != nil {
		return err
	}

	// 确保挂载路径已解析
	entry.GetMountPath()

	// 自动检测驱动
	driver, err := m.DetectDriver(entry)
	if err != nil {
		return fmt.Errorf("cannot detect driver for %s: %w", entryName, err)
	}

	return driver.Unmount(ctx, entry)
}

// Status 获取条目状态
func (m *Manager) Status(ctx context.Context, entryName string) (*MountStatus, error) {
	entry, err := m.findEntry(entryName)
	if err != nil {
		return nil, err
	}

	// 确保挂载路径已解析
	entry.GetMountPath()

	// 自动检测驱动
	driver, err := m.DetectDriver(entry)
	if err != nil {
		return nil, fmt.Errorf("cannot detect driver for %s: %w", entryName, err)
	}

	return driver.Status(ctx, entry)
}

// RefreshAllStatus 刷新所有条目的挂载状态
func (m *Manager) RefreshAllStatus(ctx context.Context) error {
	for i := range m.config.Mounts {
		entry := &m.config.Mounts[i]
		entry.GetMountPath()

		driver, err := m.DetectDriver(entry)
		if err != nil {
			// 记录错误但继续处理其他条目
			fmt.Fprintf(os.Stderr, "Warning: cannot detect driver for %s: %v\n", entry.Name, err)
			entry.IsMounted = false
			continue
		}

		status, err := driver.Status(ctx, entry)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to get status for %s: %v\n", entry.Name, err)
			entry.IsMounted = false
			continue
		}

		entry.IsMounted = status.Mounted
	}
	return nil
}

// GetMounts 获取所有挂载条目
func (m *Manager) GetMounts() []config.MountEntry {
	return m.config.Mounts
}

// GetMount 获取指定名称的挂载条目
func (m *Manager) GetMount(name string) (*config.MountEntry, error) {
	return m.findEntry(name)
}

// findEntry 根据名称查找条目
func (m *Manager) findEntry(name string) (*config.MountEntry, error) {
	for i := range m.config.Mounts {
		if m.config.Mounts[i].Name == name {
			return &m.config.Mounts[i], nil
		}
	}
	return nil, fmt.Errorf("mount entry not found: %s", name)
}

// GetRegisteredDrivers 获取所有已注册的驱动类型
func (m *Manager) GetRegisteredDrivers() []string {
	return m.registry.List()
}
