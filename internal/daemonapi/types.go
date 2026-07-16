package daemonapi

import "fmt"

type Source struct {
	URL      string `json:"url"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Path     string `json:"path,omitempty"`
}

type EntrySnapshot struct {
	Name         string         `json:"name"`
	Type         string         `json:"type"`
	MountDirPath string         `json:"mount_dir_path"`
	Source       Source         `json:"source"`
	Options      map[string]any `json:"options,omitempty"`
}

func (e EntrySnapshot) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if e.MountDirPath == "" {
		return fmt.Errorf("mount_dir_path is required")
	}
	if e.Type == "webdav" && e.Source.URL == "" {
		return fmt.Errorf("webdav source url is required")
	}
	return nil
}

type BatchRequest struct {
	Entries []EntrySnapshot `json:"entries"`
}

type OperationResult struct {
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Managed      bool          `json:"managed"`
	Handled      bool          `json:"handled"`
	Success      bool          `json:"success"`
	Skipped      bool          `json:"skipped,omitempty"`
	Mounted      bool          `json:"mounted"`
	Backend      string        `json:"backend,omitempty"`
	MountDirPath string        `json:"mount_dir_path,omitempty"`
	LogPath      string        `json:"log_path,omitempty"`
	Message      string        `json:"message,omitempty"`
	Error        *ErrorPayload `json:"error,omitempty"`
}

type BatchResponse struct {
	Results []OperationResult `json:"results"`
}

type StatusResponse struct {
	Statuses []OperationResult `json:"statuses"`
}

type HealthResponse struct {
	OK              bool     `json:"ok"`
	Version         string   `json:"version"`
	PID             int      `json:"pid"`
	ManagedTypes    []string `json:"managed_types"`
	MountedSessions int      `json:"mounted_sessions"`
}

type ShutdownResponse struct {
	OK        bool            `json:"ok"`
	Unmounted int             `json:"unmounted"`
	Errors    []ShutdownError `json:"errors,omitempty"`
}

type ShutdownError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type ErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func ManagedTypes() []string {
	return []string{"webdav"}
}

func IsManagedType(t string) bool {
	return t == "webdav"
}
