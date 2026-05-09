package mountkit

import (
	"fmt"

	"github.com/rclone/rclone/fs/config"
)

func initRcloneConfig() {
	config.SetConfigPath("")
}

func RegisterSMBRemote(name, host string, port int, user, pass, shareName string) string {
	config.FileSetValue(name, "type", "smb")
	config.FileSetValue(name, "host", host)
	config.FileSetValue(name, "user", user)
	config.FileSetValue(name, "pass", pass)
	if port != 445 {
		config.FileSetValue(name, "port", fmt.Sprintf("%d", port))
	}
	return name + ":" + shareName
}

func RegisterWebDAVRemote(name, url, user, pass string) string {
	config.FileSetValue(name, "type", "webdav")
	config.FileSetValue(name, "url", url)
	if user != "" {
		config.FileSetValue(name, "user", user)
	}
	if pass != "" {
		config.FileSetValue(name, "pass", pass)
	}
	return name + ":/"
}

func RegisterSFTPRemote(name, host, remotePath, sshBinary string) string {
	config.FileSetValue(name, "type", "sftp")
	config.FileSetValue(name, "host", host)
	config.FileSetValue(name, "shell_type", "unix")
	if sshBinary != "" {
		config.FileSetValue(name, "ssh", sshBinary)
	}
	return name + ":" + remotePath
}
