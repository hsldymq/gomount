package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

const (
	DefaultPort    = 13579
	DaemonEnvKey   = "GOMOUNT_DAEMON"
	DaemonEnvValue = "1"
)

type DaemonConfig struct {
	Port int `yaml:"port,omitempty" mapstructure:"port"`
}

func (c DaemonConfig) GetPort() int {
	if c.Port <= 0 {
		return DefaultPort
	}
	return c.Port
}

func IsDaemon() bool {
	return os.Getenv(DaemonEnvKey) == DaemonEnvValue
}

func dataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "gomount")
	return dir, os.MkdirAll(dir, 0755)
}

func pidFilePath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.pid"), nil
}

func portFilePath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "daemon.port"), nil
}

type DaemonInfo struct {
	PID  int
	Port int
}

func ReadDaemonInfo() (*DaemonInfo, error) {
	pidPath, err := pidFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return nil, fmt.Errorf("daemon not running: %w", err)
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return nil, fmt.Errorf("invalid pid file: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("daemon process not found: %w", err)
	}

	if err := proc.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidPath)
		return nil, fmt.Errorf("daemon process %d not alive", pid)
	}

	portPath, err := portFilePath()
	if err != nil {
		return nil, err
	}

	portData, err := os.ReadFile(portPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read port file: %w", err)
	}

	var port int
	if _, err := fmt.Sscanf(string(portData), "%d", &port); err != nil {
		return nil, fmt.Errorf("invalid port file: %w", err)
	}

	return &DaemonInfo{PID: pid, Port: port}, nil
}

func WriteDaemonInfo(pid int, port int) error {
	pidPath, err := pidFilePath()
	if err != nil {
		return err
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return err
	}

	portPath, err := portFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(portPath, []byte(strconv.Itoa(port)), 0644)
}

func CleanupDaemonInfo() {
	pidPath, _ := pidFilePath()
	os.Remove(pidPath)
	portPath, _ := portFilePath()
	os.Remove(portPath)
}

func StartDaemon(configPath string, cfg DaemonConfig) error {
	me, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable: %w", err)
	}

	port := cfg.GetPort()

	args := []string{me}
	if configPath != "" {
		args = append(args, "-c", configPath)
	}

	env := append(os.Environ(),
		DaemonEnvKey+"="+DaemonEnvValue,
		"GOMOUNT_DAEMON_PORT="+strconv.Itoa(port),
	)

	// Add original user info
	if currentUser, err := user.Current(); err == nil {
		env = append(env,
			"GOMOUNT_ORIGINAL_UID="+currentUser.Uid,
			"GOMOUNT_ORIGINAL_GID="+currentUser.Gid,
			"GOMOUNT_ORIGINAL_USERNAME="+currentUser.Username,
		)
	}

	null, err := os.Open(os.DevNull)
	if err != nil {
		return err
	}
	defer null.Close()

	attr := &os.ProcAttr{
		Env:   env,
		Files: []*os.File{null, null, null},
		Sys: &syscall.SysProcAttr{
			Setsid: true,
		},
	}

	proc, err := os.StartProcess(me, args, attr)
	if err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}
	proc.Release()

	return waitForDaemon(port, 10*time.Second)
}

func waitForDaemon(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		client, err := NewWSClient(port)
		if err == nil {
			client.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("daemon did not become ready within %v", timeout)
}

func StopDaemon() error {
	info, err := ReadDaemonInfo()
	if err != nil {
		return err
	}

	client, err := NewWSClient(info.Port)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w", err)
	}
	defer client.Close()

	meta := MetaInfo{}
	_, err = client.Stop(meta)
	return err
}

func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
