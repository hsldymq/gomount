package sshtunnel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type TunnelRecord struct {
	LocalPort  int    `json:"local_port"`
	RemoteAddr string `json:"remote_addr"`
	JumpHost   string `json:"jump_host"`
}

var (
	stateMu sync.Mutex
)

func stateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "gomount")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func stateFilePath() (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "tunnels.json"), nil
}

func loadState() (map[string]TunnelRecord, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]TunnelRecord), nil
		}
		return nil, err
	}

	var records map[string]TunnelRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func saveState(records map[string]TunnelRecord) error {
	path, err := stateFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func saveRecord(name string, record TunnelRecord) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	records, err := loadState()
	if err != nil {
		return err
	}
	records[name] = record
	return saveState(records)
}

func loadRecord(name string) (TunnelRecord, bool, error) {
	stateMu.Lock()
	defer stateMu.Unlock()

	records, err := loadState()
	if err != nil {
		return TunnelRecord{}, false, err
	}
	r, ok := records[name]
	return r, ok, nil
}

func deleteRecord(name string) error {
	stateMu.Lock()
	defer stateMu.Unlock()

	records, err := loadState()
	if err != nil {
		return err
	}
	delete(records, name)
	return saveState(records)
}
