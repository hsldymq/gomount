package config

// Config 主配置结构
type Config struct {
	Mounts     []MountEntry `yaml:"mounts" mapstructure:"mounts" validate:"required,min=1"`
	Workspaces []Workspace  `yaml:"workspaces,omitempty" mapstructure:"workspaces"`
}

// Workspace 挂载组/工作区
type Workspace struct {
	Name        string   `yaml:"name" mapstructure:"name" validate:"required"`
	Description string   `yaml:"description,omitempty" mapstructure:"description"`
	Mounts      []string `yaml:"mounts" mapstructure:"mounts" validate:"required,min=1"`
}

// MountEntry 单个挂载配置（支持多种协议）
type MountEntry struct {
	// 基础字段
	Name         string `yaml:"name" mapstructure:"name" validate:"required"`
	Type         string `yaml:"type,omitempty" mapstructure:"type"`
	MountDirPath string `yaml:"mount_dir_path" mapstructure:"mount_dir_path" validate:"required"`

	// SMB配置（向后兼容）
	SMBAddr   string `yaml:"smb_addr,omitempty" mapstructure:"smb_addr"`
	SMBPort   int    `yaml:"smb_port,omitempty" mapstructure:"smb_port" validate:"min=1,max=65535"`
	ShareName string `yaml:"share_name,omitempty" mapstructure:"share_name"`
	Username  string `yaml:"username,omitempty" mapstructure:"username"`
	Password  string `yaml:"password,omitempty" mapstructure:"password"`

	// SSH配置（用于SSHFS）
	SSH *SSHConfig `yaml:"ssh,omitempty" mapstructure:"ssh"`

	// SSHFS特定
	RemotePath string `yaml:"remote_path,omitempty" mapstructure:"remote_path"`

	// WebDAV配置
	WebDAV *WebDAVConfig `yaml:"webdav,omitempty" mapstructure:"webdav"`

	// 驱动特定选项
	Options map[string]interface{} `yaml:"options,omitempty" mapstructure:"options"`

	// 运行时字段（不从配置加载）
	IsMounted bool `yaml:"-" mapstructure:"-"`
}

// SSHConfig SSH连接配置
type SSHConfig struct {
	Host              string `yaml:"host" mapstructure:"host" validate:"required"`
	Port              int    `yaml:"port,omitempty" mapstructure:"port" validate:"min=1,max=65535"`
	User              string `yaml:"user" mapstructure:"user" validate:"required"`
	Password          string `yaml:"password,omitempty" mapstructure:"password"`
	KeyFile           string `yaml:"key_file,omitempty" mapstructure:"key_file"`
	KeepaliveInterval int    `yaml:"keepalive_interval,omitempty" mapstructure:"keepalive_interval"`
	KeepaliveCountMax int    `yaml:"keepalive_count_max,omitempty" mapstructure:"keepalive_count_max"`
}

// WebDAVConfig WebDAV配置
type WebDAVConfig struct {
	URL      string `yaml:"url" mapstructure:"url" validate:"required,url"`
	Username string `yaml:"username,omitempty" mapstructure:"username"`
	Password string `yaml:"password,omitempty" mapstructure:"password"`
}

// GetMountPath 返回此条目的实际挂载路径
func (m *MountEntry) GetMountPath() string {
	return m.MountDirPath
}

// GetSMBPort 返回 SMB 端口，如果未设置则默认为 445
func (m *MountEntry) GetSMBPort() int {
	if m.SMBPort == 0 {
		return 445
	}
	return m.SMBPort
}

// GetSSHPort 返回 SSH 端口，如果未设置则默认为 22
func (s *SSHConfig) GetPort() int {
	if s.Port == 0 {
		return 22
	}
	return s.Port
}

// GetKeepaliveInterval 返回保活间隔，默认30秒
func (s *SSHConfig) GetKeepaliveInterval() int {
	if s.KeepaliveInterval == 0 {
		return 30
	}
	return s.KeepaliveInterval
}

// GetKeepaliveCountMax 返回保活最大失败次数，默认3
func (s *SSHConfig) GetKeepaliveCountMax() int {
	if s.KeepaliveCountMax == 0 {
		return 3
	}
	return s.KeepaliveCountMax
}

// HasPassword 返回是否配置了密码
func (m *MountEntry) HasPassword() bool {
	return m.Password != ""
}

// GetEffectiveSMBAddr 返回有效的SMB地址
func (m *MountEntry) GetEffectiveSMBAddr() string {
	return m.SMBAddr
}

// GetEffectiveShareName 返回有效的共享名称
func (m *MountEntry) GetEffectiveShareName() string {
	return m.ShareName
}

// GetEffectiveSMBUsername 返回有效的SMB用户名
func (m *MountEntry) GetEffectiveSMBUsername() string {
	return m.Username
}

// GetEffectiveSMBPassword 返回有效的SMB密码
func (m *MountEntry) GetEffectiveSMBPassword() string {
	return m.Password
}

// GetEffectiveSMBPort 返回有效的SMB端口
func (m *MountEntry) GetEffectiveSMBPort() int {
	return m.GetSMBPort()
}
