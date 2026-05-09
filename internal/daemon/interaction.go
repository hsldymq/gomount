package daemon

import "sync"

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
	defer m.mu.Unlock()

	ch, ok := m.pending[id]
	if !ok {
		return false
	}

	delete(m.pending, id)
	ch <- payload
	close(ch)
	return true
}
