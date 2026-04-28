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
	Name         string `yaml:"name" mapstructure:"name" validate:"required"`
	Type         string `yaml:"type,omitempty" mapstructure:"type"`
	MountDirPath string `yaml:"mount_dir_path" mapstructure:"mount_dir_path" validate:"required"`

	SMB    *SMBConfig    `yaml:"smb,omitempty" mapstructure:"smb"`
	SSHFS  *SSHFSConfig  `yaml:"sshfs,omitempty" mapstructure:"sshfs"`
	WebDAV *WebDAVConfig `yaml:"webdav,omitempty" mapstructure:"webdav"`

	Options   map[string]interface{} `yaml:"options,omitempty" mapstructure:"options"`
	IsMounted bool                   `yaml:"-" mapstructure:"-"`
}

// SMBConfig SMB/CIFS配置
type SMBConfig struct {
	Addr      string `yaml:"addr" mapstructure:"addr" validate:"required"`
	Port      int    `yaml:"port,omitempty" mapstructure:"port" validate:"min=1,max=65535"`
	ShareName string `yaml:"share_name" mapstructure:"share_name" validate:"required"`
	Username  string `yaml:"username" mapstructure:"username" validate:"required"`
	Password  string `yaml:"password,omitempty" mapstructure:"password"`
}

// SSHFSConfig SSHFS配置
type SSHFSConfig struct {
	Host       string `yaml:"host" mapstructure:"host" validate:"required"`
	RemotePath string `yaml:"remote_path" mapstructure:"remote_path" validate:"required"`
}

// WebDAVConfig WebDAV配置
type WebDAVConfig struct {
	URL      string `yaml:"url" mapstructure:"url" validate:"required,url"`
	Username string `yaml:"username,omitempty" mapstructure:"username"`
	Password string `yaml:"password,omitempty" mapstructure:"password"`
}

func (m *MountEntry) GetMountPath() string {
	return m.MountDirPath
}

func (s *SMBConfig) GetPort() int {
	if s.Port == 0 {
		return 445
	}
	return s.Port
}

func (m *MountEntry) HasPassword() bool {
	if m.SMB != nil {
		return m.SMB.Password != ""
	}
	return false
}
