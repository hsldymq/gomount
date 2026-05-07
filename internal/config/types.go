package config

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

type StringOrSlice []string

func (s *StringOrSlice) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		*s = []string{value.Value}
		return nil
	}
	if value.Kind == yaml.SequenceNode {
		var items []string
		if err := value.Decode(&items); err != nil {
			return err
		}
		*s = items
		return nil
	}
	return fmt.Errorf("sorting.by must be a string or list of strings")
}

type Config struct {
	Mounts  []MountEntry   `yaml:"mounts" mapstructure:"mounts"`
	Include []string       `yaml:"include,omitempty" mapstructure:"include"`
	Sorting *SortingConfig `yaml:"sorting,omitempty" mapstructure:"sorting"`
}

// MountEntry 单个挂载配置（支持多种协议）
type MountEntry struct {
	Name         string `yaml:"name" mapstructure:"name" validate:"required"`
	Type         string `yaml:"type" mapstructure:"type" validate:"required"`
	MountDirPath string `yaml:"mount_dir_path" mapstructure:"mount_dir_path" validate:"required"`

	SMB    *SMBConfig    `yaml:"smb,omitempty" mapstructure:"smb"`
	SSHFS  *SSHFSConfig  `yaml:"sshfs,omitempty" mapstructure:"sshfs"`
	WebDAV *WebDAVConfig `yaml:"webdav,omitempty" mapstructure:"webdav"`

	SSHTunnel *SSHTunnelConfig `yaml:"ssh_tunnel,omitempty" mapstructure:"ssh_tunnel"`

	Options   map[string]interface{} `yaml:"options,omitempty" mapstructure:"options"`
	IsMounted bool                   `yaml:"-" mapstructure:"-"`
}

// SMBConfig SMB/CIFS配置
type SMBConfig struct {
	Addr      string `yaml:"addr" mapstructure:"addr" validate:"required"`
	Port      int    `yaml:"port,omitempty" mapstructure:"port" validate:"omitempty,min=1,max=65535"`
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

type SSHTunnelConfig struct {
	Host string `yaml:"host" mapstructure:"host" validate:"required"`
}

type SortingConfig struct {
	By StringOrSlice `yaml:"by" mapstructure:"by"`
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

// ValidateDriverConfig 根据 type 字段检查对应的配置块是否存在
func (m *MountEntry) ValidateDriverConfig() error {
	switch m.Type {
	case "smb":
		if m.SMB == nil {
			return fmt.Errorf("mount entry '%s': type is 'smb' but 'smb' config is missing", m.Name)
		}
	case "sshfs":
		if m.SSHFS == nil {
			return fmt.Errorf("mount entry '%s': type is 'sshfs' but 'sshfs' config is missing", m.Name)
		}
	case "webdav":
		if m.WebDAV == nil {
			return fmt.Errorf("mount entry '%s': type is 'webdav' but 'webdav' config is missing", m.Name)
		}
	default:
		return fmt.Errorf("mount entry '%s': unknown type '%s'", m.Name, m.Type)
	}
	return nil
}
