package interaction

import (
	"os"
	"os/user"
	"path/filepath"
	"testing"
)

func TestGetCurrentUser(t *testing.T) {
	u, err := GetCurrentUser()
	if err != nil {
		t.Fatalf("GetCurrentUser() error = %v", err)
	}
	if u.Uid == "" {
		t.Error("expected non-empty Uid")
	}
	if u.Gid == "" {
		t.Error("expected non-empty Gid")
	}
}

func TestGetFileOwner(t *testing.T) {
	dir := t.TempDir()
	uid, gid, err := GetFileOwner(dir)
	if err != nil {
		t.Fatalf("GetFileOwner() error = %v", err)
	}

	currentUser, _ := user.Current()
	expectedUid := atoiOr(currentUser.Uid, -1)
	expectedGid := atoiOr(currentUser.Gid, -1)

	if uid != expectedUid {
		t.Errorf("expected uid %d, got %d", expectedUid, uid)
	}
	if gid != expectedGid {
		t.Errorf("expected gid %d, got %d", expectedGid, gid)
	}
}

func TestGetFileOwner_NonExistent(t *testing.T) {
	_, _, err := GetFileOwner("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestMkdirAll(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "a", "b", "c")
	if err := MkdirAll(target, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestChown(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("skipping: TestChown requires root privileges (os.Chown needs CAP_CHOWN)")
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "testfile")
	if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	currentUser, _ := user.Current()
	uid := atoiOr(currentUser.Uid, -1)
	gid := atoiOr(currentUser.Gid, -1)

	if err := Chown(file, uid, gid); err != nil {
		t.Fatalf("Chown() error = %v", err)
	}

	gotUid, gotGid, err := GetFileOwner(file)
	if err != nil {
		t.Fatalf("GetFileOwner() error = %v", err)
	}
	if gotUid != uid {
		t.Errorf("expected uid %d, got %d", uid, gotUid)
	}
	if gotGid != gid {
		t.Errorf("expected gid %d, got %d", gid, gotGid)
	}
}

func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}
