package sshtunnel

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTunnelRecordSerializesPID(t *testing.T) {
	record := TunnelRecord{
		LocalPort:  10500,
		RemoteAddr: "10.0.0.2:445",
		JumpHost:   "bastion",
		PID:        12345,
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("marshal record: %v", err)
	}
	if !strings.Contains(string(data), `"pid":12345`) {
		t.Fatalf("expected pid in serialized record, got %s", string(data))
	}
}

func TestParsePIDs(t *testing.T) {
	pids := parsePIDs([]byte("123\n456\n"))
	if len(pids) != 2 || pids[0] != 123 || pids[1] != 456 {
		t.Fatalf("unexpected pids: %v", pids)
	}
}
