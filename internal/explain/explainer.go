// Package explain implements the wolf explain AI error diagnostic tool.
// It pattern-matches Wolf/LLVM/C compiler error output and produces
// human-readable explanations with fix suggestions — inspired by Rust's
// legendary error messages.
//
// Architecture:
//   - ExplainCache: the JSON written by `wolf build` on failure
//   - Explainer: the pattern-matching engine
//   - Explanation: a structured result with description + fix hints
//   - CacheFile: the default path for the explain cache
package explain

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CacheFile is the name of the file that `wolf build` writes on failure.
// It lives in the project root (same dir as wolf.config).
const CacheFile = ".wolf_explain_cache"

// ExplainCache is the JSON structure written by `wolf build` on failure.
type ExplainCache struct {
	File    string   `json:"file"`    // Source file that failed
	Errors  []string `json:"errors"`  // Raw error strings from the compiler
	Phase   string   `json:"phase"`   // "parse", "resolve", "typecheck", "llvm", "link", "runtime"
	Version string   `json:"version"` // wolf version at time of failure
}

// Explanation is the structured result returned by Explainer.Explain.
type Explanation struct {
	// ErrorCode is a short wolf-specific error code (e.g., "W-E001").
	ErrorCode string
	// Summary is one sentence describing the problem.
	Summary string
	// Detail is a fuller explanation of why this error occurs.
	Detail string
	// Fix is an actionable suggestion to resolve the error.
	Fix string
	// Example shows a corrected code snippet when applicable.
	Example string
	// Phase is the compiler phase where the error occurred.
	Phase string
	// Raw is the original error string from the compiler.
	Raw string
}

// Explainer is the main error pattern-matching engine.
type Explainer struct {
	patterns []pattern
}

// New creates a new Explainer with the full built-in pattern database.
func New() *Explainer {
	return &Explainer{patterns: allPatterns()}
}

// ExplainError takes a single raw error string and returns an Explanation.
// It tries each pattern in priority order and returns the first match.
// If no pattern matches, it returns a generic explanation.
func (e *Explainer) ExplainError(raw, phase string) Explanation {
	rawLower := strings.ToLower(raw)
	for _, p := range e.patterns {
		for _, kw := range p.keywords {
			if strings.Contains(rawLower, strings.ToLower(kw)) {
				return Explanation{
					ErrorCode: p.code,
					Summary:   p.summary,
					Detail:    p.detail,
					Fix:       p.fix,
					Example:   p.example,
					Phase:     phase,
					Raw:       raw,
				}
			}
		}
	}
	return genericExplanation(raw, phase)
}

// ExplainCache loads the cache file from the given project root and returns
// all explanations for the recorded errors.
func (e *Explainer) ExplainCache(projectRoot string) ([]Explanation, error) {
	cachePath := filepath.Join(projectRoot, CacheFile)
	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no explain cache found — run 'wolf build' first to capture an error")
		}
		return nil, fmt.Errorf("cannot read explain cache: %w", err)
	}

	var cache ExplainCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("explain cache is malformed: %w", err)
	}

	var results []Explanation
	for _, raw := range cache.Errors {
		results = append(results, e.ExplainError(raw, cache.Phase))
	}
	if len(results) == 0 {
		results = append(results, Explanation{
			ErrorCode: "W-I001",
			Summary:   "No errors recorded in cache.",
			Detail:    "The last build captured no error messages.",
			Fix:       "Run 'wolf build <file.wolf>' to capture a new error.",
			Phase:     cache.Phase,
		})
	}
	return results, nil
}

// WriteCache writes an ExplainCache to the project root.
// Called by `wolf build` immediately before returning an error to the user.
func WriteCache(projectRoot, file, phase string, errors []string) {
	cache := ExplainCache{
		File:    file,
		Errors:  errors,
		Phase:   phase,
		Version: "0.1.0-dev",
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return // non-fatal, best-effort
	}
	cachePath := filepath.Join(projectRoot, CacheFile)
	_ = os.WriteFile(cachePath, data, 0644)
}

// Format renders an Explanation as a human-readable terminal string.
// Modelled after Rust's error output format.
func Format(ex Explanation, index, total int) string {
	var sb strings.Builder

	if total > 1 {
		sb.WriteString(fmt.Sprintf("\n╔═ Error %d of %d [%s] ═══════════════════════════════════════╗\n", index, total, ex.ErrorCode))
	} else {
		sb.WriteString(fmt.Sprintf("\n╔═ Error [%s] ══════════════════════════════════════════════════╗\n", ex.ErrorCode))
	}

	// Raw compiler message
	sb.WriteString(fmt.Sprintf("║\n║  📋 Compiler says:\n║     %s\n║\n", wrapWords(ex.Raw, 70, "║     ")))

	// Summary
	sb.WriteString(fmt.Sprintf("║  🔍 What this means:\n║     %s\n║\n", wrapWords(ex.Summary, 70, "║     ")))

	// Detail
	if ex.Detail != "" {
		sb.WriteString(fmt.Sprintf("║  📖 Why it happens:\n║     %s\n║\n", wrapWords(ex.Detail, 70, "║     ")))
	}

	// Fix
	if ex.Fix != "" {
		sb.WriteString(fmt.Sprintf("║  ✅ How to fix it:\n║     %s\n║\n", wrapWords(ex.Fix, 70, "║     ")))
	}

	// Example
	if ex.Example != "" {
		sb.WriteString("║  💡 Example:\n")
		for _, line := range strings.Split(ex.Example, "\n") {
			sb.WriteString(fmt.Sprintf("║     %s\n", line))
		}
		sb.WriteString("║\n")
	}

	// Phase
	if ex.Phase != "" {
		sb.WriteString(fmt.Sprintf("║  ⚙  Phase: %s\n", ex.Phase))
	}

	sb.WriteString("╚═══════════════════════════════════════════════════════════════╝\n")
	return sb.String()
}

// genericExplanation returns a helpful fallback for unrecognised errors.
func genericExplanation(raw, phase string) Explanation {
	return Explanation{
		ErrorCode: "W-E000",
		Summary:   "An unrecognised compiler error occurred.",
		Detail:    "This error did not match any known Wolf error pattern. This may be a Wolf compiler bug or a very unusual code pattern.",
		Fix:       "1. Read the raw error above carefully — the file and line number tell you exactly where to look.\n2. Run 'wolf check <file.wolf>' for type-only diagnostics without a full build.\n3. If the error references LLVM IR, run 'wolf_out/main <file.wolf> --dump-wir' to inspect the WIR before lowering.\n4. File a bug at https://github.com/wolflang/wolf/issues with a minimal reproduction.",
		Phase:     phase,
		Raw:       raw,
	}
}

// wrapWords wraps text at a given width, prefixing continuation lines.
func wrapWords(text string, width int, prefix string) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return text
	}
	var lines []string
	current := ""
	for _, w := range words {
		if len(current)+len(w)+1 > width && current != "" {
			lines = append(lines, current)
			current = w
		} else {
			if current == "" {
				current = w
			} else {
				current += " " + w
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return strings.Join(lines, "\n"+prefix)
}
