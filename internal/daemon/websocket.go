package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		host := r.Host
		if idx := strings.LastIndex(host, ":"); idx != -1 {
			host = host[:idx]
		}
		return host == "127.0.0.1" || host == "localhost" || host == "::1"
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebSocketServer struct {
	handlers       *Handlers
	clients        map[string]*ClientConn
	mu             sync.RWMutex
	interactionMgr *InteractionManager
	sem            chan struct{} // semaphore
}

type ClientConn struct {
	ID   string
	Conn *websocket.Conn
	mu   sync.Mutex
}

func NewWebSocketServer(handlers *Handlers) *WebSocketServer {
	return &WebSocketServer{
		handlers:       handlers,
		clients:        make(map[string]*ClientConn),
		interactionMgr: NewInteractionManager(),
		sem:            make(chan struct{}, 100),
	}
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	client := &ClientConn{
		ID:   generateClientID(),
		Conn: conn,
	}

	s.mu.Lock()
	s.clients[client.ID] = client
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, client.ID)
		s.mu.Unlock()
	}()

	s.handleConnection(client)
}

func (s *WebSocketServer) handleConnection(client *ClientConn) {
	client.Conn.SetReadLimit(1024 * 1024) // 1MB limit
	for {
		_, data, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			s.sendError(client, "", "invalid message format")
			continue
		}

		s.sem <- struct{}{}
		go func(m Message) {
			defer func() { <-s.sem }()
			s.handleMessage(client, &m)
		}(msg)
	}
}

func (s *WebSocketServer) handleMessage(client *ClientConn, msg *Message) {
	switch msg.Type {
	case MsgTypeCommand:
		s.handleCommand(client, msg)
	case MsgTypeInteractionResponse:
		s.handleInteractionResponse(client, msg)
	default:
		s.sendError(client, msg.ID, "unknown message type")
	}
}

func (s *WebSocketServer) sendError(client *ClientConn, id string, errMsg string) {
	msg := &Message{
		Type: MsgTypeError,
		ID:   id,
	}
	payload, err := json.Marshal(ResultPayload{
		Status: "error",
		Error:  errMsg,
	})
	if err != nil {
		// Log error but still try to send without payload
		msg.Payload = nil
	} else {
		msg.Payload = payload
	}
	if err := s.sendMessage(client, msg); err != nil {
		// Log error
		_ = err
	}
}

func (s *WebSocketServer) sendMessage(client *ClientConn, msg *Message) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}
	return nil
}

func generateClientID() string {
	// Simple ID generation - you can use uuid or random string
	return fmt.Sprintf("client-%d", time.Now().UnixNano())
}

func (s *WebSocketServer) handleCommand(client *ClientConn, msg *Message) {
	var payload CommandPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		s.sendError(client, msg.ID, "invalid command payload")
		return
	}

	ctx := context.WithValue(context.Background(), "client", client)
	ctx = context.WithValue(ctx, "msg_id", msg.ID)
	ctx = context.WithValue(ctx, "interaction_mgr", s.interactionMgr)

	switch payload.Action {
	case ActionMount:
		s.handleMount(ctx, client, msg.ID, payload)
	case ActionUnmount:
		s.handleUnmount(ctx, client, msg.ID, payload)
	case ActionList:
		s.handleList(ctx, client, msg.ID)
	case ActionStatus:
		s.handleStatus(ctx, client, msg.ID, payload)
	case ActionStop:
		s.handleStop(ctx, client, msg.ID)
	default:
		s.sendError(client, msg.ID, "unknown action: "+payload.Action)
	}
}

func (s *WebSocketServer) handleMount(ctx context.Context, client *ClientConn, msgID string, payload CommandPayload) {
	var results []string
	var successCount, failCount int

	for _, name := range payload.Names {
		msg, err := s.handlers.Mount(ctx, name)
		if err != nil {
			failCount++
			results = append(results, fmt.Sprintf("%s: failed - %v", name, err))
		} else {
			successCount++
			results = append(results, fmt.Sprintf("%s: %s", name, msg))
		}
	}

	status := "success"
	if failCount > 0 {
		if successCount > 0 {
			status = "partial"
		} else {
			status = "error"
		}
	}

	resultPayload, err := json.Marshal(ResultPayload{
		Status:  status,
		Message: fmt.Sprintf("Mount results: %v", results),
	})
	if err != nil {
		s.sendError(client, msgID, "failed to marshal result")
		return
	}

	if err := s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	}); err != nil {
		// Log error
		_ = err
	}
}

func (s *WebSocketServer) handleUnmount(ctx context.Context, client *ClientConn, msgID string, payload CommandPayload) {
	var results []string
	var successCount, failCount int

	for _, name := range payload.Names {
		msg, err := s.handlers.Unmount(ctx, name)
		if err != nil {
			failCount++
			results = append(results, fmt.Sprintf("%s: failed - %v", name, err))
		} else {
			successCount++
			results = append(results, fmt.Sprintf("%s: %s", name, msg))
		}
	}

	status := "success"
	if failCount > 0 {
		if successCount > 0 {
			status = "partial"
		} else {
			status = "error"
		}
	}

	resultPayload, err := json.Marshal(ResultPayload{
		Status:  status,
		Message: fmt.Sprintf("Unmount results: %v", results),
	})
	if err != nil {
		s.sendError(client, msgID, "failed to marshal result")
		return
	}

	if err := s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	}); err != nil {
		// Log error
		_ = err
	}
}

func (s *WebSocketServer) handleList(ctx context.Context, client *ClientConn, msgID string) {
	entries := s.handlers.List()

	resultPayload, err := json.Marshal(ResultPayload{
		Status: "success",
		Data:   entries,
	})
	if err != nil {
		s.sendError(client, msgID, "failed to marshal result")
		return
	}

	if err := s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	}); err != nil {
		// Log error
		_ = err
	}
}

func (s *WebSocketServer) handleStatus(ctx context.Context, client *ClientConn, msgID string, payload CommandPayload) {
	if len(payload.Names) == 0 {
		s.sendError(client, msgID, "status requires a name")
		return
	}

	name := payload.Names[0]
	status, err := s.handlers.Status(ctx, name)
	if err != nil {
		s.sendError(client, msgID, err.Error())
		return
	}

	resultPayload, err := json.Marshal(ResultPayload{
		Status:  "success",
		Message: status.Message,
		Data:    status,
	})
	if err != nil {
		s.sendError(client, msgID, "failed to marshal result")
		return
	}

	if err := s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	}); err != nil {
		// Log error
		_ = err
	}
}

func (s *WebSocketServer) handleStop(ctx context.Context, client *ClientConn, msgID string) {
	s.handlers.UnmountAll()

	resultPayload, err := json.Marshal(ResultPayload{
		Status:  "success",
		Message: "daemon stopping",
	})
	if err != nil {
		s.sendError(client, msgID, "failed to marshal result")
		return
	}

	if err := s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	}); err != nil {
		// Log error
		_ = err
	}

	// Trigger shutdown
	go s.handlers.Shutdown()
}

func (s *WebSocketServer) handleInteractionResponse(client *ClientConn, msg *Message) {
	var payload InteractionResponsePayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return
	}

	if s.interactionMgr.HandleResponse(msg.ID, &payload) {
		return
	}
}
