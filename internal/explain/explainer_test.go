package explain_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wolflang/wolf/internal/explain"
)

// ── Explainer.ExplainError ────────────────────────────────────────────────────

func TestExplainError_UndefinedVariable(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("resolver error: undefined variable $counter", "resolve")
	if r.ErrorCode != "W-E010" {
		t.Errorf("expected W-E010, got %s", r.ErrorCode)
	}
	if r.Phase != "resolve" {
		t.Errorf("expected phase=resolve, got %s", r.Phase)
	}
	if !strings.Contains(r.Fix, "var $") {
		t.Errorf("fix should mention 'var $', got: %s", r.Fix)
	}
}

func TestExplainError_UntermString(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("unterminated string literal on line 12", "lex")
	if r.ErrorCode != "W-E002" {
		t.Errorf("expected W-E002, got %s", r.ErrorCode)
	}
}

func TestExplainError_ParseError(t *testing.T) {
	ex := explain.New()
	// "unexpected token" matches W-E001 (highest-priority unexpected-token pattern)
	// even when the token happens to be 'enum'. W-E008 only fires when the raw
	// error does NOT contain the "unexpected token" keyword.
	r := ex.ExplainError("parse error: unexpected token 'enum'", "parse")
	if r.ErrorCode != "W-E001" {
		t.Errorf("expected W-E001 (unexpected token), got %s", r.ErrorCode)
	}
}

func TestExplainError_EnumSyntaxError(t *testing.T) {
	ex := explain.New()
	// A pure enum syntax error that does not contain "unexpected token"
	r := ex.ExplainError("invalid enum declaration — expected enum value list", "parse")
	if r.ErrorCode != "W-E008" {
		t.Errorf("expected W-E008 (enum), got %s", r.ErrorCode)
	}
}

func TestExplainError_LLVMError(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("LLVM error: invalid IR — instruction references undefined value", "llvm")
	if r.ErrorCode != "W-E030" {
		t.Errorf("expected W-E030, got %s", r.ErrorCode)
	}
}

func TestExplainError_UndefinedReference_Linker(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("undefined reference to `wolf_http_serve'", "link")
	if r.ErrorCode != "W-E031" {
		t.Errorf("expected W-E031, got %s", r.ErrorCode)
	}
}

func TestExplainError_Segfault(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("runtime: signal 11 — segmentation fault in wolf_http_req_file", "runtime")
	if r.ErrorCode != "W-E032" {
		t.Errorf("expected W-E032, got %s", r.ErrorCode)
	}
}

func TestExplainError_LLCNotFound(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("llc: command not found — please install LLVM", "build")
	if r.ErrorCode != "W-E040" {
		t.Errorf("expected W-E040, got %s", r.ErrorCode)
	}
}

func TestExplainError_PortInUse(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("bind failed: address already in use on port 8080", "runtime")
	if r.ErrorCode != "W-E050" {
		t.Errorf("expected W-E050, got %s", r.ErrorCode)
	}
}

// Unknown errors must fall back to the generic pattern (W-E000), not panic.
func TestExplainError_Unknown_Fallback(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("some completely unknown compiler error xyzzy", "unknown")
	if r.ErrorCode != "W-E000" {
		t.Errorf("expected W-E000 fallback, got %s", r.ErrorCode)
	}
	if r.Raw == "" {
		t.Error("fallback explanation must preserve the raw error string")
	}
}

// Case-insensitive matching is required.
func TestExplainError_CaseInsensitive(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("UNDEFINED VARIABLE $Foo in scope", "resolve")
	if r.ErrorCode != "W-E010" {
		t.Errorf("expected W-E010 (case-insensitive), got %s", r.ErrorCode)
	}
}

// ── WriteCache / ExplainCache round-trip ─────────────────────────────────────

func TestWriteCache_And_ExplainCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	errs := []string{"resolver error: undefined variable $x", "parser errors: 1 errors found"}

	explain.WriteCache(dir, "test.wolf", "resolve", errs)

	cachePath := filepath.Join(dir, explain.CacheFile)
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Fatal("WriteCache did not create the cache file")
	}

	ex := explain.New()
	results, err := ex.ExplainCache(dir)
	if err != nil {
		t.Fatalf("ExplainCache returned error: %v", err)
	}
	if len(results) != len(errs) {
		t.Errorf("expected %d explanations, got %d", len(errs), len(results))
	}
	// First error should be recognised (undefined variable → W-E010)
	if results[0].ErrorCode != "W-E010" {
		t.Errorf("expected W-E010 for first error, got %s", results[0].ErrorCode)
	}
}

func TestExplainCache_NoCacheFile(t *testing.T) {
	dir := t.TempDir()
	ex := explain.New()
	_, err := ex.ExplainCache(dir)
	if err == nil {
		t.Error("expected error when no cache file exists")
	}
	if !strings.Contains(err.Error(), "wolf build") {
		t.Errorf("error should mention 'wolf build', got: %v", err)
	}
}

func TestExplainCache_EmptyErrors(t *testing.T) {
	dir := t.TempDir()
	// Write a cache with no errors
	explain.WriteCache(dir, "test.wolf", "parse", []string{})

	ex := explain.New()
	results, err := ex.ExplainCache(dir)
	if err != nil {
		t.Fatalf("ExplainCache returned error: %v", err)
	}
	// Should return exactly one "no errors recorded" result
	if len(results) != 1 {
		t.Errorf("expected 1 fallback result, got %d", len(results))
	}
	if results[0].ErrorCode != "W-I001" {
		t.Errorf("expected W-I001 info code, got %s", results[0].ErrorCode)
	}
}

func TestExplainCache_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, explain.CacheFile)
	_ = os.WriteFile(path, []byte("not-json{{{"), 0644)

	ex := explain.New()
	_, err := ex.ExplainCache(dir)
	if err == nil {
		t.Error("expected error for malformed cache JSON")
	}
}

// ── Format ────────────────────────────────────────────────────────────────────

func TestFormat_SingleError(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("undefined variable $count", "resolve")
	out := explain.Format(r, 1, 1)

	if !strings.Contains(out, "W-E010") {
		t.Error("formatted output should contain the error code")
	}
	if !strings.Contains(out, "╔") && !strings.Contains(out, "╚") {
		t.Error("formatted output should contain box-drawing characters")
	}
	if !strings.Contains(out, "undefined") {
		t.Error("formatted output should contain the raw error")
	}
}

func TestFormat_MultipleErrors_Index(t *testing.T) {
	ex := explain.New()
	r := ex.ExplainError("undefined variable $x", "resolve")
	out := explain.Format(r, 2, 5)

	if !strings.Contains(out, "Error 2 of 5") {
		t.Errorf("formatted output should show index/total, got: %s", out)
	}
}

// ── WriteCache is non-fatal (must not panic on bad dir) ──────────────────────

func TestWriteCache_NonFatal_BadDir(t *testing.T) {
	// Writing to a non-existent directory should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("WriteCache panicked: %v", r)
		}
	}()
	explain.WriteCache("/nonexistent/path/that/cannot/exist", "test.wolf", "parse", []string{"err"})
}

// ── Cache JSON structure ──────────────────────────────────────────────────────

func TestWriteCache_JSONStructure(t *testing.T) {
	dir := t.TempDir()
	explain.WriteCache(dir, "src/main.wolf", "llvm", []string{"LLVM error: bad IR"})

	data, err := os.ReadFile(filepath.Join(dir, explain.CacheFile))
	if err != nil {
		t.Fatalf("cannot read cache file: %v", err)
	}

	var cache struct {
		File    string   `json:"file"`
		Errors  []string `json:"errors"`
		Phase   string   `json:"phase"`
		Version string   `json:"version"`
	}
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("cache file is not valid JSON: %v", err)
	}
	if cache.File != "src/main.wolf" {
		t.Errorf("expected file=src/main.wolf, got %s", cache.File)
	}
	if cache.Phase != "llvm" {
		t.Errorf("expected phase=llvm, got %s", cache.Phase)
	}
	if len(cache.Errors) != 1 {
		t.Errorf("expected 1 error, got %d", len(cache.Errors))
	}
	if cache.Version == "" {
		t.Error("version should not be empty")
	}
}

// ── detectPhase is tested indirectly via WriteCache, but also directly here ──
// Note: detectPhase is in cmd/wolf/main.go (package main), so we test the
// phase detection logic by verifying ExplainCache respects the written phase.
func TestExplainCache_PhaseIsPropagated(t *testing.T) {
	dir := t.TempDir()
	explain.WriteCache(dir, "main.wolf", "typecheck", []string{"type mismatch: cannot assign string to int"})

	ex := explain.New()
	results, err := ex.ExplainCache(dir)
	if err != nil {
		t.Fatalf("ExplainCache error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Phase from the cache must flow through to the explanation
	if results[0].Phase != "typecheck" {
		t.Errorf("expected phase=typecheck, got %s", results[0].Phase)
	}
	// Type mismatch should match W-E020
	if results[0].ErrorCode != "W-E020" {
		t.Errorf("expected W-E020, got %s", results[0].ErrorCode)
	}
}
