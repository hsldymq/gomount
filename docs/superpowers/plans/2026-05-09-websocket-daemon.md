# gomount WebSocket Daemon 重构实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 gomount 从 HTTP API 架构重构为 WebSocket 通信架构，支持交互式密码输入，解决 daemon 后台运行无法获取 sudo/ssh 密码的问题。

**Architecture:** CLI 作为短生命周期进程，通过 WebSocket 与长期运行的 daemon 通信。daemon 以 root 运行执行特权操作（mount.cifs），通过 WebSocket 向 CLI 请求用户输入。所有驱动统一通过 WebSocket 协议通信。

**Tech Stack:** Go 1.25, gorilla/websocket, rclone mountlib

---

## 文件结构

### 新增文件
- `internal/daemon/websocket.go` - WebSocket 服务器和客户端实现
- `internal/daemon/protocol.go` - WebSocket 消息协议定义
- `internal/daemon/privilege.go` - 特权操作管理（root 权限检查、sudo 启动）
- `internal/daemon/interaction.go` - 交互式输入处理

### 修改文件
- `cmd/gomount/main.go` - 修改 daemon 检测和启动逻辑
- `cmd/gomount/cmd_daemon.go` - 重构 daemon 命令
- `cmd/gomount/cmd_mount.go` - 适配 WebSocket 通信
- `cmd/gomount/cmd_unmount.go` - 适配 WebSocket 通信
- `cmd/gomount/cmd_list.go` - 适配 WebSocket 通信
- `cmd/gomount/cmd_interactive.go` - 适配 WebSocket 通信
- `cmd/gomount/cmd_mkdir.go` - 子命令改为 `c`
- `internal/daemon/daemon.go` - 移除 HTTP 相关代码，改为 WebSocket
- `internal/daemon/server.go` - 重构为 WebSocket 服务器
- `internal/daemon/client.go` - 重构为 WebSocket 客户端
- `internal/daemon/handler.go` - 适配 WebSocket 消息处理
- `internal/daemon/types.go` - 更新类型定义
- `internal/drivers/smb/driver.go` - 移除 sudo 包装，适配交互
- `internal/interaction/sudo.go` - 修改或移除（daemon 直接 root 执行）

### 删除文件
- `internal/daemon/client.go` - HTTP 客户端（被 WebSocket 替代）

---

## Task 1: WebSocket 协议定义

**Files:**
- Create: `internal/daemon/protocol.go`

- [ ] **Step 1: 定义消息类型常量**

```go
package daemon

const (
    // 消息类型
    MsgTypeCommand            = "command"
    MsgTypeInteraction        = "interaction"
    MsgTypeInteractionResponse = "interaction_response"
    MsgTypeResult             = "result"
    MsgTypeError              = "error"
    
    // 命令动作
    ActionMount   = "mount"
    ActionUnmount = "unmount"
    ActionList    = "list"
    ActionStatus  = "status"
    ActionStop    = "stop"
    
    // 输入类型
    InputTypePassword = "password"
    InputTypeText     = "text"
    InputTypeConfirm  = "confirm"
)
```

- [ ] **Step 2: 定义消息结构体**

```go
package daemon

// Message 基础消息结构
type Message struct {
    Type    string          `json:"type"`
    ID      string          `json:"id"`
    Payload json.RawMessage `json:"payload"`
}

// CommandPayload 命令消息载荷
type CommandPayload struct {
    Action string   `json:"action"`
    Names  []string `json:"names,omitempty"`
    Meta   MetaInfo `json:"meta,omitempty"`
}

// MetaInfo 用户元信息
type MetaInfo struct {
    UID      int    `json:"uid"`
    GID      int    `json:"gid"`
    Username string `json:"username"`
    Home     string `json:"home"`
}

// InteractionPayload 交互请求载荷
type InteractionPayload struct {
    Prompt    string `json:"prompt"`
    InputType string `json:"input_type"`
    Mask      bool   `json:"mask,omitempty"`
    Options   []string `json:"options,omitempty"`
}

// InteractionResponsePayload 交互响应载荷
type InteractionResponsePayload struct {
    Value string `json:"value"`
}

// ResultPayload 结果载荷
type ResultPayload struct {
    Status  string      `json:"status"`
    Message string      `json:"message,omitempty"`
    Error   string      `json:"error,omitempty"`
    Data    interface{} `json:"data,omitempty"`
}

// MountEntryStatus 挂载条目状态
type MountEntryStatus struct {
    Name      string `json:"name"`
    Type      string `json:"type"`
    MountPath string `json:"mount_path"`
    Mounted   bool   `json:"mounted"`
}
```

- [ ] **Step 3: 提交**

```bash
git add internal/daemon/protocol.go
git commit -m "feat: define WebSocket message protocol"
```

---

## Task 2: WebSocket 服务器实现

**Files:**
- Create: `internal/daemon/websocket.go`
- Modify: `internal/daemon/server.go`

- [ ] **Step 1: 创建 WebSocket 升级器**

```go
package daemon

import (
    "net/http"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // 只允许 localhost
    },
}
```

- [ ] **Step 2: 实现 WebSocket 连接处理**

```go
package daemon

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "sync"
    
    "github.com/gorilla/websocket"
)

type WebSocketServer struct {
    handlers *Handlers
    clients  map[string]*ClientConn
    mu       sync.RWMutex
}

type ClientConn struct {
    ID       string
    Conn     *websocket.Conn
    mu       sync.Mutex
    pending  map[string]chan *Message
}

func NewWebSocketServer(handlers *Handlers) *WebSocketServer {
    return &WebSocketServer{
        handlers: handlers,
        clients:  make(map[string]*ClientConn),
    }
}

func (s *WebSocketServer) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        return
    }
    defer conn.Close()
    
    client := &ClientConn{
        ID:      generateClientID(),
        Conn:    conn,
        pending: make(map[string]chan *Message),
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
```

- [ ] **Step 3: 实现命令处理**

```go
func (s *WebSocketServer) handleCommand(client *ClientConn, msg *Message) {
    var payload CommandPayload
    if err := json.Unmarshal(msg.Payload, &payload); err != nil {
        s.sendError(client, msg.ID, "invalid command payload")
        return
    }
    
    ctx := context.WithValue(context.Background(), "client", client)
    ctx = context.WithValue(ctx, "msg_id", msg.ID)
    
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
        s.sendError(client, msg.ID, "unknown action")
    }
}
```

- [ ] **Step 4: 提交**

```bash
git add internal/daemon/websocket.go internal/daemon/server.go
git commit -m "feat: implement WebSocket server"
```

---

## Task 3: 交互式输入处理

**Files:**
- Create: `internal/daemon/interaction.go`

- [ ] **Step 1: 实现交互请求发送**

```go
package daemon

import (
    "encoding/json"
    "fmt"
    "sync"
    "time"
    
    "github.com/google/uuid"
)

type InteractionManager struct {
    mu       sync.Mutex
    pending  map[string]chan *InteractionResponsePayload
}

func NewInteractionManager() *InteractionManager {
    return &InteractionManager{
        pending: make(map[string]chan *InteractionResponsePayload),
    }
}

func (im *InteractionManager) RequestInput(client *ClientConn, prompt string, inputType string, mask bool) (string, error) {
    id := uuid.New().String()
    
    ch := make(chan *InteractionResponsePayload, 1)
    im.mu.Lock()
    im.pending[id] = ch
    im.mu.Unlock()
    
    defer func() {
        im.mu.Lock()
        delete(im.pending, id)
        im.mu.Unlock()
    }()
    
    // 发送交互请求
    msg := &Message{
        Type: MsgTypeInteraction,
        ID:   id,
    }
    payload, _ := json.Marshal(InteractionPayload{
        Prompt:    prompt,
        InputType: inputType,
        Mask:      mask,
    })
    msg.Payload = payload
    
    client.mu.Lock()
    data, _ := json.Marshal(msg)
    err := client.Conn.WriteMessage(websocket.TextMessage, data)
    client.mu.Unlock()
    
    if err != nil {
        return "", fmt.Errorf("failed to send interaction request: %w", err)
    }
    
    // 等待响应（超时 30 秒）
    select {
    case resp := <-ch:
        return resp.Value, nil
    case <-time.After(30 * time.Second):
        return "", fmt.Errorf("interaction timeout")
    }
}

func (im *InteractionManager) HandleResponse(msgID string, payload *InteractionResponsePayload) bool {
    im.mu.Lock()
    ch, exists := im.pending[msgID]
    im.mu.Unlock()
    
    if !exists {
        return false
    }
    
    ch <- payload
    return true
}
```

- [ ] **Step 2: 在服务器中集成交互管理**

```go
// 在 WebSocketServer 中添加
func (s *WebSocketServer) handleInteractionResponse(client *ClientConn, msg *Message) {
    var payload InteractionResponsePayload
    if err := json.Unmarshal(msg.Payload, &payload); err != nil {
        return
    }
    
    // 转发给交互管理器
    if s.interactionMgr.HandleResponse(msg.ID, &payload) {
        return
    }
    
    // 如果没有对应的 pending 请求，忽略
}
```

- [ ] **Step 3: 提交**

```bash
git add internal/daemon/interaction.go
git commit -m "feat: implement interactive input handling"
```

---

## Task 4: WebSocket 客户端实现

**Files:**
- Create: `internal/daemon/client.go` (替换原有 HTTP 客户端)

- [ ] **Step 1: 实现 WebSocket 客户端连接**

```go
package daemon

import (
    "encoding/json"
    "fmt"
    "net/url"
    "sync"
    
    "github.com/gorilla/websocket"
)

type WSClient struct {
    conn     *websocket.Conn
    mu       sync.Mutex
    pending  map[string]chan *Message
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
    
    // 等待响应
    resp := <-ch
    return resp, nil
}

func (c *WSClient) Close() {
    c.conn.Close()
}
```

- [ ] **Step 2: 实现高层命令方法**

```go
func (c *WSClient) Mount(names []string, meta MetaInfo) (*ResultPayload, error) {
    payload, _ := json.Marshal(CommandPayload{
        Action: ActionMount,
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
    
    // 处理交互请求
    for resp.Type == MsgTypeInteraction {
        var interaction InteractionPayload
        json.Unmarshal(resp.Payload, &interaction)
        
        // 在终端提示用户输入
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
        // 使用终端密码输入（不显示字符）
        bytePassword, _ := term.ReadPassword(int(syscall.Stdin))
        fmt.Println()
        return string(bytePassword)
    }
    
    reader := bufio.NewReader(os.Stdin)
    value, _ := reader.ReadString('\n')
    return strings.TrimSpace(value)
}
```

- [ ] **Step 3: 提交**

```bash
git add internal/daemon/client.go
git commit -m "feat: implement WebSocket client"
```

---

## Task 5: 特权操作管理

**Files:**
- Create: `internal/daemon/privilege.go`

- [ ] **Step 1: 实现 root 权限检查和管理**

```go
package daemon

import (
    "fmt"
    "os"
    "os/exec"
    "os/user"
    "strconv"
    "syscall"
)

// IsRoot 检查当前是否以 root 运行
func IsRoot() bool {
    return os.Getuid() == 0
}

// GetOriginalUser 获取原始用户信息（当通过 sudo 启动时）
func GetOriginalUser() (*user.User, error) {
    // 优先从 sudo 环境变量获取
    if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
        return user.Lookup(sudoUser)
    }
    
    // 否则返回当前用户
    return user.Current()
}

// GetOriginalUID 获取原始用户 UID
func GetOriginalUID() int {
    if sudoUID := os.Getenv("SUDO_UID"); sudoUID != "" {
        uid, _ := strconv.Atoi(sudoUID)
        return uid
    }
    return os.Getuid()
}

// GetOriginalGID 获取原始用户 GID
func GetOriginalGID() int {
    if sudoGID := os.Getenv("SUDO_GID"); sudoGID != "" {
        gid, _ := strconv.Atoi(sudoGID)
        return gid
    }
    return os.Getgid()
}

// NeedsPrivilege 检查配置中是否有需要 root 的操作
func NeedsPrivilege(cfg *config.Config) bool {
    for _, entry := range cfg.Mounts {
        if entry.Type == "smb" {
            return true
        }
    }
    return false
}

// StartWithSudo 以 sudo 启动当前程序
func StartWithSudo(args ...string) error {
    me, err := os.Executable()
    if err != nil {
        return fmt.Errorf("cannot find executable: %w", err)
    }
    
    sudoArgs := append([]string{me}, args...)
    cmd := exec.Command("sudo", sudoArgs...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    
    return cmd.Run()
}

// DropPrivileges 降权到原始用户
func DropPrivileges() error {
    uid := GetOriginalUID()
    gid := GetOriginalGID()
    
    if uid == 0 {
        return nil // 已经是 root，不需要降权
    }
    
    if err := syscall.Setgid(gid); err != nil {
        return fmt.Errorf("failed to set gid: %w", err)
    }
    if err := syscall.Setuid(uid); err != nil {
        return fmt.Errorf("failed to set uid: %w", err)
    }
    
    return nil
}

// RunAsUser 以指定用户身份运行函数
func RunAsUser(uid, gid int, fn func() error) error {
    // 保存当前权限
    oldUID := os.Getuid()
    oldGID := os.Getgid()
    
    // 降权
    if err := syscall.Setgid(gid); err != nil {
        return err
    }
    if err := syscall.Setuid(uid); err != nil {
        return err
    }
    
    // 执行函数
    err := fn()
    
    // 恢复权限
    syscall.Setuid(oldUID)
    syscall.Setgid(oldGID)
    
    return err
}
```

- [ ] **Step 2: 提交**

```bash
git add internal/daemon/privilege.go
git commit -m "feat: implement privilege management"
```

---

## Task 6: 修改 SMB Driver

**Files:**
- Modify: `internal/drivers/smb/driver.go`

- [ ] **Step 1: 移除 sudo 包装，直接执行 mount.cifs**

```go
func (d *Driver) mountCIFS(ctx context.Context, entry *config.MountEntry) error {
    // ... 前面的代码保持不变 ...
    
    cmd := d.buildMountCommand(entry, credsFile, smbAddr, smbPort)
    
    // 直接执行，不需要 sudo（daemon 已经是 root）
    if err := interaction.RunCommand(cmd); err != nil {
        if entry.SSHTunnel != nil {
            sshtunnel.Teardown(entry.Name)
        }
        return &drivers.DriverError{
            Driver: d.Type(), Op: "mount", Entry: entry.Name,
            Err: &drivers.CommandError{Cmd: "mount.cifs", Err: err},
        }
    }
    
    return nil
}
```

- [ ] **Step 2: 修改 Unmount，直接执行 umount**

```go
func (d *Driver) Unmount(ctx context.Context, entry *config.MountEntry) error {
    if d.mountMgr.IsMounted(entry.MountDirPath) {
        // rclone 挂载，使用 mountMgr
        if err := d.mountMgr.Unmount(entry.MountDirPath); err != nil {
            return &drivers.DriverError{
                Driver: d.Type(), Op: "unmount", Entry: entry.Name,
                Err: err,
            }
        }
    } else {
        // 系统挂载，直接 umount（不需要 sudo）
        cmd := exec.CommandContext(ctx, "umount", entry.MountDirPath)
        if err := interaction.RunCommandSilent(cmd); err != nil {
            return &drivers.DriverError{
                Driver: d.Type(), Op: "unmount", Entry: entry.Name,
                Err: err,
            }
        }
    }
    
    if entry.SSHTunnel != nil {
        _ = sshtunnel.Teardown(entry.Name)
    }
    
    return nil
}
```

- [ ] **Step 3: 提交**

```bash
git add internal/drivers/smb/driver.go
git commit -m "refactor: remove sudo wrapper from SMB driver"
```

---

## Task 7: 修改 CLI 命令

**Files:**
- Modify: `cmd/gomount/cmd_mount.go`
- Modify: `cmd/gomount/cmd_unmount.go`
- Modify: `cmd/gomount/cmd_list.go`
- Modify: `cmd/gomount/cmd_interactive.go`
- Modify: `cmd/gomount/cmd_daemon.go`
- Modify: `cmd/gomount/cmd_mkdir.go`
- Modify: `cmd/gomount/main.go`

- [ ] **Step 1: 修改 cmd_mount.go**

```go
func runMount(cmd *cobra.Command, args []string) error {
    cfg, err := config.LoadConfig(configPath)
    if err != nil {
        return err
    }
    
    client, err := ensureDaemon(cfg)
    if err != nil {
        return err
    }
    defer client.Close()
    
    if len(args) == 0 {
        fmt.Fprintln(os.Stderr, "No mount entry specified. Use 'gomount interactive' for interactive selection.")
        return nil
    }
    
    // 获取当前用户信息
    meta := getMetaInfo()
    
    var failCount int
    for _, name := range args {
        result, err := client.Mount([]string{name}, meta)
        if err != nil {
            fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", name, err)
            failCount++
            continue
        }
        if result.Status == "success" {
            fmt.Printf("  %s: %s\n", name, result.Message)
        } else {
            fmt.Fprintf(os.Stderr, "  ERROR %s: %s\n", name, result.Error)
            failCount++
        }
    }
    
    if failCount > 0 && failCount == len(args) {
        return fmt.Errorf("%d mount(s) failed", failCount)
    }
    return nil
}

func getMetaInfo() daemon.MetaInfo {
    currentUser, _ := user.Current()
    uid, _ := strconv.Atoi(currentUser.Uid)
    gid, _ := strconv.Atoi(currentUser.Gid)
    
    home, _ := os.UserHomeDir()
    
    return daemon.MetaInfo{
        UID:      uid,
        GID:      gid,
        Username: currentUser.Username,
        Home:     home,
    }
}
```

- [ ] **Step 2: 修改 ensureDaemon 函数**

```go
func ensureDaemon(cfg *config.Config) (*daemon.WSClient, error) {
    // 尝试连接 WebSocket
    client, err := daemon.NewWSClient(daemon.DefaultPort)
    if err == nil {
        return client, nil
    }
    
    // 检查是否需要 root
    needsRoot := daemon.NeedsPrivilege(cfg)
    
    if needsRoot && !daemon.IsRoot() {
        // 需要 root 但当前不是 root，用 sudo 启动
        fmt.Println("Starting daemon with root privileges...")
        if err := daemon.StartWithSudo(os.Args[1:]...); err != nil {
            return nil, fmt.Errorf("failed to start daemon with sudo: %w", err)
        }
    } else {
        // 直接启动 daemon
        daemonCfg := daemon.DaemonConfig{Port: daemon.DefaultPort}
        if err := daemon.StartDaemon(configPath, daemonCfg); err != nil {
            return nil, fmt.Errorf("failed to start daemon: %w", err)
        }
    }
    
    // 再次尝试连接
    return daemon.NewWSClient(daemon.DefaultPort)
}
```

- [ ] **Step 3: 修改子命令简称**

```go
// cmd_mkdir.go
var mkdirCmd = &cobra.Command{
    Use:     "mkdir [path]",
    Aliases: []string{"c"},  // 改为 c
    // ...
}

// cmd_daemon.go
var daemonCmd = &cobra.Command{
    Use:     "daemon",
    Aliases: []string{"d"},  // 改为 d
    // ...
}
```

- [ ] **Step 4: 提交**

```bash
git add cmd/gomount/
git commit -m "refactor: adapt CLI commands to WebSocket architecture"
```

---

## Task 8: 修改 Daemon 启动逻辑

**Files:**
- Modify: `internal/daemon/daemon.go`
- Modify: `cmd/gomount/cmd_daemon.go`

- [ ] **Step 1: 修改 StartDaemon 函数**

```go
func StartDaemon(configPath string, cfg DaemonConfig) error {
    me, err := os.Executable()
    if err != nil {
        return fmt.Errorf("cannot find executable: %w", err)
    }
    
    port := cfg.GetPort()
    
    args := []string{me}
    if configPath != "" {
        args = append(args, "-c", configPath)
    }
    args = append(args, "daemon", "run")  // 添加 daemon run 子命令
    
    env := append(os.Environ(),
        DaemonEnvKey+"="+DaemonEnvValue,
        "GOMOUNT_DAEMON_PORT="+strconv.Itoa(port),
    )
    
    // 保存原始用户信息（用于降权）
    if currentUser, err := user.Current(); err == nil {
        env = append(env,
            "GOMOUNT_ORIGINAL_UID="+currentUser.Uid,
            "GOMOUNT_ORIGINAL_GID="+currentUser.Gid,
            "GOMOUNT_ORIGINAL_USER="+currentUser.Username,
        )
    }
    
    null, err := os.Open(os.DevNull)
    if err != nil {
        return err
    }
    defer null.Close()
    
    attr := &os.ProcAttr{
        Env:   env,
        Files: []*os.File{null, null, null},
        Sys: &syscall.SysProcAttr{
            Setsid: true,
        },
    }
    
    proc, err := os.StartProcess(me, args, attr)
    if err != nil {
        return fmt.Errorf("failed to start daemon: %w", err)
    }
    proc.Release()
    
    return waitForDaemon(port, 10*time.Second)
}
```

- [ ] **Step 2: 修改 runAsDaemon 函数**

```go
func runAsDaemon() {
    cfg, err := config.LoadConfig(configPath)
    if err != nil {
        fmt.Fprintf(os.Stderr, "daemon: failed to load config: %v\n", err)
        os.Exit(1)
    }
    
    mountMgr := mountkit.NewManager()
    mgr := createDriverManagerWithMountKit(cfg, mountMgr)
    
    daemonCfg := DaemonConfig{}
    if cfg.Daemon != nil {
        daemonCfg.Port = cfg.Daemon.Port
    }
    
    handlers := &Handlers{
        Mount: func(ctx context.Context, name string, meta MetaInfo) (string, error) {
            // 创建目录（如果需要）
            entry, err := mgr.GetMount(name)
            if err != nil {
                return "", err
            }
            
            if err := ensureMountDir(entry.MountDirPath, meta); err != nil {
                return "", err
            }
            
            if err := mgr.Mount(ctx, name); err != nil {
                return "", err
            }
            return "mounted", nil
        },
        // ... 其他 handlers
    }
    
    if err := RunDaemon(handlers, daemonCfg); err != nil {
        fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
        os.Exit(1)
    }
}

func ensureMountDir(path string, meta MetaInfo) error {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        // 创建目录
        if err := os.MkdirAll(path, 0755); err != nil {
            return fmt.Errorf("failed to create mount directory: %w", err)
        }
        
        // 设置所有者
        if err := os.Chown(path, meta.UID, meta.GID); err != nil {
            return fmt.Errorf("failed to set mount directory owner: %w", err)
        }
    }
    return nil
}
```

- [ ] **Step 3: 提交**

```bash
git add internal/daemon/daemon.go cmd/gomount/cmd_daemon.go
git commit -m "refactor: update daemon startup logic for WebSocket"
```

---

## Task 9: 添加 gorilla/websocket 依赖

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: 添加依赖**

```bash
go get github.com/gorilla/websocket
```

- [ ] **Step 2: 提交**

```bash
git add go.mod go.sum
git commit -m "deps: add gorilla/websocket dependency"
```

---

## Task 10: 测试和验证

**Files:**
- 所有修改的文件

- [ ] **Step 1: 编译检查**

```bash
go build ./cmd/gomount
```

- [ ] **Step 2: 运行测试**

```bash
go test ./...
```

- [ ] **Step 3: 手动测试基本流程**

```bash
# 1. 启动 daemon
./gomount daemon run

# 2. 测试挂载
./gomount mount nas

# 3. 测试列表
./gomount list

# 4. 测试卸载
./gomount umount nas

# 5. 停止 daemon
./gomount daemon down
```

- [ ] **Step 4: 提交**

```bash
git commit -m "test: verify WebSocket daemon implementation"
```

---

## 总结

这个实施计划将 gomount 从 HTTP API 架构重构为 WebSocket 通信架构，主要变化：

1. **通信协议**：HTTP → WebSocket
2. **交互方式**：支持交互式密码输入
3. **权限模型**：daemon 以 root 运行，CLI 传递用户信息
4. **命令结构**：子命令调整（d→daemon, c→mkdir）

**关键设计决策：**
- daemon 长期运行，以 root 权限执行特权操作
- CLI 短生命周期，负责用户交互和传递信息
- WebSocket 支持双向通信，daemon 可以主动请求用户输入
- 目录创建和权限设置由 daemon 处理，使用 CLI 传递的元信息

**Plan complete and saved to `docs/superpowers/plans/2026-05-09-websocket-daemon.md`.**

**Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints for review

**Which approach?**