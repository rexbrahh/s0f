package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

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
		ID:   fmt.Sprintf("cli-%d", time.Now().UnixNano()),
		Type: "ping",
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if err := ipc.WriteFrame(conn, payload); err != nil {
		return err
	}
	respBytes, err := ipc.ReadFrame(conn)
	if err != nil {
		return err
	}
	var resp ipc.Response
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("daemon error: %s (%s)", resp.Error.Message, resp.Error.Code)
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
