package daemon

type MountRequest struct {
	Name string `json:"name"`
}

type UnmountRequest struct {
	Name string `json:"name"`
}

type MountResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}



type ListResponse struct {
	Mounts []MountEntryStatus `json:"mounts"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

type ShutdownResponse struct {
	Success bool `json:"success"`
}
