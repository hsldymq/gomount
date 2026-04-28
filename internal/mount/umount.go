package mount

import (
	"os/exec"
)

// BuildUmountCommand 为外部使用构建 umount 命令
func BuildUmountCommand(mountPath string) *exec.Cmd {
	return exec.Command("umount", mountPath)
}
