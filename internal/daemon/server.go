package daemon

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/hsldymq/gomount/internal/drivers"
)

type Handlers struct {
	Mount      func(ctx context.Context, name string) (string, error)
	Unmount    func(ctx context.Context, name string) (string, error)
	UnmountAll func()
	List       func() []MountEntryStatus
	Status     func(ctx context.Context, name string) (*drivers.MountStatus, error)
	Shutdown   func()
}

func RunDaemon(handlers *Handlers, cfg DaemonConfig) error {
	port := cfg.GetPort()
	if envPort := os.Getenv("GOMOUNT_DAEMON_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			port = p
		}
	}

	wsServer := NewWebSocketServer(handlers)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", wsServer.HandleWebSocket)

	srv := &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	WriteDaemonInfo(os.Getpid(), port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		handlers.UnmountAll()
		srv.Close()
	}()

	handlers.Shutdown = func() {
		handlers.UnmountAll()
		CleanupDaemonInfo()
		srv.Close()
	}

	err := srv.ListenAndServe()
	CleanupDaemonInfo()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
