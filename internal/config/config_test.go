package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestFile(t *testing.T, dir string, name string, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_BasicMounts(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `
mounts:
  - name: test
    type: sshfs
    mount_dir_path: ~/mnt/test
    sshfs:
      host: example.com
      remote_path: /data
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(cfg.Mounts))
	}
	if cfg.Mounts[0].Name != "test" {
		t.Errorf("expected name 'test', got '%s'", cfg.Mounts[0].Name)
	}
}

func TestLoadConfig_EmptyConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `{}
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected 0 mounts, got %d", len(cfg.Mounts))
	}
}

func TestLoadConfig_IncludeSingleFile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "shared.yaml", `
mounts:
  - name: shared
    type: sshfs
    mount_dir_path: ~/mnt/shared
    sshfs:
      host: shared.example.com
      remote_path: /data
`)
	path := writeTestFile(t, dir, "config.yaml", `
include:
  - shared.yaml
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 1 {
		t.Fatalf("expected 1 mount, got %d", len(cfg.Mounts))
	}
	if cfg.Mounts[0].Name != "shared" {
		t.Errorf("expected name 'shared', got '%s'", cfg.Mounts[0].Name)
	}
}

func TestLoadConfig_IncludeWithLocalMounts(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "shared.yaml", `
mounts:
  - name: shared
    type: sshfs
    mount_dir_path: ~/mnt/shared
    sshfs:
      host: shared.example.com
      remote_path: /data
`)
	path := writeTestFile(t, dir, "config.yaml", `
mounts:
  - name: local
    type: sshfs
    mount_dir_path: ~/mnt/local
    sshfs:
      host: local.example.com
      remote_path: /local

include:
  - shared.yaml
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(cfg.Mounts))
	}
	if cfg.Mounts[0].Name != "local" {
		t.Errorf("expected first mount 'local', got '%s'", cfg.Mounts[0].Name)
	}
	if cfg.Mounts[1].Name != "shared" {
		t.Errorf("expected second mount 'shared', got '%s'", cfg.Mounts[1].Name)
	}
}

func TestLoadConfig_IncludeGlob(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "includes")
	os.MkdirAll(subdir, 0755)
	writeTestFile(t, subdir, "a.yaml", `
mounts:
  - name: alpha
    type: sshfs
    mount_dir_path: ~/mnt/a
    sshfs:
      host: a.example.com
      remote_path: /a
`)
	writeTestFile(t, subdir, "b.yaml", `
mounts:
  - name: beta
    type: sshfs
    mount_dir_path: ~/mnt/b
    sshfs:
      host: b.example.com
      remote_path: /b
`)
	path := writeTestFile(t, dir, "config.yaml", `
include:
  - "includes/*.yaml"
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(cfg.Mounts))
	}
}

func TestLoadConfig_IncludeNotFoundWarning(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `
include:
  - nonexistent.yaml
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected 0 mounts, got %d", len(cfg.Mounts))
	}
}

func TestLoadConfig_GlobNoMatchSilent(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `
include:
  - "nomatch-*.yaml"
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 0 {
		t.Errorf("expected 0 mounts, got %d", len(cfg.Mounts))
	}
}

func TestLoadConfig_CircularInclude(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.yaml", `
include:
  - b.yaml
mounts:
  - name: from-a
    type: sshfs
    mount_dir_path: ~/mnt/a
    sshfs:
      host: a.example.com
      remote_path: /a
`)
	writeTestFile(t, dir, "b.yaml", `
include:
  - a.yaml
mounts:
  - name: from-b
    type: sshfs
    mount_dir_path: ~/mnt/b
    sshfs:
      host: b.example.com
      remote_path: /b
`)
	cfg, err := LoadConfig(filepath.Join(dir, "a.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 2 {
		t.Fatalf("expected 2 mounts, got %d", len(cfg.Mounts))
	}
}

func TestLoadConfig_NameConflict(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "shared.yaml", `
mounts:
  - name: dup
    type: sshfs
    mount_dir_path: ~/mnt/shared
    sshfs:
      host: shared.example.com
      remote_path: /data
`)
	path := writeTestFile(t, dir, "config.yaml", `
mounts:
  - name: dup
    type: sshfs
    mount_dir_path: ~/mnt/local
    sshfs:
      host: local.example.com
      remote_path: /local

include:
  - shared.yaml
`)
	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("expected error for name conflict")
	}
}

func TestLoadConfig_SortingByName(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `
mounts:
  - name: charlie
    type: sshfs
    mount_dir_path: ~/mnt/c
    sshfs:
      host: c.example.com
      remote_path: /c
  - name: alpha
    type: sshfs
    mount_dir_path: ~/mnt/a
    sshfs:
      host: a.example.com
      remote_path: /a
  - name: bravo
    type: sshfs
    mount_dir_path: ~/mnt/b
    sshfs:
      host: b.example.com
      remote_path: /b

sorting:
  by: name
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Mounts) != 3 {
		t.Fatalf("expected 3 mounts, got %d", len(cfg.Mounts))
	}
	names := []string{cfg.Mounts[0].Name, cfg.Mounts[1].Name, cfg.Mounts[2].Name}
	expected := []string{"alpha", "bravo", "charlie"}
	for i, n := range expected {
		if names[i] != n {
			t.Errorf("position %d: expected '%s', got '%s'", i, n, names[i])
		}
	}
}

func TestLoadConfig_SortingByNameDesc(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `
mounts:
  - name: alpha
    type: sshfs
    mount_dir_path: ~/mnt/a
    sshfs:
      host: a.example.com
      remote_path: /a
  - name: bravo
    type: sshfs
    mount_dir_path: ~/mnt/b
    sshfs:
      host: b.example.com
      remote_path: /b

sorting:
  by:
    - "-name"
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mounts[0].Name != "bravo" {
		t.Errorf("expected first 'bravo', got '%s'", cfg.Mounts[0].Name)
	}
	if cfg.Mounts[1].Name != "alpha" {
		t.Errorf("expected second 'alpha', got '%s'", cfg.Mounts[1].Name)
	}
}

func TestLoadConfig_IncludedFileSortingIgnored(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "shared.yaml", `
mounts:
  - name: z-shared
    type: sshfs
    mount_dir_path: ~/mnt/z
    sshfs:
      host: z.example.com
      remote_path: /z
  - name: a-shared
    type: sshfs
    mount_dir_path: ~/mnt/a
    sshfs:
      host: a.example.com
      remote_path: /a

sorting:
  by: name
`)
	path := writeTestFile(t, dir, "config.yaml", `
include:
  - shared.yaml
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mounts[0].Name != "z-shared" {
		t.Errorf("sorting from included file should be ignored; expected 'z-shared' first, got '%s'", cfg.Mounts[0].Name)
	}
}

func TestLoadConfig_NoSorting(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "config.yaml", `
mounts:
  - name: zebra
    type: sshfs
    mount_dir_path: ~/mnt/z
    sshfs:
      host: z.example.com
      remote_path: /z
  - name: alpha
    type: sshfs
    mount_dir_path: ~/mnt/a
    sshfs:
      host: a.example.com
      remote_path: /a
`)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mounts[0].Name != "zebra" || cfg.Mounts[1].Name != "alpha" {
		t.Errorf("expected declaration order preserved: zebra, alpha; got %s, %s", cfg.Mounts[0].Name, cfg.Mounts[1].Name)
	}
}
