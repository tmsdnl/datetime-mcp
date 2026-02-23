package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/tmsdnl/datetime-mcp/internal/datetime"
	"github.com/tmsdnl/datetime-mcp/internal/formats"
)

// Config holds MCP server configuration.
type Config struct {
	Timezone   string // default IANA tz; empty = TZ env or local
	FormatsDir string // override XDG formats directory
	Log        bool
}

// Run starts the MCP server. It reads JSON-RPC messages from stdin and writes
// responses to stdout. Blocks until stdin closes or SIGTERM/SIGINT (F-028).
func Run(cfg Config) error {
	logger := makeLogger(cfg.Log)

	dir := cfg.FormatsDir
	if dir == "" {
		dir = defaultFormatsDir()
	}

	fmts, errs := formats.Load(dir)
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "warning: %v\n", e)
	}
	reg := formats.NewRegistry(fmts)

	tzEnv := os.Getenv("TZ")
	defaultLoc, err := datetime.ResolveTimezone(cfg.Timezone, tzEnv)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %v; using UTC\n", err)
		defaultLoc = time.UTC
	}
	logger("MCP server starting, timezone: %s", defaultLoc.String())

	srv := &server{
		reg:        reg,
		defaultLoc: defaultLoc,
		logger:     logger,
		in:         os.Stdin,
		out:        os.Stdout,
	}
	return srv.serve()
}

type server struct {
	reg         *formats.Registry
	defaultLoc  *time.Location
	logger      func(string, ...any)
	in          io.Reader
	out         io.Writer
	initialized bool
}

func (s *server) serve() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle SIGTERM/SIGINT for graceful shutdown (F-028, NF-042).
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-sigCh:
			s.logger("received shutdown signal")
			cancel()
		case <-ctx.Done():
		}
	}()

	scanner := bufio.NewScanner(s.in)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}
			s.logger("recv: %s", line)
			resp := s.handleMessage(line)
			if resp != nil {
				data, err := json.Marshal(resp)
				if err != nil {
					s.logger("marshal error: %v", err)
					continue
				}
				s.logger("send: %s", data)
				fmt.Fprintf(s.out, "%s\n", data)
			}
		}
	}()

	select {
	case <-doneCh: // stdin closed (EOF)
	case <-ctx.Done(): // signal received
	}

	signal.Stop(sigCh)
	s.logger("MCP server stopped")
	return nil
}

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func errorResponse(id json.RawMessage, code int, message string) *jsonrpcResponse {
	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonrpcError{Code: code, Message: message},
	}
}

func resultResponse(id json.RawMessage, result any) *jsonrpcResponse {
	return &jsonrpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}

func (s *server) handleMessage(data []byte) *jsonrpcResponse {
	var req jsonrpcRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return errorResponse(nil, -32700, "parse error: "+err.Error())
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		s.initialized = true
		return nil // notification, no response
	case "ping":
		return resultResponse(req.ID, map[string]any{})
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return errorResponse(req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *server) handleInitialize(req jsonrpcRequest) *jsonrpcResponse {
	result := map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities": map[string]any{
			"tools": map[string]any{
				"listChanged": false,
			},
		},
		"serverInfo": map[string]any{
			"name":        "datetime-mcp",
			"version":     "0.0.0", // replaced at build time
			"description": "Self-contained date/time provider for Claude Desktop and Claude Code",
		},
	}
	return resultResponse(req.ID, result)
}

func defaultFormatsDir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "datetime-mcp", "formats")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "datetime-mcp", "formats")
}

func makeLogger(enabled bool) func(string, ...any) {
	if !enabled {
		return func(string, ...any) {}
	}
	return func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, "[datetime-mcp] "+format+"\n", args...)
	}
}
