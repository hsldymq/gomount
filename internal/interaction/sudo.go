package interaction

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strings"
)

func IsRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		return false
	}
	return currentUser.Uid == "0"
}

func HasSudo() bool {
	_, err := exec.LookPath("sudo")
	return err == nil
}

func NeedsPrivilege() bool {
	return !IsRoot()
}

func CanSudoWithoutPassword() bool {
	cmd := exec.Command("sudo", "-n", "true")
	err := cmd.Run()
	return err == nil
}

func WrapWithSudo(cmd *exec.Cmd) (*exec.Cmd, error) {
	if IsRoot() {
		return cmd, nil
	}

	if !HasSudo() {
		return nil, fmt.Errorf("privilege escalation required but sudo is not available")
	}

	var args []string

	if cmd.Env != nil {
		currentEnv := make(map[string]struct{}, len(os.Environ()))
		for _, e := range os.Environ() {
			k, _, _ := strings.Cut(e, "=")
			currentEnv[k] = struct{}{}
		}
		var extraEnv []string
		for _, e := range cmd.Env {
			k, _, _ := strings.Cut(e, "=")
			if _, ok := currentEnv[k]; !ok {
				extraEnv = append(extraEnv, e)
			}
		}
		if len(extraEnv) > 0 {
			args = append(args, "env")
			args = append(args, extraEnv...)
		}
	}

	args = append(args, cmd.Path)
	args = append(args, cmd.Args[1:]...)

	sudoCmd := exec.Command("sudo", args...)
	sudoCmd.Env = cmd.Env

	return sudoCmd, nil
}

func EnsureSudoCached() error {
	if IsRoot() {
		return nil
	}

	if !HasSudo() {
		return fmt.Errorf("privilege escalation required but sudo is not available")
	}

	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
