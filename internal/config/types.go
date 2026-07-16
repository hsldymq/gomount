package config

import (
	"fmt"
	"net/url"

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
	Daemon  *DaemonConfig  `yaml:"daemon,omitempty" mapstructure:"daemon"`
}

// MountEntry 单个挂载配置（支持多种协议）
type MountEntry struct {
	Name         string `yaml:"name" mapstructure:"name" validate:"required"`
	Type         string `yaml:"type" mapstructure:"type" validate:"required"`
	MountDirPath string `yaml:"mount_dir_path" mapstructure:"mount_dir_path" validate:"required"`

	SMB       *SMBConfig       `yaml:"smb,omitempty" mapstructure:"smb"`
	SSHFS     *SSHFSConfig     `yaml:"sshfs,omitempty" mapstructure:"sshfs"`
	WebDAV    *WebDAVConfig    `yaml:"webdav,omitempty" mapstructure:"webdav"`
	AliyunOSS *AliyunOSSConfig `yaml:"aliyun_oss,omitempty" mapstructure:"aliyun_oss"`

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

type WebDAVConfig struct {
	URL      string `yaml:"url" mapstructure:"url" validate:"required,url"`
	Username string `yaml:"username,omitempty" mapstructure:"username"`
	Password string `yaml:"password,omitempty" mapstructure:"password"`
	Path     string `yaml:"path,omitempty" mapstructure:"path"`
}

// AliyunOSSConfig configures an Alibaba Cloud OSS bucket through rclone's S3 backend.
type AliyunOSSConfig struct {
	Bucket          string `yaml:"bucket" mapstructure:"bucket" validate:"required"`
	Path            string `yaml:"path,omitempty" mapstructure:"path"`
	Endpoint        string `yaml:"endpoint" mapstructure:"endpoint" validate:"required"`
	AccessKeyID     string `yaml:"access_key_id" mapstructure:"access_key_id" validate:"required"`
	AccessKeySecret string `yaml:"access_key_secret" mapstructure:"access_key_secret" validate:"required"`
	SecurityToken   string `yaml:"security_token,omitempty" mapstructure:"security_token"`
}

type SSHTunnelConfig struct {
	Host string `yaml:"host" mapstructure:"host" validate:"required"`
}

type SortingConfig struct {
	By StringOrSlice `yaml:"by" mapstructure:"by"`
}

type DaemonConfig struct {
	LogTarget  string `yaml:"log_target,omitempty" mapstructure:"log_target"`
	LogFile    string `yaml:"log_file,omitempty" mapstructure:"log_file"`
	SocketPath string `yaml:"socket_path,omitempty" mapstructure:"socket_path"`
}

func (d *DaemonConfig) GetLogTarget() string {
	if d == nil || d.LogTarget == "" {
		return "syslog"
	}
	return d.LogTarget
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
		if m.WebDAV.URL == "" {
			return fmt.Errorf("mount entry '%s': webdav.url is required", m.Name)
		}
		parsedURL, err := url.Parse(m.WebDAV.URL)
		if err != nil {
			return fmt.Errorf("mount entry '%s': invalid webdav.url: %w", m.Name, err)
		}
		if parsedURL.User != nil {
			return fmt.Errorf("mount entry '%s': webdav.url must not include credentials; use webdav.username and webdav.password", m.Name)
		}
	case "aliyun_oss":
		if m.AliyunOSS == nil {
			return fmt.Errorf("mount entry '%s': type is 'aliyun_oss' but 'aliyun_oss' config is missing", m.Name)
		}
		if m.AliyunOSS.Bucket == "" {
			return fmt.Errorf("mount entry '%s': aliyun_oss.bucket is required", m.Name)
		}
		if m.AliyunOSS.Endpoint == "" {
			return fmt.Errorf("mount entry '%s': aliyun_oss.endpoint is required", m.Name)
		}
		if m.AliyunOSS.AccessKeyID == "" {
			return fmt.Errorf("mount entry '%s': aliyun_oss.access_key_id is required", m.Name)
		}
		if m.AliyunOSS.AccessKeySecret == "" {
			return fmt.Errorf("mount entry '%s': aliyun_oss.access_key_secret is required", m.Name)
		}
	default:
		return fmt.Errorf("mount entry '%s': unknown type '%s'", m.Name, m.Type)
	}
	return nil
}
