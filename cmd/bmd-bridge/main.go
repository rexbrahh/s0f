package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

// nativeRequest mirrors Chrome Native Messaging messages.
type nativeRequest struct {
	ID     string          `json:"id,omitempty"`
	Type   string          `json:"type"`
	Params json.RawMessage `json:"params,omitempty"`
}

func main() {
	profile := os.Getenv("S0F_PROFILE")
	if profile == "" {
		profile = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "BookmarkRuntime", "default")
	}
	socketPath := filepath.Join(profile, "ipc.sock")
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bridge dial error: %v\n", err)
		return
	}
	defer conn.Close()

	stdin := bufio.NewReader(os.Stdin)
	stdout := bufio.NewWriter(os.Stdout)
	defer stdout.Flush()

	for {
		msg, err := readNativeMessage(stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bridge read error: %v\n", err)
			return
		}
		payload, err := json.Marshal(msg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
			continue
		}
		if err := writeFrame(conn, payload); err != nil {
			fmt.Fprintf(os.Stderr, "ipc write error: %v\n", err)
			return
		}
		if err := forwardResponses(conn, stdout); err != nil {
			fmt.Fprintf(os.Stderr, "forward error: %v\n", err)
			return
		}
	}
}

func forwardResponses(conn net.Conn, out *bufio.Writer) error {
	for {
		frame, err := readFrame(conn)
		if err != nil {
			return err
		}
		if err := writeNativeMessage(out, frame); err != nil {
			return err
		}
		out.Flush()
		// For non-streaming RPCs we expect a single response containing ok/error fields.
		var resp map[string]any
		if err := json.Unmarshal(frame, &resp); err == nil {
			if ok, exists := resp["ok"].(bool); exists {
				// Standard request/response -> stop forwarding for this cycle.
				_ = ok
				return nil
			}
		}
	}
}

func readNativeMessage(r *bufio.Reader) (*nativeRequest, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	var req nativeRequest
	if err := json.Unmarshal(buf, &req); err != nil {
		return nil, err
	}
	return &req, nil
}

func writeNativeMessage(w io.Writer, payload []byte) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(payload))); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}

func readFrame(r io.Reader) ([]byte, error) {
	var length uint32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func writeFrame(w io.Writer, payload []byte) error {
	if err := binary.Write(w, binary.LittleEndian, uint32(len(payload))); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
