package interaction

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func RunCommand(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		stderr := strings.TrimSpace(stderrBuf.String())
		if stderr != "" {
			return fmt.Errorf("%s", stderr)
		}
		return err
	}
	return nil
}

func RunCommandSilent(cmd *exec.Cmd) error {
	cmd.Stdin = os.Stdin
	cmd.Stdout = nil

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err != nil {
		stderr := strings.TrimSpace(stderrBuf.String())
		if stderr != "" {
			return fmt.Errorf("%s", stderr)
		}
		return err
	}
	return nil
}
