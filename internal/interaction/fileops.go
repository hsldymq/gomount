package interaction

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

func GetCurrentUser() (*user.User, error) {
	return user.Current()
}

func GetFileOwner(path string) (uid, gid int, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, fmt.Errorf("failed to get file owner: unsupported platform")
	}
	return int(stat.Uid), int(stat.Gid), nil
}

func IsOwnedByUser(path string, uid int) (bool, error) {
	fileUid, _, err := GetFileOwner(path)
	if err != nil {
		return false, err
	}
	return fileUid == uid, nil
}

func IsOwnedByCurrentUser(path string) (bool, error) {
	u, err := GetCurrentUser()
	if err != nil {
		return false, err
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return false, fmt.Errorf("failed to parse current user uid: %w", err)
	}
	return IsOwnedByUser(path, uid)
}

func MkdirAll(path string, perm os.FileMode) error {
	err := os.MkdirAll(path, perm)
	if err == nil {
		return nil
	}
	if !os.IsPermission(err) {
		return err
	}
	return sudoMkdirAll(path, perm)
}

func sudoMkdirAll(path string, perm os.FileMode) error {
	if IsRoot() {
		return fmt.Errorf("mkdir %s: permission denied even as root", path)
	}
	if !HasSudo() {
		return fmt.Errorf("mkdir %s: permission denied and sudo is not available", path)
	}
	cmd := exec.Command("mkdir", "-p", "-m", fmt.Sprintf("%o", perm), path)
	cmd, err := WrapWithSudo(cmd)
	if err != nil {
		return err
	}
	return RunCommandSilent(cmd)
}

func Chown(path string, uid, gid int) error {
	err := os.Chown(path, uid, gid)
	if err == nil {
		return nil
	}
	if !os.IsPermission(err) {
		return err
	}
	return sudoChown(path, uid, gid)
}

func sudoChown(path string, uid, gid int) error {
	if IsRoot() {
		return fmt.Errorf("chown %s: permission denied even as root", path)
	}
	if !HasSudo() {
		return fmt.Errorf("chown %s: permission denied and sudo is not available", path)
	}
	cmd := exec.Command("chown", "-R", fmt.Sprintf("%d:%d", uid, gid), path)
	cmd, err := WrapWithSudo(cmd)
	if err != nil {
		return err
	}
	return RunCommandSilent(cmd)
}
