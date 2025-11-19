package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	fmt.Println("  init      Initialize a local profile (placeholder)")
	fmt.Println("  ping      Call the daemon ping endpoint via IPC")
	fmt.Println("  tree      Fetch the current bookmark tree from the daemon")
	fmt.Println("  apply     Send apply_ops payload (JSON) to the daemon")
	fmt.Println("  search    Run substring search over title/url")
	fmt.Println("  watch     Stream tree_changed events from the daemon")
	fmt.Println("  vcs push|pull    Trigger VCS push or pull via the daemon")
	fmt.Println("  version   Print CLI version")
}

func initProfile() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	profilePath := fs.String("profile", "./_dev_profile", "Profile directory")
	_ = fs.Parse(os.Args[2:])
	fmt.Printf("[scaffold] would initialize profile at %s\n", *profilePath)
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

	socketPath := *socket
	if socketPath == "" {
		socketPath = filepath.Join(*profile, "ipc.sock")
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

func vcsCommand(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: s0f vcs <push|pull> [options]")
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
	socketPath := socketOverride
	if socketPath == "" {
		socketPath = filepath.Join(profile, "ipc.sock")
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
