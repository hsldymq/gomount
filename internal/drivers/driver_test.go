package drivers

import (
	"context"
	"fmt"
	"testing"

	"github.com/hsldymq/gomount/internal/config"
)

// MockDriver 用于测试的模拟驱动
type MockDriver struct {
	mockType       string
	mountCalled    bool
	unmountCalled  bool
	statusCalled   bool
	validateCalled bool
	mountError     error
	unmountError   error
	statusResult   *MountStatus
	statusError    error
	validateError  error
}

func (m *MockDriver) Type() string {
	return m.mockType
}

func (m *MockDriver) Mount(ctx context.Context, entry *config.MountEntry) error {
	m.mountCalled = true
	return m.mountError
}

func (m *MockDriver) Unmount(ctx context.Context, entry *config.MountEntry) error {
	m.unmountCalled = true
	return m.unmountError
}

func (m *MockDriver) Status(ctx context.Context, entry *config.MountEntry) (*MountStatus, error) {
	m.statusCalled = true
	return m.statusResult, m.statusError
}

func (m *MockDriver) Validate(entry *config.MountEntry) error {
	m.validateCalled = true
	return m.validateError
}

func TestDriverRegistry_RegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	mockDriver := &MockDriver{mockType: "mock"}

	// 注册驱动
	registry.Register(mockDriver)

	// 获取驱动
	driver, ok := registry.Get("mock")
	if !ok {
		t.Fatal("expected to get driver, but got none")
	}

	if driver.Type() != "mock" {
		t.Errorf("expected type 'mock', got '%s'", driver.Type())
	}
}

func TestDriverRegistry_GetNotFound(t *testing.T) {
	registry := NewRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("expected not found, but got a driver")
	}
}

func TestDriverRegistry_Detect_WithType(t *testing.T) {
	registry := NewRegistry()

	// 注册驱动
	registry.Register(&MockDriver{mockType: "smb"})
	registry.Register(&MockDriver{mockType: "sshfs"})

	// 测试显式类型
	entry := &config.MountEntry{
		Name: "test",
		Type: "smb",
	}

	driver, err := registry.Detect(entry)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if driver.Type() != "smb" {
		t.Errorf("expected smb driver, got %s", driver.Type())
	}
}

func TestDriverRegistry_Detect_UnknownType(t *testing.T) {
	registry := NewRegistry()

	entry := &config.MountEntry{
		Name: "test",
		Type: "unknown",
	}

	_, err := registry.Detect(entry)
	if err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

func TestDriverRegistry_Detect_EmptyType(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&MockDriver{mockType: "smb"})

	entry := &config.MountEntry{
		Name: "test",
	}

	_, err := registry.Detect(entry)
	if err == nil {
		t.Error("expected error when type is empty, got nil")
	}
}

func TestDriverRegistry_List(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&MockDriver{mockType: "smb"})
	registry.Register(&MockDriver{mockType: "sshfs"})

	types := registry.List()
	if len(types) != 2 {
		t.Errorf("expected 2 types, got %d", len(types))
	}
}

func TestDriverError(t *testing.T) {
	err := &DriverError{
		Driver: "smb",
		Op:     "mount",
		Entry:  "test",
		Err:    fmt.Errorf("test error"),
	}

	expected := "smb driver mount test: test error"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}

	if err.Unwrap().Error() != "test error" {
		t.Error("Unwrap() should return original error")
	}
}
