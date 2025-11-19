package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rexliu/s0f/pkg/config"
	"github.com/rexliu/s0f/pkg/core"
	"github.com/rexliu/s0f/pkg/ipc"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	switch os.Args[1] {
	case "init":
		initProfile()
	case "version":
		fmt.Println("s0f CLI scaffolding")
	case "ping":
		if err := pingCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "ping error: %v\n", err)
			os.Exit(1)
		}
	case "tree":
		if err := treeCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "tree error: %v\n", err)
			os.Exit(1)
		}
	case "apply":
		if err := applyCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "apply error: %v\n", err)
			os.Exit(1)
		}
	case "search":
		if err := searchCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "search error: %v\n", err)
			os.Exit(1)
		}
	case "watch":
		if err := watchCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
			os.Exit(1)
		}
	case "snapshot":
		if err := snapshotCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "snapshot error: %v\n", err)
			os.Exit(1)
		}
	case "diag":
		if err := diagCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "diag error: %v\n", err)
			os.Exit(1)
		}
	case "remote":
		if err := remoteCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "remote error: %v\n", err)
			os.Exit(1)
		}
	case "vcs":
		if err := vcsCommand(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "vcs error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand %q\n", os.Args[1])
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println("Usage: s0f <command> [options]")
	fmt.Println("Commands:")
	fmt.Println("  init      Initialize a local profile (writes config.toml)")
	fmt.Println("  ping      Call the daemon ping endpoint via IPC")
	fmt.Println("  tree      Fetch the current bookmark tree from the daemon")
	fmt.Println("  apply     Send apply_ops payload (JSON) to the daemon")
	fmt.Println("  search    Run substring search over title/url")
	fmt.Println("  watch     Stream tree_changed events from the daemon")
	fmt.Println("  snapshot  Fetch snapshot payload via IPC")
	fmt.Println("  diag      Print profile configuration paths")
	fmt.Println("  remote    Manage Git remote configuration (set/show)")
	fmt.Println("  vcs push|pull    Trigger VCS push or pull via the daemon")
	fmt.Println("  version   Print CLI version")
}

func initProfile() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	profilePath := fs.String("profile", "./_dev_profile", "Profile directory")
	name := fs.String("name", "dev", "Profile name")
	force := fs.Bool("force", false, "Overwrite existing config if present")
	_ = fs.Parse(os.Args[2:])
	if err := os.MkdirAll(*profilePath, 0o700); err != nil {
		fmt.Fprintf(os.Stderr, "init error: %v\n", err)
		os.Exit(1)
	}
	configPath := filepath.Join(*profilePath, "config.toml")
	if _, err := os.Stat(configPath); err == nil && !*force {
		fmt.Fprintf(os.Stderr, "config already exists at %s (use --force to overwrite)\n", configPath)
		os.Exit(1)
	}
	cfg := config.DefaultProfile(*name)
	if err := config.Save(configPath, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "init error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("initialized profile %s at %s\n", cfg.ProfileName, *profilePath)
}

func pingCommand(args []string) error {
	fs := flag.NewFlagSet("ping", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	socket := fs.String("socket", "", "Override socket path")
	_ = fs.Parse(args)

	resp, err := rpcCall(*profile, *socket, "ping", nil)
	if err != nil {
		return err
	}
	var data struct {
		Now int64 `json:"now"`
	}
	if err := json.Unmarshal(resp.Result, &data); err != nil {
		return fmt.Errorf("decode result: %w", err)
	}
	fmt.Printf("daemon responded: now=%d\n", data.Now)
	return nil
}

func treeCommand(args []string) error {
	fs := flag.NewFlagSet("tree", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	socket := fs.String("socket", "", "Override socket path")
	_ = fs.Parse(args)

	resp, err := rpcCall(*profile, *socket, "get_tree", json.RawMessage(`{}`))
	if err != nil {
		return err
	}
	var payload struct {
		Tree core.Tree `json:"tree"`
	}
	if err := json.Unmarshal(resp.Result, &payload); err != nil {
		return fmt.Errorf("decode tree: %w", err)
	}
	out, err := json.MarshalIndent(payload.Tree, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func applyCommand(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	socket := fs.String("socket", "", "Override socket path")
	filePath := fs.String("file", "", "Path to JSON payload for apply_ops (defaults to stdin)")
	inline := fs.String("ops", "", "Inline JSON payload for apply_ops")
	_ = fs.Parse(args)

	var payload []byte
	var err error
	switch {
	case *filePath != "":
		payload, err = os.ReadFile(*filePath)
	case *inline != "":
		payload = []byte(*inline)
	default:
		payload, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return err
	}
	payload = []byte(strings.TrimSpace(string(payload)))
	if len(payload) == 0 {
		return fmt.Errorf("empty apply_ops payload")
	}

	resp, err := rpcCall(*profile, *socket, "apply_ops", json.RawMessage(payload))
	if err != nil {
		return err
	}
	var data struct {
		Tree      core.Tree      `json:"tree"`
		VCSStatus map[string]any `json:"vcsStatus"`
	}
	if err := json.Unmarshal(resp.Result, &data); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func searchCommand(args []string) error {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	socket := fs.String("socket", "", "Override socket path")
	query := fs.String("query", "", "Search query (substring)")
	limit := fs.Int("limit", 50, "Maximum results (1-500)")
	_ = fs.Parse(args)

	payload := map[string]any{
		"query": *query,
		"limit": *limit,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	resp, err := rpcCall(*profile, *socket, "search", raw)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(resp.Result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func watchCommand(args []string) error {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	socket := fs.String("socket", "", "Override socket path")
	_ = fs.Parse(args)

	socketPath, err := resolveSocketPath(*profile, *socket)
	if err != nil {
		return err
	}
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("dial %s: %w", socketPath, err)
	}
	defer conn.Close()

	req := ipc.Request{
		ID:   fmt.Sprintf("cli-watch-%d", time.Now().UnixNano()),
		Type: "subscribe_events",
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if err := ipc.WriteFrame(conn, payload); err != nil {
		return err
	}
	fmt.Println("Subscribed to tree_changed events (Ctrl+C to exit)")
	for {
		frame, err := ipc.ReadFrame(conn)
		if err != nil {
			return err
		}
		fmt.Println(string(frame))
	}
}

func snapshotCommand(args []string) error {
	fs := flag.NewFlagSet("snapshot", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	socket := fs.String("socket", "", "Override socket path")
	_ = fs.Parse(args)
	resp, err := rpcCall(*profile, *socket, "get_snapshot", json.RawMessage(`{}`))
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(resp.Result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func diagCommand(args []string) error {
	fs := flag.NewFlagSet("diag", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	_ = fs.Parse(args)
	cfg, err := config.LoadProfile(*profile)
	if err != nil {
		return err
	}
	fmt.Printf("Profile: %s\n", cfg.ProfileName)
	fmt.Printf("Config: %s\n", filepath.Join(*profile, "config.toml"))
	fmt.Printf("DB Path: %s\n", config.ResolvePath(*profile, cfg.Storage.DBPath))
	fmt.Printf("Socket: %s\n", config.ResolvePath(*profile, cfg.IPC.SocketPath))
	if cfg.Logging.FilePath != "" {
		fmt.Printf("Log File: %s\n", config.ResolvePath(*profile, cfg.Logging.FilePath))
	}
	fmt.Printf("VCS Branch: %s (enabled=%t)\n", cfg.VCS.Branch, cfg.VCS.Enabled)
	if cfg.VCS.Remote.URL != "" {
		fmt.Printf("Remote URL: %s\n", cfg.VCS.Remote.URL)
	}
	return nil
}

func remoteCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: s0f remote <set|show> [options]")
	}
	sub := args[0]
	switch sub {
	case "set":
		fs := flag.NewFlagSet("remote set", flag.ExitOnError)
		profile := fs.String("profile", "./_dev_profile", "Profile directory")
		url := fs.String("url", "", "Remote Git URL")
		cred := fs.String("credential", "", "Credential reference (optional)")
		_ = fs.Parse(args[1:])
		if *url == "" {
			return fmt.Errorf("--url is required")
		}
		cfg, err := config.LoadProfile(*profile)
		if err != nil {
			return err
		}
		cfg.VCS.Remote.URL = *url
		cfg.VCS.Remote.CredentialRef = *cred
		cfg.VCS.Enabled = true
		if err := config.Save(filepath.Join(*profile, "config.toml"), cfg); err != nil {
			return err
		}
		fmt.Printf("remote set to %s\n", *url)
		return nil
	case "show":
		fs := flag.NewFlagSet("remote show", flag.ExitOnError)
		profile := fs.String("profile", "./_dev_profile", "Profile directory")
		_ = fs.Parse(args[1:])
		cfg, err := config.LoadProfile(*profile)
		if err != nil {
			return err
		}
		if cfg.VCS.Remote.URL == "" {
			fmt.Println("remote not configured")
		} else {
			fmt.Printf("remote URL: %s\n", cfg.VCS.Remote.URL)
			if cfg.VCS.Remote.CredentialRef != "" {
				fmt.Printf("credential ref: %s\n", cfg.VCS.Remote.CredentialRef)
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown remote subcommand %q", sub)
	}
}

func vcsCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: s0f vcs <push|pull|status> [options]")
	}
	sub := args[0]
	fs := flag.NewFlagSet("vcs", flag.ExitOnError)
	profile := fs.String("profile", "./_dev_profile", "Profile directory")
	socket := fs.String("socket", "", "Override socket path")
	_ = fs.Parse(args[1:])

	var method string
	switch sub {
	case "push":
		method = "vcs_push"
	case "pull":
		method = "vcs_pull"
	case "status":
		method = "vcs_status"
	default:
		return fmt.Errorf("unknown vcs subcommand %q", sub)
	}

	resp, err := rpcCall(*profile, *socket, method, json.RawMessage(`{}`))
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(resp.Result, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func rpcCall(profile, socketOverride, method string, params json.RawMessage) (*ipc.Response, error) {
	socketPath, err := resolveSocketPath(profile, socketOverride)
	if err != nil {
		return nil, err
	}
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", socketPath, err)
	}
	defer conn.Close()

	req := ipc.Request{
		ID:     fmt.Sprintf("cli-%d", time.Now().UnixNano()),
		Type:   method,
		Params: params,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := ipc.WriteFrame(conn, payload); err != nil {
		return nil, err
	}
	respBytes, err := ipc.ReadFrame(conn)
	if err != nil {
		return nil, err
	}
	var resp ipc.Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("daemon error: %s (%s)", resp.Error.Message, resp.Error.Code)
	}
	return &resp, nil
}

func resolveSocketPath(profile, override string) (string, error) {
	if override != "" {
		return override, nil
	}
	cfg, err := config.LoadProfile(profile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("config not found in %s (run 's0f init --profile %s')", profile, profile)
		}
		return "", fmt.Errorf("load config: %w", err)
	}
	return config.ResolvePath(profile, cfg.IPC.SocketPath), nil
}
