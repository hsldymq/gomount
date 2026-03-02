package interaction

import (
	"bufio"
	"bytes"
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

func PromptSudoPassword() (string, error) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "╔════════════════════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stderr, "║  需要管理员权限才能挂载文件系统                              ║")
	fmt.Fprintln(os.Stderr, "╠════════════════════════════════════════════════════════════╣")
	fmt.Fprintln(os.Stderr, "║  请输入您的 sudo 密码                                        ║")
	fmt.Fprintln(os.Stderr, "║  （输入时不会显示任何字符，输入完成后按回车）                  ║")
	fmt.Fprintln(os.Stderr, "╚════════════════════════════════════════════════════════════╝")
	fmt.Fprint(os.Stderr, "密码: ")
	os.Stderr.Sync()

	reader := bufio.NewReader(os.Stdin)
	password, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	password = strings.TrimSpace(password)
	fmt.Fprintln(os.Stderr)

	return password, nil
}

func RunWithSudo(cmd *exec.Cmd) error {
	if IsRoot() {
		return cmd.Run()
	}

	if !HasSudo() {
		return fmt.Errorf("privilege escalation required but sudo is not available")
	}

	if CanSudoWithoutPassword() {
		return runSudoCommand(cmd, "")
	}

	password, err := PromptSudoPassword()
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}

	return runSudoCommand(cmd, password)
}

func runSudoCommand(cmd *exec.Cmd, password string) error {
	sudoArgs := []string{"-S", "-p", ""}
	sudoArgs = append(sudoArgs, cmd.Path)
	sudoArgs = append(sudoArgs, cmd.Args[1:]...)

	sudoCmd := exec.Command("sudo", sudoArgs...)

	// 捕获 stderr 以显示详细错误
	var stderrBuf bytes.Buffer
	sudoCmd.Stderr = &stderrBuf

	if password != "" {
		stdin, err := sudoCmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdin pipe: %w", err)
		}

		go func() {
			defer stdin.Close()
			fmt.Fprintln(stdin, password)
		}()
	} else {
		sudoCmd.Stdin = os.Stdin
	}

	sudoCmd.Stdout = cmd.Stdout

	if err := sudoCmd.Run(); err != nil {
		stderr := stderrBuf.String()
		if stderr != "" {
			return fmt.Errorf("%s", strings.TrimSpace(stderr))
		}
		return fmt.Errorf("sudo command failed: %w", err)
	}

	return nil
}

func RunWithSudoSilent(cmd *exec.Cmd) error {
	if IsRoot() {
		return cmd.Run()
	}

	if !HasSudo() {
		return fmt.Errorf("privilege escalation required but sudo is not available")
	}

	sudoArgs := []string{"-S", "-p", ""}
	sudoArgs = append(sudoArgs, cmd.Path)
	sudoArgs = append(sudoArgs, cmd.Args[1:]...)

	sudoCmd := exec.Command("sudo", sudoArgs...)
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = cmd.Stdout
	sudoCmd.Stderr = cmd.Stderr

	return sudoCmd.Run()
}
