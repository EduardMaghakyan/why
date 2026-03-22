package symbols

import (
	"path/filepath"
	"sort"
	"strings"

	ts "github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// Symbol represents a named code construct (function, class, method).
type Symbol struct {
	Name      string // e.g. "Server.Start" or "processPayment"
	Kind      string // "function", "method", "class"
	StartLine int    // 1-indexed
	EndLine   int    // 1-indexed, inclusive
}

type langConfig struct {
	loader func() *ts.Language
	query  string
}

// Supported languages: only these grammars are loaded (keeps binary smaller).
var languages = map[string]langConfig{
	".go": {
		loader: grammars.GoLanguage,
		query: `
			(function_declaration name: (identifier) @name) @func
			(method_declaration name: (field_identifier) @name) @method
		`,
	},
	".py": {
		loader: grammars.PythonLanguage,
		query: `
			(function_definition name: (identifier) @name) @func
			(class_definition name: (identifier) @name) @class
		`,
	},
	".js": {
		loader: grammars.JavascriptLanguage,
		query: `
			(function_declaration name: (identifier) @name) @func
			(class_declaration name: (identifier) @name) @class
			(method_definition name: (property_identifier) @name) @method
		`,
	},
	".jsx": {
		loader: grammars.JavascriptLanguage,
		query: `
			(function_declaration name: (identifier) @name) @func
			(class_declaration name: (identifier) @name) @class
			(method_definition name: (property_identifier) @name) @method
		`,
	},
	".ts": {
		loader: grammars.TypescriptLanguage,
		query: `
			(function_declaration name: (identifier) @name) @func
			(class_declaration name: (type_identifier) @name) @class
			(method_definition name: (property_identifier) @name) @method
		`,
	},
	".tsx": {
		loader: grammars.TypescriptLanguage,
		query: `
			(function_declaration name: (identifier) @name) @func
			(class_declaration name: (type_identifier) @name) @class
			(method_definition name: (property_identifier) @name) @method
		`,
	},
	".rs": {
		loader: grammars.RustLanguage,
		query: `
			(function_item name: (identifier) @name) @func
			(impl_item type: (type_identifier) @name) @class
		`,
	},
	".java": {
		loader: grammars.JavaLanguage,
		query: `
			(method_declaration name: (identifier) @name) @method
			(class_declaration name: (identifier) @name) @class
		`,
	},
	".rb": {
		loader: grammars.RubyLanguage,
		query: `
			(method name: (identifier) @name) @method
			(class name: (constant) @name) @class
		`,
	},
	".c": {
		loader: grammars.CLanguage,
		query: `
			(function_definition declarator: (function_declarator declarator: (identifier) @name)) @func
		`,
	},
	".cpp": {
		loader: grammars.CppLanguage,
		query: `
			(function_definition declarator: (function_declarator declarator: (identifier) @name)) @func
			(class_specifier name: (type_identifier) @name) @class
		`,
	},
	".h": {
		loader: grammars.CLanguage,
		query: `
			(function_definition declarator: (function_declarator declarator: (identifier) @name)) @func
		`,
	},
}

// Extract parses a file and returns all top-level symbols with line ranges.
func Extract(filePath string, content []byte) []Symbol {
	ext := strings.ToLower(filepath.Ext(filePath))
	cfg, ok := languages[ext]
	if !ok {
		return nil
	}

	lang := cfg.loader()
	if lang == nil {
		return nil
	}

	parser := ts.NewParser(lang)
	tree, err := parser.Parse(content)
	if err != nil || tree == nil {
		return nil
	}

	q, err := ts.NewQuery(cfg.query, lang)
	if err != nil {
		return nil
	}

	cursor := q.Exec(tree.RootNode(), lang, content)

	var syms []Symbol
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		var name string
		var kind string
		var startLine, endLine int

		for _, cap := range match.Captures {
			node := cap.Node

			switch cap.Name {
			case "name":
				name = cap.Text(content)
			case "func":
				kind = "function"
				startLine = int(node.StartPoint().Row) + 1
				endLine = int(node.EndPoint().Row) + 1
			case "method":
				kind = "method"
				startLine = int(node.StartPoint().Row) + 1
				endLine = int(node.EndPoint().Row) + 1
			case "class":
				kind = "class"
				startLine = int(node.StartPoint().Row) + 1
				endLine = int(node.EndPoint().Row) + 1
			}
		}

		if name != "" && kind != "" {
			syms = append(syms, Symbol{
				Name:      name,
				Kind:      kind,
				StartLine: startLine,
				EndLine:   endLine,
			})
		}
	}

	sort.Slice(syms, func(i, j int) bool {
		return syms[i].StartLine < syms[j].StartLine
	})

	return syms
}

// FindAt returns the innermost symbol containing the given line (1-indexed).
func FindAt(symbols []Symbol, line int) *Symbol {
	var best *Symbol
	for i := range symbols {
		s := &symbols[i]
		if line >= s.StartLine && line <= s.EndLine {
			if best == nil || (s.EndLine-s.StartLine) < (best.EndLine-best.StartLine) {
				best = s
			}
		}
	}
	return best
}
