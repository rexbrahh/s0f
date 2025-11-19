package ipc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// HandlerFunc processes RPC params and returns a result or structured error.
type HandlerFunc func(context.Context, json.RawMessage) (any, *Error)

// Logger is satisfied by logging.Logger; kept minimal to avoid dependency cycles.
type Logger interface {
	Printf(format string, v ...any)
}

// Server listens for IPC requests over Unix sockets.
type Server struct {
	ln       net.Listener
	mu       sync.RWMutex
	handlers map[string]HandlerFunc
	closed   bool
	logger   Logger
}

// NewServer constructs an IPC server.
func NewServer(logger Logger) *Server {
	return &Server{
		handlers: make(map[string]HandlerFunc),
		logger:   logger,
	}
}

// Register installs a handler for a method.
func (s *Server) Register(method string, handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[method] = handler
}

// Start begins accepting connections on endpoint.
func (s *Server) Start(ctx context.Context, endpoint string) error {
	if s == nil {
		return errors.New("nil server")
	}
	ln, err := net.Listen("unix", endpoint)
	if err != nil {
		return err
	}
	s.ln = ln
	go s.acceptLoop(ctx)
	return nil
}

func (s *Server) acceptLoop(ctx context.Context) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			if ctx.Err() != nil || s.isClosed() {
				return
			}
			s.logf("accept error: %v", err)
			continue
		}
		go s.handleConn(ctx, conn)
	}
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	for {
		payload, err := readFrame(conn)
		if err != nil {
			return
		}
		var req Request
		if err := json.Unmarshal(payload, &req); err != nil {
			s.writeError(conn, req.ID, "INVALID_REQUEST", "invalid json", nil)
			continue
		}
		traceID := fmt.Sprintf("ipc-%d", time.Now().UnixNano())
		handler := s.lookupHandler(req.Type)
		if handler == nil {
			s.writeError(conn, req.ID, "INVALID_REQUEST", "unknown method", map[string]any{"method": req.Type, "traceId": traceID})
			continue
		}
		result, rpcErr := handler(ctx, req.Params)
		resp := Response{ID: req.ID, TraceID: traceID}
		if rpcErr != nil {
			resp.Error = rpcErr
		} else {
			raw, err := json.Marshal(result)
			if err != nil {
				s.writeError(conn, req.ID, "INTERNAL", err.Error(), map[string]any{"traceId": traceID})
				continue
			}
			resp.OK = true
			resp.Result = raw
		}
		if err := s.writeResponse(conn, resp); err != nil {
			return
		}
	}
}

func (s *Server) lookupHandler(method string) HandlerFunc {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.handlers[method]
}

func (s *Server) writeResponse(conn net.Conn, resp Response) error {
	payload, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return writeFrame(conn, payload)
}

func (s *Server) writeError(conn net.Conn, id, code, msg string, details map[string]any) {
	resp := Response{ID: id, TraceID: fmt.Sprintf("ipc-%d", time.Now().UnixNano())}
	resp.Error = &Error{Code: code, Message: msg, Details: details}
	_ = s.writeResponse(conn, resp)
}

// Stop shuts down the listener.
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	if s.ln != nil {
		return s.ln.Close()
	}
	return nil
}

func (s *Server) isClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

func (s *Server) logf(format string, v ...any) {
	if s.logger != nil {
		s.logger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

// Errorf helps build protocol errors.
func Errorf(code, message string, details map[string]any) *Error {
	return &Error{Code: code, Message: message, Details: details}
}
