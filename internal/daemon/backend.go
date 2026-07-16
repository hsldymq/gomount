package daemon

import (
	"fmt"
	"sync"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

type MountSession interface {
	Unmount() error
	Done() <-chan error
}

type Mounter interface {
	Mount(entry daemonapi.EntrySnapshot) (MountSession, error)
}

type SessionManager struct {
	mounter        Mounter
	sessions       map[string]MountSession
	managedTypes   map[string]bool
	cleanupSignals chan struct{}
	mu             sync.Mutex
}

func NewSessionManager(mounter Mounter) *SessionManager {
	managedTypes := daemonapi.ManagedTypes()
	if mounter == nil {
		mounter = NewRcloneMounter()
		managedTypes = SupportedManagedTypes()
	}
	managed := make(map[string]bool, len(managedTypes))
	for _, t := range managedTypes {
		managed[t] = true
	}
	return &SessionManager{
		mounter:        mounter,
		sessions:       make(map[string]MountSession),
		managedTypes:   managed,
		cleanupSignals: make(chan struct{}, 16),
	}
}

func (m *SessionManager) ManagedTypes() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	types := make([]string, 0, len(m.managedTypes))
	for t := range m.managedTypes {
		types = append(types, t)
	}
	return types
}

func (m *SessionManager) MountedSessionCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

func (m *SessionManager) UnmountAll() (int, []daemonapi.ShutdownError) {
	m.mu.Lock()
	sessions := make(map[string]MountSession, len(m.sessions))
	for name, session := range m.sessions {
		sessions[name] = session
	}
	m.mu.Unlock()

	unmounted := 0
	var errs []daemonapi.ShutdownError
	for name, session := range sessions {
		if err := session.Unmount(); err != nil {
			errs = append(errs, daemonapi.ShutdownError{Name: name, Message: err.Error()})
			continue
		}
		m.mu.Lock()
		delete(m.sessions, name)
		m.mu.Unlock()
		unmounted++
	}
	return unmounted, errs
}

func (m *SessionManager) Mount(entry daemonapi.EntrySnapshot) daemonapi.OperationResult {
	if !m.isManaged(entry.Type) {
		return unmanagedResult(entry)
	}
	if err := entry.Validate(); err != nil {
		return invalidRequestResult(entry, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[entry.Name]; ok {
		return daemonapi.OperationResult{
			Name:         entry.Name,
			Type:         entry.Type,
			Managed:      true,
			Handled:      true,
			Success:      true,
			Skipped:      true,
			Mounted:      true,
			Backend:      "rclone-lib",
			MountDirPath: entry.MountDirPath,
			Message:      "already mounted",
		}
	}

	session, err := m.mounter.Mount(entry)
	if err != nil {
		return daemonapi.OperationResult{
			Name:         entry.Name,
			Type:         entry.Type,
			Managed:      true,
			Handled:      true,
			Success:      false,
			Mounted:      false,
			Backend:      "rclone-lib",
			MountDirPath: entry.MountDirPath,
			Message:      "mount failed",
			Error: &daemonapi.ErrorPayload{
				Code:    "backend_mount_failed",
				Message: "failed to mount webdav backend",
				Detail:  err.Error(),
			},
		}
	}

	m.sessions[entry.Name] = session
	m.watchSession(entry.Name, session)
	return daemonapi.OperationResult{
		Name:         entry.Name,
		Type:         entry.Type,
		Managed:      true,
		Handled:      true,
		Success:      true,
		Mounted:      true,
		Backend:      "rclone-lib",
		MountDirPath: entry.MountDirPath,
		Message:      "mounted",
	}
}

func (m *SessionManager) Unmount(entry daemonapi.EntrySnapshot) daemonapi.OperationResult {
	if !m.isManaged(entry.Type) {
		return unmanagedResult(entry)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[entry.Name]
	if !ok {
		return daemonapi.OperationResult{
			Name:         entry.Name,
			Type:         entry.Type,
			Managed:      true,
			Handled:      true,
			Success:      true,
			Skipped:      true,
			Mounted:      false,
			Backend:      "rclone-lib",
			MountDirPath: entry.MountDirPath,
			Message:      "already unmounted",
		}
	}

	if err := session.Unmount(); err != nil {
		return daemonapi.OperationResult{
			Name:         entry.Name,
			Type:         entry.Type,
			Managed:      true,
			Handled:      true,
			Success:      false,
			Mounted:      true,
			Backend:      "rclone-lib",
			MountDirPath: entry.MountDirPath,
			Message:      "unmount failed",
			Error: &daemonapi.ErrorPayload{
				Code:    "backend_unmount_failed",
				Message: "failed to unmount webdav backend",
				Detail:  err.Error(),
			},
		}
	}

	delete(m.sessions, entry.Name)
	return daemonapi.OperationResult{
		Name:         entry.Name,
		Type:         entry.Type,
		Managed:      true,
		Handled:      true,
		Success:      true,
		Mounted:      false,
		Backend:      "rclone-lib",
		MountDirPath: entry.MountDirPath,
		Message:      "unmounted",
	}
}

func (m *SessionManager) Status(entry daemonapi.EntrySnapshot) daemonapi.OperationResult {
	if !m.isManaged(entry.Type) {
		return unmanagedResult(entry)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	_, mounted := m.sessions[entry.Name]
	message := "not mounted"
	if mounted {
		message = "mounted"
	}
	return daemonapi.OperationResult{
		Name:         entry.Name,
		Type:         entry.Type,
		Managed:      true,
		Handled:      true,
		Success:      true,
		Mounted:      mounted,
		Backend:      "rclone-lib",
		MountDirPath: entry.MountDirPath,
		Message:      message,
	}
}

func (m *SessionManager) isManaged(t string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.managedTypes[t]
}

func (m *SessionManager) watchSession(name string, session MountSession) {
	done := session.Done()
	if done == nil {
		return
	}
	go func() {
		<-done
		m.mu.Lock()
		delete(m.sessions, name)
		m.mu.Unlock()
		select {
		case m.cleanupSignals <- struct{}{}:
		default:
		}
	}()
}

func (m *SessionManager) waitForSessionCleanup() {
	<-m.cleanupSignals
}

func invalidRequestResult(entry daemonapi.EntrySnapshot, err error) daemonapi.OperationResult {
	return daemonapi.OperationResult{
		Name:         entry.Name,
		Type:         entry.Type,
		Managed:      true,
		Handled:      true,
		Success:      false,
		Mounted:      false,
		Backend:      "rclone-lib",
		MountDirPath: entry.MountDirPath,
		Message:      "invalid request",
		Error: &daemonapi.ErrorPayload{
			Code:    "invalid_request",
			Message: "invalid daemon mount request",
			Detail:  err.Error(),
		},
	}
}

func unmanagedResult(entry daemonapi.EntrySnapshot) daemonapi.OperationResult {
	return daemonapi.OperationResult{
		Name:         entry.Name,
		Type:         entry.Type,
		Managed:      false,
		Handled:      false,
		Success:      false,
		Mounted:      false,
		MountDirPath: entry.MountDirPath,
		Message:      fmt.Sprintf("type %s is not managed by daemon", entry.Type),
	}
}
