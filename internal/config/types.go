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

	SMB    *SMBConfig    `yaml:"smb,omitempty" mapstructure:"smb"`
	SSHFS  *SSHFSConfig  `yaml:"sshfs,omitempty" mapstructure:"sshfs"`
	WebDAV *WebDAVConfig `yaml:"webdav,omitempty" mapstructure:"webdav"`
	S3     *S3Config     `yaml:"s3,omitempty" mapstructure:"s3"`

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

type S3Config struct {
	Provider        string `yaml:"provider" mapstructure:"provider" validate:"required"`
	Bucket          string `yaml:"bucket" mapstructure:"bucket" validate:"required"`
	Path            string `yaml:"path,omitempty" mapstructure:"path"`
	Region          string `yaml:"region,omitempty" mapstructure:"region"`
	Endpoint        string `yaml:"endpoint,omitempty" mapstructure:"endpoint"`
	AccessKeyID     string `yaml:"access_key_id,omitempty" mapstructure:"access_key_id"`
	SecretAccessKey string `yaml:"secret_access_key,omitempty" mapstructure:"secret_access_key"`
	SessionToken    string `yaml:"session_token,omitempty" mapstructure:"session_token"`
	EnvAuth         bool   `yaml:"env_auth,omitempty" mapstructure:"env_auth"`
	ForcePathStyle  *bool  `yaml:"force_path_style,omitempty" mapstructure:"force_path_style"`
}

func ResolveS3Provider(provider string) string {
	switch provider {
	case "aws":
		return "AWS"
	case "aliyun_oss":
		return "Alibaba"
	case "tencent_cos":
		return "TencentCOS"
	case "qiniu_kodo":
		return "Qiniu"
	case "minio":
		return "Minio"
	case "cloudflare_r2":
		return "Cloudflare"
	case "rustfs", "other":
		return "Other"
	default:
		return provider
	}
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
	case "s3":
		if m.S3 == nil {
			return fmt.Errorf("mount entry '%s': type is 's3' but 's3' config is missing", m.Name)
		}
		if m.S3.Provider == "" {
			return fmt.Errorf("mount entry '%s': s3.provider is required", m.Name)
		}
		if m.S3.Bucket == "" {
			return fmt.Errorf("mount entry '%s': s3.bucket is required", m.Name)
		}
		if (m.S3.AccessKeyID == "") != (m.S3.SecretAccessKey == "") {
			return fmt.Errorf("mount entry '%s': s3.access_key_id and s3.secret_access_key must be provided together", m.Name)
		}
		if m.S3.EnvAuth && m.S3.AccessKeyID != "" {
			return fmt.Errorf("mount entry '%s': s3.env_auth cannot be combined with static credentials", m.Name)
		}
	default:
		return fmt.Errorf("mount entry '%s': unknown type '%s'", m.Name, m.Type)
	}
	return nil
}
