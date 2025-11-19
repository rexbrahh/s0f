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

	"github.com/rexliu/s0f/pkg/config"
	"github.com/rexliu/s0f/pkg/core"
	"github.com/rexliu/s0f/pkg/ipc"
	"github.com/rexliu/s0f/pkg/logging"
	"github.com/rexliu/s0f/pkg/storage/sqlite"
	gitvcs "github.com/rexliu/s0f/pkg/vcs/git"
)

func main() {
	profile := flag.String("profile", "./_dev_profile", "Path to profile directory")
	configPath := flag.String("config", "", "Path to config.toml (default <profile>/config.toml)")
	socket := flag.String("socket", "", "Override IPC socket path (optional)")
	flag.Parse()

	cfgPath := *configPath
	profileDir := *profile
	if cfgPath != "" {
		profileDir = filepath.Dir(cfgPath)
	} else {
		cfgPath = filepath.Join(profileDir, "config.toml")
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger := logging.New("bmd")
	logger.Printf("starting daemon profile=%s dir=%s", cfg.ProfileName, profileDir)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, profileDir, cfg, *socket, logger); err != nil {
		logger.Printf("fatal error: %v", err)
		os.Exit(1)
	}
}

type daemon struct {
	store      *sqlite.Store
	logger     *logging.Logger
	repo       *gitvcs.Repo
	profileDir string
	cfg        *config.ProfileConfig
	eventHub   *eventHub
}

func run(ctx context.Context, profileDir string, cfg *config.ProfileConfig, socketOverride string, logger *logging.Logger) error {
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return err
	}
	dbPath := config.ResolvePath(profileDir, cfg.Storage.DBPath)
	store, err := sqlite.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open sqlite: %w", err)
	}
	defer store.Close()
	if err := store.Init(ctx); err != nil {
		return fmt.Errorf("init sqlite: %w", err)
	}

	socketPath := socketOverride
	if socketPath == "" {
		socketPath = config.ResolvePath(profileDir, cfg.IPC.SocketPath)
	}
	if err := cleanupSocket(socketPath); err != nil {
		return err
	}

	srv := ipc.NewServer(logger)
	vcRepo, err := gitvcs.Init(profileDir)
	if err != nil {
		logger.Printf("warning: failed to init git repo: %v", err)
	}

	eh := newEventHub(logger)
	d := &daemon{store: store, logger: logger, repo: vcRepo, profileDir: profileDir, cfg: cfg, eventHub: eh}
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

func (d *daemon) broadcastTreeChanged(tree core.Tree) {
	if d.eventHub == nil {
		return
	}
	event := map[string]any{
		"kind":           "event",
		"event":          "tree_changed",
		"version":        tree.Version,
		"changedNodeIds": []string{},
		"generatedAt":    time.Now().UnixMilli(),
	}
	d.eventHub.broadcast(event)
}
