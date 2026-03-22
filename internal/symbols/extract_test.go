package symbols

import (
	"testing"
)

func TestExtractGo(t *testing.T) {
	src := []byte(`package main

import "fmt"

func hello() {
	fmt.Println("hello")
}

func add(a, b int) int {
	return a + b
}
`)

	syms := Extract("main.go", src)
	if len(syms) != 2 {
		t.Fatalf("want 2 symbols, got %d: %+v", len(syms), syms)
	}

	if syms[0].Name != "hello" || syms[0].Kind != "function" {
		t.Errorf("syms[0]: want hello/function, got %s/%s", syms[0].Name, syms[0].Kind)
	}
	if syms[0].StartLine != 5 || syms[0].EndLine != 7 {
		t.Errorf("hello lines: want 5-7, got %d-%d", syms[0].StartLine, syms[0].EndLine)
	}

	if syms[1].Name != "add" || syms[1].Kind != "function" {
		t.Errorf("syms[1]: want add/function, got %s/%s", syms[1].Name, syms[1].Kind)
	}
}

func TestExtractGoMethod(t *testing.T) {
	src := []byte(`package main

type Server struct{}

func (s *Server) Start() error {
	return nil
}

func (s *Server) Stop() {
}
`)

	syms := Extract("server.go", src)
	if len(syms) != 2 {
		t.Fatalf("want 2 symbols, got %d: %+v", len(syms), syms)
	}

	if syms[0].Name != "Start" || syms[0].Kind != "method" {
		t.Errorf("syms[0]: want Start/method, got %s/%s", syms[0].Name, syms[0].Kind)
	}
	if syms[1].Name != "Stop" || syms[1].Kind != "method" {
		t.Errorf("syms[1]: want Stop/method, got %s/%s", syms[1].Name, syms[1].Kind)
	}
}

func TestExtractPython(t *testing.T) {
	src := []byte(`class AuthService:
    def refresh_token(self):
        pass

    def logout(self):
        pass

def standalone():
    pass
`)

	syms := Extract("auth.py", src)
	if len(syms) < 3 {
		t.Fatalf("want at least 3 symbols, got %d: %+v", len(syms), syms)
	}

	// Should have AuthService class and functions
	names := map[string]bool{}
	for _, s := range syms {
		names[s.Name] = true
	}
	for _, want := range []string{"AuthService", "refresh_token", "standalone"} {
		if !names[want] {
			t.Errorf("missing symbol %q in %+v", want, syms)
		}
	}
}

func TestExtractTypeScript(t *testing.T) {
	src := []byte(`function processPayment(amount: number) {
  return amount * 1.1;
}

class PaymentService {
  process() {
    return true;
  }
}
`)

	syms := Extract("payment.ts", src)
	if len(syms) < 2 {
		t.Fatalf("want at least 2 symbols, got %d: %+v", len(syms), syms)
	}

	names := map[string]bool{}
	for _, s := range syms {
		names[s.Name] = true
	}
	if !names["processPayment"] {
		t.Errorf("missing processPayment in %+v", syms)
	}
	if !names["PaymentService"] {
		t.Errorf("missing PaymentService in %+v", syms)
	}
}

func TestExtractUnsupportedLanguage(t *testing.T) {
	syms := Extract("data.csv", []byte("a,b,c"))
	if syms != nil {
		t.Errorf("want nil for unsupported language, got %+v", syms)
	}
}

func TestFindAt(t *testing.T) {
	syms := []Symbol{
		{Name: "hello", Kind: "function", StartLine: 5, EndLine: 7},
		{Name: "add", Kind: "function", StartLine: 9, EndLine: 11},
	}

	// Line inside hello
	s := FindAt(syms, 6)
	if s == nil || s.Name != "hello" {
		t.Errorf("line 6: want hello, got %v", s)
	}

	// Line inside add
	s = FindAt(syms, 10)
	if s == nil || s.Name != "add" {
		t.Errorf("line 10: want add, got %v", s)
	}

	// Line outside any function
	s = FindAt(syms, 1)
	if s != nil {
		t.Errorf("line 1: want nil, got %v", s)
	}

	// Line between functions
	s = FindAt(syms, 8)
	if s != nil {
		t.Errorf("line 8: want nil, got %v", s)
	}
}

func TestFindAtNested(t *testing.T) {
	syms := []Symbol{
		{Name: "AuthService", Kind: "class", StartLine: 1, EndLine: 10},
		{Name: "refresh", Kind: "method", StartLine: 3, EndLine: 5},
	}

	// Line inside method (nested in class) → should return method (innermost)
	s := FindAt(syms, 4)
	if s == nil || s.Name != "refresh" {
		t.Errorf("line 4: want refresh (innermost), got %v", s)
	}

	// Line in class but outside method
	s = FindAt(syms, 8)
	if s == nil || s.Name != "AuthService" {
		t.Errorf("line 8: want AuthService, got %v", s)
	}
}
