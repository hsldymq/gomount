package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/term"
)

type WSClient struct {
	conn    *websocket.Conn
	mu      sync.Mutex
	pending map[string]chan *Message
}

func NewWSClient(port int) (*WSClient, error) {
	u := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("127.0.0.1:%d", port),
		Path:   "/ws",
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &WSClient{
		conn:    conn,
		pending: make(map[string]chan *Message),
	}

	go client.readLoop()

	return client, nil
}

func (c *WSClient) readLoop() {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		c.mu.Lock()
		ch, exists := c.pending[msg.ID]
		c.mu.Unlock()

		if exists {
			ch <- &msg
		}
	}
}

func (c *WSClient) Send(msg *Message) (*Message, error) {
	c.mu.Lock()
	ch := make(chan *Message, 1)
	c.pending[msg.ID] = ch
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, msg.ID)
		c.mu.Unlock()
	}()

	data, _ := json.Marshal(msg)
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return nil, err
	}

	resp := <-ch
	return resp, nil
}

func (c *WSClient) Close() {
	c.conn.Close()
}

func (c *WSClient) Mount(names []string, meta MetaInfo) (*ResultPayload, error) {
	return c.executeCommand(ActionMount, names, meta)
}

func (c *WSClient) Unmount(names []string, meta MetaInfo) (*ResultPayload, error) {
	return c.executeCommand(ActionUnmount, names, meta)
}

func (c *WSClient) List(meta MetaInfo) (*ResultPayload, error) {
	return c.executeCommand(ActionList, nil, meta)
}

func (c *WSClient) Stop(meta MetaInfo) (*ResultPayload, error) {
	return c.executeCommand(ActionStop, nil, meta)
}

func (c *WSClient) executeCommand(action string, names []string, meta MetaInfo) (*ResultPayload, error) {
	payload, _ := json.Marshal(CommandPayload{
		Action: action,
		Names:  names,
		Meta:   meta,
	})

	msg := &Message{
		Type:    MsgTypeCommand,
		ID:      generateID(),
		Payload: payload,
	}

	resp, err := c.Send(msg)
	if err != nil {
		return nil, err
	}

	// Handle interaction requests
	for resp.Type == MsgTypeInteraction {
		var interaction InteractionPayload
		json.Unmarshal(resp.Payload, &interaction)

		// Prompt user for input
		value := promptUser(interaction.Prompt, interaction.InputType, interaction.Mask)

		respPayload, _ := json.Marshal(InteractionResponsePayload{Value: value})
		respMsg := &Message{
			Type:    MsgTypeInteractionResponse,
			ID:      resp.ID,
			Payload: respPayload,
		}

		resp, err = c.Send(respMsg)
		if err != nil {
			return nil, err
		}
	}

	var result ResultPayload
	json.Unmarshal(resp.Payload, &result)
	return &result, nil
}

func promptUser(prompt string, inputType string, mask bool) string {
	fmt.Print(prompt + " ")

	if mask && inputType == InputTypePassword {
		bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		return string(bytePassword)
	}

	reader := bufio.NewReader(os.Stdin)
	value, _ := reader.ReadString('\n')
	return strings.TrimSpace(value)
}

func generateID() string {
	return fmt.Sprintf("msg-%d", time.Now().UnixNano())
}
