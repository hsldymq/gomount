package mount

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hsldymq/gomount/internal/config"
)

func Mount(entry *config.MountEntry) error {
	mounted, err := CheckEntryStatus(entry)
	if err != nil {
		return fmt.Errorf("failed to check mount status: %w", err)
	}
	if mounted {
		return fmt.Errorf("already mounted at %s", entry.MountDirPath)
	}

	if err := os.MkdirAll(entry.MountDirPath, 0755); err != nil {
		return &MountError{Op: "mount", Path: entry.MountDirPath, Err: fmt.Errorf("failed to create mount directory: %w", err)}
	}

	credsFile, err := createCredentialFile(entry)
	if err != nil {
		return &MountError{Op: "mount", Path: entry.MountDirPath, Err: err}
	}
	defer os.Remove(credsFile)

	cmd := buildMountCommand(entry, credsFile)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &MountError{Op: "mount", Path: entry.MountDirPath, Err: fmt.Errorf("%s", string(output))}
	}

	return nil
}

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

func EnsureBaseDir(baseDir string) error {
	info, err := os.Stat(baseDir)
	if os.IsNotExist(err) {
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return fmt.Errorf("failed to create base directory: %w", err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to access base directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("base_dir exists but is not a directory: %s", baseDir)
	}

	return nil
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
