package sshtunnel

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const basePort = 10500

func findFreePort() (int, error) {
	for port := basePort; port < basePort+1000; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			ln.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port in range %d-%d", basePort, basePort+1000)
}

func Establish(ctx context.Context, jumpHost, remoteAddr string) (int, error) {
	localPort, err := findFreePort()
	if err != nil {
		return 0, fmt.Errorf("failed to allocate local port: %w", err)
	}

	args := []string{
		"-NL",
		fmt.Sprintf("%d:%s", localPort, remoteAddr),
		jumpHost,
	}

	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("ssh tunnel failed to start: %w", err)
	}

	if err := waitForPort(localPort, 15*time.Second); err != nil {
		cmd.Process.Kill()
		return 0, fmt.Errorf("ssh tunnel failed to establish: %w", err)
	}

	record := TunnelRecord{
		LocalPort:  localPort,
		RemoteAddr: remoteAddr,
		JumpHost:   jumpHost,
		PID:        cmd.Process.Pid,
	}
	if err := saveRecord("_tunnel_"+fmt.Sprintf("%d", localPort), record); err != nil {
		return 0, fmt.Errorf("failed to save tunnel state: %w", err)
	}

	return localPort, nil
}

func EstablishForMount(ctx context.Context, mountName, jumpHost, remoteAddr string) (int, error) {
	existing, found, err := loadRecord(mountName)
	if err != nil {
		return 0, err
	}
	if found && isPortAlive(existing.LocalPort) {
		return existing.LocalPort, nil
	}

	localPort, err := Establish(ctx, jumpHost, remoteAddr)
	if err != nil {
		return 0, err
	}

	if err := saveRecord(mountName, TunnelRecord{
		LocalPort:  localPort,
		RemoteAddr: remoteAddr,
		JumpHost:   jumpHost,
		PID:        localRecordPID(localPort),
	}); err != nil {
		return 0, err
	}

	return localPort, nil
}

func Teardown(mountName string) error {
	record, found, err := loadRecord(mountName)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	if record.PID > 0 {
		killByPID(record.PID)
	} else {
		killByPort(record.LocalPort)
	}

	if err := deleteRecord(mountName); err != nil {
		return fmt.Errorf("failed to clean tunnel state: %w", err)
	}

	return nil
}

func isPortAlive(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func localRecordPID(port int) int {
	record, found, err := loadRecord("_tunnel_" + fmt.Sprintf("%d", port))
	if err != nil || !found {
		return 0
	}
	return record.PID
}

func killByPID(pid int) {
	process, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = process.Signal(syscall.SIGTERM)
}

func killByPort(port int) {
	pid := localRecordPID(port)
	if pid > 0 {
		killByPID(pid)
		return
	}

	out, err := exec.Command("lsof", "-ti", fmt.Sprintf("tcp:%d", port)).Output()
	if err != nil {
		return
	}
	for _, pid := range parsePIDs(out) {
		killByPID(pid)
	}
}

func parsePIDs(out []byte) []int {
	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids
}

func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for port %d", port)
}
