package daemon

import "encoding/json"

const (
	// Message types
	MsgTypeCommand             = "command"
	MsgTypeInteraction         = "interaction"
	MsgTypeInteractionResponse = "interaction_response"
	MsgTypeResult              = "result"
	MsgTypeError               = "error"

	// Command actions
	ActionMount   = "mount"
	ActionUnmount = "unmount"
	ActionList    = "list"
	ActionStatus  = "status"
	ActionStop    = "stop"

	// Input types
	InputTypePassword = "password"
	InputTypeText     = "text"
	InputTypeConfirm  = "confirm"
)

// Message base structure
type Message struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

// CommandPayload command message payload
type CommandPayload struct {
	Action string   `json:"action"`
	Names  []string `json:"names,omitempty"`
	Meta   MetaInfo `json:"meta,omitempty"`
}

// MetaInfo user meta information
type MetaInfo struct {
	UID      int    `json:"uid"`
	GID      int    `json:"gid"`
	Username string `json:"username"`
	Home     string `json:"home"`
}

// InteractionPayload interaction request payload
type InteractionPayload struct {
	Prompt    string   `json:"prompt"`
	InputType string   `json:"input_type"`
	Mask      bool     `json:"mask,omitempty"`
	Options   []string `json:"options,omitempty"`
}

// InteractionResponsePayload interaction response payload
type InteractionResponsePayload struct {
	Value string `json:"value"`
}

// ResultPayload result payload
type ResultPayload struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// MountEntryStatus mount entry status
type MountEntryStatus struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	MountPath string `json:"mount_path"`
	Mounted   bool   `json:"mounted"`
}
