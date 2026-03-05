package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eduardmaghakyan/why/internal/hook"
	"github.com/eduardmaghakyan/why/internal/store"
)

const recordWhySchema = `{
	"type": "object",
	"properties": {
		"file_path": {
			"type": "string",
			"description": "The file that is about to be edited (relative or absolute)"
		},
		"reasoning": {
			"type": "string",
			"description": "Why this change is needed - the problem, alternatives considered, and tradeoffs"
		}
	},
	"required": ["file_path", "reasoning"]
}`

type Server struct {
	store   *store.Store
	version string
}

func NewServer(whyRoot, version string) *Server {
	return &Server{
		store:   store.New(whyRoot),
		version: version,
	}
}

func (s *Server) Run() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var req Request
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		// Notifications have no ID and require no response
		if req.ID == nil {
			continue
		}

		var resp Response
		resp.JSONRPC = "2.0"
		resp.ID = req.ID

		switch req.Method {
		case "initialize":
			resp.Result = s.handleInitialize()
		case "tools/list":
			resp.Result = s.handleToolsList()
		case "tools/call":
			result, err := s.handleToolsCall(req.Params)
			if err != nil {
				resp.Error = &RPCError{Code: -32000, Message: err.Error()}
			} else {
				resp.Result = result
			}
		default:
			resp.Error = &RPCError{Code: -32601, Message: "method not found: " + req.Method}
		}

		encoder.Encode(resp)
	}

	return scanner.Err()
}

func (s *Server) handleInitialize() *InitializeResult {
	return &InitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    Capabilities{Tools: &struct{}{}},
		ServerInfo:      ServerInfo{Name: "why-tracker", Version: s.version},
	}
}

func (s *Server) handleToolsList() *ToolsListResult {
	return &ToolsListResult{
		Tools: []Tool{
			{
				Name:        "record_why",
				Description: "Record the reasoning for an upcoming file edit. Call this BEFORE every Edit, Write, or MultiEdit to capture why the change is being made.",
				InputSchema: json.RawMessage(recordWhySchema),
			},
		},
	}
}

func (s *Server) handleToolsCall(params json.RawMessage) (*ToolCallResult, error) {
	var call ToolCallParams
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if call.Name != "record_why" {
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}

	var args struct {
		FilePath  string `json:"file_path"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal(call.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Create object
	obj := &store.Object{
		Timestamp: time.Now().Format("2006-01-02 15:04"),
		Commit:    gitCommit(),
		Reasoning: args.Reasoning,
	}

	hash, err := s.store.Put(obj)
	if err != nil {
		return nil, fmt.Errorf("store object: %w", err)
	}

	// Write pending hash for the hook to consume
	absPath, _ := filepath.Abs(args.FilePath)
	key := hook.FileKey(absPath)
	if err := hook.WritePending(key, hash); err != nil {
		return nil, fmt.Errorf("write pending: %w", err)
	}

	return &ToolCallResult{
		Content: []ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Reasoning recorded for %s. Proceed with your edit.", args.FilePath)},
		},
	}, nil
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "no-git"
	}
	return strings.TrimSpace(string(out))
}
