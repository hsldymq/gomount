package daemon

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
)

func Run(ctx context.Context, socketPath string, server *Server) error {
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		if client := NewClient(socketPath); client != nil {
			if _, healthErr := client.Health(ctx); healthErr == nil {
				return fmt.Errorf("daemon is already running at %s", socketPath)
			}
		}
		_ = os.Remove(socketPath)
		ln, err = net.Listen("unix", socketPath)
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(socketPath, 0600); err != nil {
		_ = ln.Close()
		return err
	}
	defer os.Remove(socketPath)

	httpServer := &http.Server{Handler: server}
	go func() {
		<-ctx.Done()
		_ = httpServer.Close()
	}()
	err = httpServer.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
