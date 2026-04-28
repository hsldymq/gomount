package mount

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hsldymq/gomount/internal/config"
)

func CreateCredentialFile(entry *config.MountEntry) (string, error) {
	return createCredentialFile(entry)
}

func createCredentialFile(entry *config.MountEntry) (string, error) {
	tmpFile, err := os.CreateTemp("", "gomount_creds_*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create credentials file: %w", err)
	}
	defer tmpFile.Close()

	if err := tmpFile.Chmod(0600); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to set credentials file permissions: %w", err)
	}

	content := fmt.Sprintf("username=%s\npassword=%s\ndomain=%s\n",
		entry.SMB.Username,
		entry.SMB.Password,
		"",
	)

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write credentials: %w", err)
	}

	return tmpFile.Name(), nil
}

func BuildMountCommand(entry *config.MountEntry, credsFile string) *exec.Cmd {
	return buildMountCommand(entry, credsFile)
}

func buildMountCommand(entry *config.MountEntry, credsFile string) *exec.Cmd {
	smbAddr := fmt.Sprintf("//%s/%s", entry.SMB.Addr, entry.SMB.ShareName)

	options := fmt.Sprintf("credentials=%s,port=%d,file_mode=0755,dir_mode=0755,uid=%d,gid=%d",
		credsFile,
		entry.SMB.GetPort(),
		os.Getuid(),
		os.Getgid(),
	)

	args := []string{
		smbAddr,
		entry.MountDirPath,
		"-o", options,
	}

	return exec.Command("mount.cifs", args...)
}

type MountError struct {
	Op   string
	Path string
	Err  error
}

func (e *MountError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.Op, e.Path, e.Err)
}

func (e *MountError) Unwrap() error {
	return e.Err
}
