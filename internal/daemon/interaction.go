package daemon

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// InteractionManager manages pending interaction requests and their responses.
type InteractionManager struct {
	pending map[string]chan *InteractionResponsePayload
	mu      sync.Mutex
}

// NewInteractionManager creates a new InteractionManager.
func NewInteractionManager() *InteractionManager {
	return &InteractionManager{
		pending: make(map[string]chan *InteractionResponsePayload),
	}
}

// Register registers a new interaction request and returns a channel to receive the response.
func (m *InteractionManager) Register(id string) chan *InteractionResponsePayload {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch := make(chan *InteractionResponsePayload, 1)
	m.pending[id] = ch
	return ch
}

// HandleResponse handles an interaction response for the given request ID.
// Returns true if the response was handled (i.e., a pending request was found).
func (m *InteractionManager) HandleResponse(id string, payload *InteractionResponsePayload) bool {
	m.mu.Lock()
	ch, exists := m.pending[id]
	if exists {
		delete(m.pending, id)
	}
	m.mu.Unlock()

	if !exists {
		return false
	}

	select {
	case ch <- payload:
		return true
	default:
		return false
	}
}

// RequestInput sends an interaction request to the client and waits for a response.
func (im *InteractionManager) RequestInput(client *ClientConn, prompt string, inputType string, mask bool) (string, error) {
	id := generateClientID()
	ch := make(chan *InteractionResponsePayload, 1)

	im.mu.Lock()
	im.pending[id] = ch
	im.mu.Unlock()

	defer func() {
		im.mu.Lock()
		delete(im.pending, id)
		im.mu.Unlock()
		// Drain the channel to prevent goroutine leak
		select {
		case <-ch:
		default:
		}
	}()

	// Send interaction request to client
	msg := &Message{
		Type: MsgTypeInteraction,
		ID:   id,
	}
	payload, err := json.Marshal(InteractionPayload{
		Prompt:    prompt,
		InputType: inputType,
		Mask:      mask,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal interaction payload: %w", err)
	}
	msg.Payload = payload

	client.mu.Lock()
	data, err := json.Marshal(msg)
	if err != nil {
		client.mu.Unlock()
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}
	err = client.Conn.WriteMessage(websocket.TextMessage, data)
	client.mu.Unlock()

	if err != nil {
		return "", fmt.Errorf("failed to send interaction request: %w", err)
	}

	// Wait for response with timeout
	select {
	case resp := <-ch:
		return resp.Value, nil
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("interaction timeout")
	}
}
