package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rexliu/s0f/pkg/ipc"
	"github.com/rexliu/s0f/pkg/logging"
	"github.com/rexliu/s0f/pkg/storage/sqlite"
)

func main() {
	profile := flag.String("profile", "./_dev_profile", "Path to profile directory")
	socket := flag.String("socket", "", "Override IPC socket path (optional)")
	flag.Parse()

	logger := logging.New("bmd")
	logger.Printf("starting daemon with profile %s", *profile)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, *profile, *socket, logger); err != nil {
		logger.Printf("fatal error: %v", err)
		os.Exit(1)
	}
}

type daemon struct {
	store  *sqlite.Store
	logger *logging.Logger
}

func run(ctx context.Context, profileDir, socketOverride string, logger *logging.Logger) error {
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}
	dbPath := filepath.Join(profileDir, "state.db")
	store, err := sqlite.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer store.Close()
	if err := store.Init(ctx); err != nil {
		return fmt.Errorf("init sqlite: %w", err)
	}
	if _, err := store.LoadTree(ctx); err != nil {
		logger.Printf("warning: load tree failed: %v", err)
	}

	socketPath := socketOverride
	if socketPath == "" {
		socketPath = filepath.Join(profileDir, "ipc.sock")
	}
	if err := cleanupSocket(socketPath); err != nil {
		return err
	}

	srv := ipc.NewServer(logger)
	d := &daemon{store: store, logger: logger}
	d.registerHandlers(srv)

	if err := srv.Start(ctx, socketPath); err != nil {
		return fmt.Errorf("start ipc: %w", err)
	}
	defer func() {
		srv.Stop()
		cleanupSocket(socketPath)
	}()

	logger.Printf("daemon ready; socket at %s", socketPath)

	<-ctx.Done()
	logger.Println("shutting down")
	return nil
}

func cleanupSocket(path string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return err
		}
	}
	return nil
}

func pingHandler(logger *logging.Logger) ipc.HandlerFunc {
	return func(ctx context.Context, params json.RawMessage) (any, *ipc.Error) {
		_ = ctx
		_ = params
		now := time.Now().UnixMilli()
		if logger != nil {
			logger.Printf("received ping at %d", now)
		}
		return map[string]any{"now": now}, nil
	}
}
