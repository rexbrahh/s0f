package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

// message represents the Chrome Native Messaging envelope.
type message struct {
    Type string          `json:"type"`
    Data json.RawMessage `json:"data,omitempty"`
}

func main() {
    reader := bufio.NewReader(os.Stdin)
    writer := bufio.NewWriter(os.Stdout)
    defer writer.Flush()

    // Placeholder echo loop: read JSON per line and echo back.
    for {
        line, err := reader.ReadBytes('\n')
        if err != nil {
            fmt.Fprintf(os.Stderr, "bridge exiting: %v\n", err)
            return
        }
        var msg message
        if err := json.Unmarshal(line, &msg); err != nil {
            fmt.Fprintf(os.Stderr, "invalid message: %v\n", err)
            continue
        }
        if err := json.NewEncoder(writer).Encode(msg); err != nil {
            fmt.Fprintf(os.Stderr, "write error: %v\n", err)
            return
        }
        writer.Flush()
    }
}
