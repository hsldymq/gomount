package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/go-playground/validator/v10"
	"go.yaml.in/yaml/v3"
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

func doLoad(path string) (*Config, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to resolve absolute path: %w", err)}
	}

	cfg, err := loadRecursive(absPath, make(map[string]bool))
	if err != nil {
		return nil, err
	}

	if cfg.Sorting != nil && len(cfg.Sorting.By) > 0 {
		cfg.applySorting(cfg.Sorting)
	}

	for i := range cfg.Mounts {
		if err := cfg.Mounts[i].ValidateDriverConfig(); err != nil {
			return nil, &ConfigError{Path: absPath, Err: err}
		}
	}

	if err := cfg.Normalize(); err != nil {
		return nil, &ConfigError{Path: absPath, Err: fmt.Errorf("failed to normalize config: %w", err)}
	}

	return cfg, nil
}

func loadRecursive(path string, visited map[string]bool) (*Config, error) {
	var err error
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to resolve absolute path: %w", err)}
	}

	if visited[path] {
		return &Config{}, nil
	}
	visited[path] = true

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("config file not found")}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to read config: %w", err)}
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, &ConfigError{Path: path, Err: fmt.Errorf("failed to parse config: %w", err)}
	}

	for i := range cfg.Mounts {
		if err := validate.Struct(cfg.Mounts[i]); err != nil {
			return nil, &ConfigError{Path: path, Err: fmt.Errorf("mount entry validation failed: %w", err)}
		}
	}

	for _, includePattern := range cfg.Include {
		files, warn := resolveIncludePaths(includePattern, filepath.Dir(path))
		if warn != "" {
			fmt.Fprintln(os.Stderr, warn)
		}
		for _, file := range files {
			included, err := loadRecursive(file, visited)
			if err != nil {
				return nil, err
			}
			cfg.Mounts = append(cfg.Mounts, included.Mounts...)
		}
	}

	cfg.Include = nil

	if err := cfg.checkNameConflicts(); err != nil {
		return nil, &ConfigError{Path: path, Err: err}
	}

	return cfg, nil
}

func resolveIncludePaths(pattern string, baseDir string) ([]string, string) {
	expanded := pattern
	if len(expanded) > 0 && expanded[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, ""
		}
		expanded = home + expanded[1:]
	}

	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(baseDir, expanded)
	}

	matches, err := filepath.Glob(expanded)
	if err != nil {
		return nil, ""
	}

	if len(matches) == 0 {
		if containsGlobChars(expanded) {
			return nil, ""
		}
		return nil, fmt.Sprintf("WARNING: included file not found: %s", pattern)
	}

	return matches, ""
}

func containsGlobChars(path string) bool {
	for i := 0; i < len(path); i++ {
		switch path[i] {
		case '*', '[', '?':
			return true
		}
	}
	return false
}

func (c *Config) checkNameConflicts() error {
	seen := make(map[string]bool)
	for _, m := range c.Mounts {
		if seen[m.Name] {
			return fmt.Errorf("duplicate mount entry name: '%s'", m.Name)
		}
		seen[m.Name] = true
	}
	return nil
}

func (c *Config) applySorting(s *SortingConfig) {
	if len(s.By) == 0 {
		return
	}

	sort.SliceStable(c.Mounts, func(i, j int) bool {
		for _, field := range s.By {
			desc := false
			f := field
			if len(f) > 0 && f[0] == '-' {
				desc = true
				f = f[1:]
			}
			cmp := compareField(c.Mounts[i], c.Mounts[j], f)
			if cmp == 0 {
				continue
			}
			if desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

func compareField(a, b MountEntry, field string) int {
	switch field {
	case "name":
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// Normalize 应用默认值并解析路径
func (c *Config) Normalize() error {
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

	if len(mountPath) > 0 && mountPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand ~ in mount_dir_path for %s: %w", m.Name, err)
		}
		mountPath = home + mountPath[1:]
	}

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

	if mode.Perm()&0004 != 0 {
		warnings = append(warnings, "Config file is world-readable. Consider chmod 600 for better security.")
	}

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
