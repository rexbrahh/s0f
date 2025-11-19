package main

import (
	"encoding/json"
	"sync"

	"github.com/rexliu/s0f/pkg/ipc"
)

// eventHub broadcasts tree_changed events to connected clients.
type eventHub struct {
	logger  ipc.Logger
	mu      sync.Mutex
	clients map[*eventClient]struct{}
}

type eventClient struct {
	send chan []byte
}

func newEventHub(logger ipc.Logger) *eventHub {
	return &eventHub{
		logger:  logger,
		clients: make(map[*eventClient]struct{}),
	}
}

func (h *eventHub) register() *eventClient {
	h.mu.Lock()
	defer h.mu.Unlock()
	client := &eventClient{send: make(chan []byte, 16)}
	h.clients[client] = struct{}{}
	return client
}

func (h *eventHub) unregister(client *eventClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
}

func (h *eventHub) broadcast(event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		if h.logger != nil {
			h.logger.Printf("event marshal error: %v", err)
		}
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client.send <- payload:
		default:
			if h.logger != nil {
				h.logger.Printf("dropping event for slow client")
			}
		}
	}
}
