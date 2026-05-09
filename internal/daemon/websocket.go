package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Only localhost connections
	},
}

type WebSocketServer struct {
	handlers       *Handlers
	clients        map[string]*ClientConn
	mu             sync.RWMutex
	interactionMgr *InteractionManager
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

		go s.handleMessage(client, &msg)
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
	payload, _ := json.Marshal(ResultPayload{
		Status: "error",
		Error:  errMsg,
	})
	msg.Payload = payload
	s.sendMessage(client, msg)
}

func (s *WebSocketServer) sendMessage(client *ClientConn, msg *Message) {
	client.mu.Lock()
	defer client.mu.Unlock()

	data, _ := json.Marshal(msg)
	client.Conn.WriteMessage(websocket.TextMessage, data)
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

	resultPayload, _ := json.Marshal(ResultPayload{
		Status:  status,
		Message: fmt.Sprintf("Mount results: %v", results),
	})

	s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	})
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

	resultPayload, _ := json.Marshal(ResultPayload{
		Status:  status,
		Message: fmt.Sprintf("Unmount results: %v", results),
	})

	s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	})
}

func (s *WebSocketServer) handleList(ctx context.Context, client *ClientConn, msgID string) {
	entries := s.handlers.List()

	resultPayload, _ := json.Marshal(ResultPayload{
		Status: "success",
		Data:   entries,
	})

	s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	})
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

	resultPayload, _ := json.Marshal(ResultPayload{
		Status:  "success",
		Message: status.Message,
		Data:    status,
	})

	s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	})
}

func (s *WebSocketServer) handleStop(ctx context.Context, client *ClientConn, msgID string) {
	s.handlers.UnmountAll()

	resultPayload, _ := json.Marshal(ResultPayload{
		Status:  "success",
		Message: "daemon stopping",
	})

	s.sendMessage(client, &Message{
		Type:    MsgTypeResult,
		ID:      msgID,
		Payload: resultPayload,
	})

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
