package daemon

import (
	"errors"
	"testing"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

func TestSessionManagerStatusReturnsUnmanagedForSMB(t *testing.T) {
	mgr := NewSessionManager(nil)
	entry := daemonapi.EntrySnapshot{Name: "nas", Type: "smb", MountDirPath: "/mnt/nas"}

	status := mgr.Status(entry)
	if status.Managed {
		t.Fatal("expected smb to be unmanaged")
	}
	if status.Handled {
		t.Fatal("expected unmanaged status not to be handled")
	}
	if status.Success {
		t.Fatal("expected unmanaged status not to be a successful handled operation")
	}
}

func TestSessionManagerMountAndUnmountWebDAVWithFakeMounter(t *testing.T) {
	fake := &fakeMounter{}
	mgr := NewSessionManager(fake)
	entry := daemonapi.EntrySnapshot{
		Name:         "docs",
		Type:         "webdav",
		MountDirPath: "/mnt/docs",
		Source:       daemonapi.Source{URL: "https://cloud.example.com/dav", Path: "/docs"},
	}

	mountResult := mgr.Mount(entry)
	if !mountResult.Success || !mountResult.Mounted {
		t.Fatalf("expected successful mount, got %+v", mountResult)
	}
	if fake.mountedURL != "https://cloud.example.com/dav" || fake.mountedPath != "/docs" {
		t.Fatalf("mounted source = %q %q", fake.mountedURL, fake.mountedPath)
	}

	status := mgr.Status(entry)
	if !status.Managed || !status.Mounted {
		t.Fatalf("expected mounted managed status, got %+v", status)
	}

	unmountResult := mgr.Unmount(entry)
	if !unmountResult.Success || unmountResult.Mounted {
		t.Fatalf("expected successful unmount, got %+v", unmountResult)
	}
	if !fake.unmounted {
		t.Fatal("expected fake session to be unmounted")
	}
}

func TestSessionManagerRemovesCompletedSession(t *testing.T) {
	done := make(chan error, 1)
	fake := &fakeMounter{done: done}
	mgr := NewSessionManager(fake)
	entry := daemonapi.EntrySnapshot{
		Name:         "docs",
		Type:         "webdav",
		MountDirPath: "/mnt/docs",
		Source:       daemonapi.Source{URL: "https://cloud.example.com/dav"},
	}

	result := mgr.Mount(entry)
	if !result.Mounted {
		t.Fatalf("expected mounted result, got %+v", result)
	}
	done <- nil
	mgr.waitForSessionCleanup()

	status := mgr.Status(entry)
	if status.Mounted {
		t.Fatalf("expected completed session to be removed, got %+v", status)
	}
}

func TestSessionManagerRejectsInvalidWebDAVSnapshot(t *testing.T) {
	mgr := NewSessionManager(&fakeMounter{})
	entry := daemonapi.EntrySnapshot{Name: "docs", Type: "webdav", MountDirPath: "/mnt/docs"}

	result := mgr.Mount(entry)
	if result.Success || result.Error == nil || result.Error.Code != "invalid_request" {
		t.Fatalf("expected invalid request result, got %+v", result)
	}
}

func TestSessionManagerMountedSessionCount(t *testing.T) {
	mgr := NewSessionManager(&fakeMounter{})
	entry := daemonapi.EntrySnapshot{Name: "docs", Type: "webdav", MountDirPath: "/mnt/docs", Source: daemonapi.Source{URL: "https://cloud.example.com/dav"}}

	if got := mgr.MountedSessionCount(); got != 0 {
		t.Fatalf("MountedSessionCount() = %d, want 0", got)
	}
	mgr.Mount(entry)
	if got := mgr.MountedSessionCount(); got != 1 {
		t.Fatalf("MountedSessionCount() = %d, want 1", got)
	}
}

func TestSessionManagerUnmountAllKeepsFailedSessions(t *testing.T) {
	fake := &fakeMounter{unmountErr: errors.New("busy")}
	mgr := NewSessionManager(fake)
	entry := daemonapi.EntrySnapshot{Name: "docs", Type: "webdav", MountDirPath: "/mnt/docs", Source: daemonapi.Source{URL: "https://cloud.example.com/dav"}}
	mgr.Mount(entry)

	unmounted, errs := mgr.UnmountAll()
	if unmounted != 0 || len(errs) != 1 || errs[0].Name != "docs" {
		t.Fatalf("expected failed unmount to be reported, got unmounted=%d errs=%+v", unmounted, errs)
	}
	if got := mgr.MountedSessionCount(); got != 1 {
		t.Fatalf("expected failed session to remain mounted, got count %d", got)
	}
}

type fakeMounter struct {
	mountedURL  string
	mountedPath string
	unmounted   bool
	unmountErr  error
	done        chan error
}

func (m *fakeMounter) Mount(entry daemonapi.EntrySnapshot) (MountSession, error) {
	m.mountedURL = entry.Source.URL
	m.mountedPath = entry.Source.Path
	return fakeSession{unmount: func() { m.unmounted = true }, unmountErr: m.unmountErr, done: m.done}, nil
}

type fakeSession struct {
	unmount    func()
	unmountErr error
	done       chan error
}

func (s fakeSession) Unmount() error {
	s.unmount()
	return s.unmountErr
}

func (s fakeSession) Done() <-chan error {
	return s.done
}
