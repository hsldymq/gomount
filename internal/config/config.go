package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

// DefaultConfigPath 返回默认配置文件路径
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gomount.yaml")
}

// LoadConfig 从指定路径或默认路径载入配置
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		path = DefaultConfigPath()
	}
	return doLoad(path)
}

// doLoad 从指定路径加载配置
func doLoad(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("config file not found")}
	}

	// Create new viper instance
	v := viper.New()

	// Set config path and file
	v.SetConfigFile(path)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to read config: %w", err)}
	}

	// Unmarshal config
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to parse config: %w", err)}
	}

	// Validate config
	if err := validate.Struct(cfg); err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("config validation failed: %w", err)}
	}

	// Validate driver config for each mount entry
	for i := range cfg.Mounts {
		if err := cfg.Mounts[i].ValidateDriverConfig(); err != nil {
			return nil, &ConfigError{Path: path, Err: err}
		}
	}

	// Normalize mount paths
	if err := cfg.Normalize(); err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to normalize config: %w", err)}
	}

	return cfg, nil
}

// Normalize 应用默认值并解析路径
func (c *Config) Normalize() error {
	// Normalize each mount entry
	for i := range c.Mounts {
		if err := c.Mounts[i].Normalize(); err != nil {
			return err
		}
	}

	return nil
}

// Normalize 解析单个条目的挂载路径
func (m *MountEntry) Normalize() error {
	mountPath := m.MountDirPath

	// Expand ~ if present
	if len(mountPath) > 0 && mountPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand ~ in mount_dir_path for %s: %w", m.Name, err)
		}
		mountPath = home + mountPath[1:]
	}

	// Convert to absolute path
	var err error
	mountPath, err = filepath.Abs(mountPath)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path for mount_dir_path of %s: %w", m.Name, err)
	}

	m.MountDirPath = mountPath
	return nil
}

// EnsureMountDir 如果挂载目录的父目录不存在则创建
func (m *MountEntry) EnsureMountDir() error {
	parentDir := filepath.Dir(m.MountDirPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", m.Name, err)
	}
	return nil
}

// FindByName 按名称查找挂载条目
func (c *Config) FindByName(name string) (*MountEntry, bool) {
	for i := range c.Mounts {
		if c.Mounts[i].Name == name {
			return &c.Mounts[i], true
		}
	}
	return nil, false
}

// GetMountEntryNames 返回所有挂载条目名称
func (c *Config) GetMountEntryNames() []string {
	names := make([]string, len(c.Mounts))
	for i, m := range c.Mounts {
		names[i] = m.Name
	}
	return names
}

// SetMountStatus 更新条目的挂载状态
func (c *Config) SetMountStatus(name string, isMounted bool) {
	for i := range c.Mounts {
		if c.Mounts[i].Name == name {
			c.Mounts[i].IsMounted = isMounted
			break
		}
	}
}

// CheckConfigPermissions 检查配置文件是否具有安全权限
func CheckConfigPermissions(path string) ([]string, []error) {
	var warnings []string
	var errors []error

	info, err := os.Stat(path)
	if err != nil {
		errors = append(errors, fmt.Errorf("failed to stat config file: %w", err))
		return warnings, errors
	}

	mode := info.Mode()

	// Check if world-readable
	if mode.Perm()&0004 != 0 {
		warnings = append(warnings, "Config file is world-readable. Consider chmod 600 for better security.")
	}

	// Check if group-readable (warn if not owner-only)
	if mode.Perm()&0040 != 0 {
		warnings = append(warnings, "Config file is group-readable. For best security, only the owner should read it.")
	}

	return warnings, errors
}

// ConfigError 配置加载或验证错误
type ConfigError struct {
	Path string
	Err  error
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error (%s): %s", e.Path, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}
