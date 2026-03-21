package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

const whyBlameSchema = `{
	"type": "object",
	"properties": {
		"file_path": {
			"type": "string",
			"description": "The file to show line-by-line reasoning for (relative path)"
		}
	},
	"required": ["file_path"]
}`

const whyHistorySchema = `{
	"type": "object",
	"properties": {
		"file_path": {
			"type": "string",
			"description": "The file to show edit history for (relative path)"
		},
		"related": {
			"type": "boolean",
			"description": "If true, also show files that were changed together"
		}
	},
	"required": ["file_path"]
}`

type Server struct {
	store   *store.Store
	refs    *store.Refs
	version string
}

func NewServer(whyRoot, version string) *Server {
	return &Server{
		store:   store.New(whyRoot),
		refs:    store.NewRefs(whyRoot),
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
			{
				Name:        "why_blame",
				Description: "Show line-by-line reasoning for a file. Each line is annotated with the commit and reasoning summary explaining why it was changed.",
				InputSchema: json.RawMessage(whyBlameSchema),
			},
			{
				Name:        "why_history",
				Description: "Show the edit history with full reasoning for a file, sorted chronologically. Optionally shows related files that were changed together.",
				InputSchema: json.RawMessage(whyHistorySchema),
			},
		},
	}
}

func (s *Server) handleToolsCall(params json.RawMessage) (*ToolCallResult, error) {
	var call ToolCallParams
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	switch call.Name {
	case "record_why":
		return s.handleRecordWhy(call.Arguments)
	case "why_blame":
		return s.handleWhyBlame(call.Arguments)
	case "why_history":
		return s.handleWhyHistory(call.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func (s *Server) handleRecordWhy(arguments json.RawMessage) (*ToolCallResult, error) {
	var args struct {
		FilePath  string `json:"file_path"`
		Reasoning string `json:"reasoning"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	obj := &store.Object{
		Timestamp: time.Now().Format("2006-01-02 15:04"),
		Commit:    gitCommit(),
		TurnID:    hook.ReadTurnID(),
		Reasoning: args.Reasoning,
	}

	hash, err := s.store.Put(obj)
	if err != nil {
		return nil, fmt.Errorf("store object: %w", err)
	}

	if err := hook.WritePending(hash); err != nil {
		return nil, fmt.Errorf("write pending: %w", err)
	}

	return &ToolCallResult{
		Content: []ContentBlock{
			{Type: "text", Text: fmt.Sprintf("Reasoning recorded for %s. Proceed with your edit.", args.FilePath)},
		},
	}, nil
}

func (s *Server) handleWhyBlame(arguments json.RawMessage) (*ToolCallResult, error) {
	var args struct {
		FilePath string `json:"file_path"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	filePath := mcpRelPath(args.FilePath)

	sourceBytes, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read source: %w", err)
	}
	sourceLines := strings.Split(strings.TrimSuffix(string(sourceBytes), "\n"), "\n")

	hashes, _ := s.refs.Read(filePath)

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", filePath)

	cache := map[string]*store.Object{}
	prevHash := ""

	for i, line := range sourceLines {
		hash := ""
		if i < len(hashes) {
			hash = hashes[i]
		}

		if hash != prevHash && hash != "" {
			obj, ok := cache[hash]
			if !ok {
				obj, err = s.store.Get(hash)
				if err != nil {
					fmt.Fprintf(&b, "%4d │ %s\n", i+1, line)
					prevHash = hash
					continue
				}
				cache[hash] = obj
			}
			summary := mcpTruncate(obj.Reasoning, 70)
			fmt.Fprintf(&b, "── %s: %s ──\n", obj.Commit, summary)
		}

		fmt.Fprintf(&b, "%4d │ %s\n", i+1, line)
		prevHash = hash
	}

	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: b.String()}},
	}, nil
}

func (s *Server) handleWhyHistory(arguments json.RawMessage) (*ToolCallResult, error) {
	var args struct {
		FilePath string `json:"file_path"`
		Related  bool   `json:"related"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	filePath := mcpRelPath(args.FilePath)

	hashes, _ := s.refs.Read(filePath)

	seen := map[string]bool{}
	var unique []string
	for _, h := range hashes {
		if h != "" && !seen[h] {
			seen[h] = true
			unique = append(unique, h)
		}
	}

	type entry struct {
		hash string
		obj  *store.Object
	}
	var entries []entry
	for _, h := range unique {
		obj, err := s.store.Get(h)
		if err != nil {
			continue
		}
		entries = append(entries, entry{h, obj})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].obj.Timestamp < entries[j].obj.Timestamp
	})

	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "## %s | %s\n\n%s\n", e.obj.Timestamp, e.obj.Commit, e.obj.Reasoning)

		if args.Related {
			related := s.refs.FindRelated(filePath, s.store)
			if len(related) > 0 {
				fmt.Fprintf(&b, "\n  Also changed:\n")
				for _, r := range related {
					fmt.Fprintf(&b, "    %s\n", r)
				}
			}
		}

		fmt.Fprintf(&b, "\n---\n\n")
	}

	if b.Len() == 0 {
		b.WriteString("No reasoning history found for " + filePath)
	}

	return &ToolCallResult{
		Content: []ContentBlock{{Type: "text", Text: b.String()}},
	}, nil
}

func gitCommit() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "no-git"
	}
	return strings.TrimSpace(string(out))
}

func mcpRelPath(path string) string {
	if !filepath.IsAbs(path) {
		return path
	}
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}
	return rel
}

func mcpTruncate(s string, max int) string {
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		s = s[:idx]
	}
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
