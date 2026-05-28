// Package mlbridge provides the Wolf-to-Python ML bridge.
// It enables @ml blocks to execute Python code from Wolf programs,
// bridging variables between the two runtimes.
//
// The initial implementation uses subprocess execution (os/exec)
// for portability. A future cgo+libpython implementation can be
// swapped in for production performance.
package mlbridge

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// Bridge manages Python execution for @ml blocks.
type Bridge struct {
	mu          sync.Mutex
	pythonPath  string
	venvPath    string
	modelCache  map[string]*CachedModel
	initialized bool
}

// CachedModel holds a cached ML model reference.
type CachedModel struct {
	Name   string
	Path   string
	Loaded bool
}

// ExecResult holds the result of an @ml block execution.
type ExecResult struct {
	Output map[string]interface{} // out variables
	Stdout string                 // captured stdout
	Error  error
}

// New creates a new ML Bridge.
func New() *Bridge {
	return &Bridge{
		modelCache: make(map[string]*CachedModel),
	}
}

// Init initializes the Python bridge, finding the Python interpreter.
func (b *Bridge) Init() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.initialized {
		return nil
	}

	// Find Python 3
	pythonPath, err := findPython()
	if err != nil {
		return fmt.Errorf("ml bridge: %w", err)
	}
	b.pythonPath = pythonPath
	b.initialized = true
	return nil
}

// Shutdown cleans up the Python bridge.
func (b *Bridge) Shutdown() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.initialized = false
	b.modelCache = make(map[string]*CachedModel)
	return nil
}

// IsInitialized returns whether the bridge is ready.
func (b *Bridge) IsInitialized() bool {
	return b.initialized
}

// PythonPath returns the resolved Python interpreter path.
func (b *Bridge) PythonPath() string {
	return b.pythonPath
}

// validPythonIdent matches safe Python identifier names: [a-zA-Z_][a-zA-Z0-9_]*
var validPythonIdent = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// isValidPythonIdentifier returns true only for safe, injection-free variable names.
func isValidPythonIdentifier(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	return validPythonIdent.MatchString(name)
}

// Exec executes a Python source string with variable bridging.
// inVars are injected into the Python namespace, outVars are extracted after execution.
func (b *Bridge) Exec(pythonSrc string, inVars map[string]interface{}, outVars []string) (*ExecResult, error) {
	if !b.initialized {
		if err := b.Init(); err != nil {
			return nil, err
		}
	}

	result := &ExecResult{
		Output: make(map[string]interface{}),
	}

	// Build a wrapper Python script that:
	// 1. Deserializes input variables from JSON
	// 2. Executes the user's Python code
	// 3. Serializes output variables to JSON
	wrapper, err := b.buildWrapper(pythonSrc, inVars, outVars)
	if err != nil {
		return nil, err
	}

	// Execute via subprocess
	cmd := exec.Command(b.pythonPath, "-c", wrapper)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")

	// If venv is set, prepend it to PATH
	if b.venvPath != "" {
		venvBin := filepath.Join(b.venvPath, "bin")
		cmd.Env = append(cmd.Env, fmt.Sprintf("VIRTUAL_ENV=%s", b.venvPath))
		for i, env := range cmd.Env {
			if strings.HasPrefix(env, "PATH=") {
				cmd.Env[i] = fmt.Sprintf("PATH=%s:%s", venvBin, env[5:])
				break
			}
		}
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Error = fmt.Errorf("python execution failed: %w\n%s", err, string(output))
		return result, result.Error
	}

	// Parse the output — last line is JSON with out vars
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]

		// Try to parse the last line as our JSON output marker
		if strings.HasPrefix(lastLine, "__WOLF_OUT__:") {
			jsonStr := strings.TrimPrefix(lastLine, "__WOLF_OUT__:")
			if err := json.Unmarshal([]byte(jsonStr), &result.Output); err != nil {
				return result, fmt.Errorf("failed to parse output vars: %w", err)
			}
			// Stdout is everything except the last line
			if len(lines) > 1 {
				result.Stdout = strings.Join(lines[:len(lines)-1], "\n")
			}
		} else {
			// No output vars — all output is stdout
			result.Stdout = string(output)
		}
	}

	return result, nil
}

// ExecAsync runs an @ml block in a goroutine and returns the result via channel.
func (b *Bridge) ExecAsync(pythonSrc string, inVars map[string]interface{}, outVars []string) <-chan *ExecResult {
	ch := make(chan *ExecResult, 1)
	go func() {
		result, _ := b.Exec(pythonSrc, inVars, outVars)
		ch <- result
	}()
	return ch
}

// LoadModel caches a model reference for reuse across @ml blocks.
func (b *Bridge) LoadModel(name string) (*CachedModel, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if model, ok := b.modelCache[name]; ok {
		return model, nil
	}

	model := &CachedModel{
		Name:   name,
		Loaded: true,
	}
	b.modelCache[name] = model
	return model, nil
}

// GetModel retrieves a cached model.
func (b *Bridge) GetModel(name string) (*CachedModel, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	model, ok := b.modelCache[name]
	return model, ok
}

// SetVenvPath sets the virtual environment path for Python execution.
func (b *Bridge) SetVenvPath(path string) {
	b.venvPath = path
}

// ========== Internal Helpers ==========

// buildWrapper creates a Python wrapper script that handles variable bridging.
// Fix #5: all variable names are validated against a strict Python identifier regex
// before being embedded in code. JSON data is injected via base64 to prevent
// single-quote or backslash sequences in serialized values from breaking the script.
func (b *Bridge) buildWrapper(userCode string, inVars map[string]interface{}, outVars []string) (string, error) {
	var sb strings.Builder

	sb.WriteString("import json, sys, base64\n")

	// Inject input variables via base64-encoded JSON — avoids any quoting issues
	if len(inVars) > 0 {
		// Validate all variable names before touching the code string
		for k := range inVars {
			cleanK := strings.TrimPrefix(k, "$")
			if !isValidPythonIdentifier(cleanK) {
				return "", fmt.Errorf("ml bridge: invalid variable name %q (must be [a-zA-Z_][a-zA-Z0-9_]*)", k)
			}
		}
		inJSON, err := json.Marshal(inVars)
		if err != nil {
			return "", fmt.Errorf("ml bridge: failed to serialize input vars: %w", err)
		}
		// Embed as base64 — safe against any content in the JSON values
		b64 := base64.StdEncoding.EncodeToString(inJSON)
		sb.WriteString(fmt.Sprintf("__wolf_in__ = json.loads(base64.b64decode('%s').decode('utf-8'))\n", b64))
		for k := range inVars {
			clean := strings.TrimPrefix(k, "$")
			// clean is already validated above
			sb.WriteString(fmt.Sprintf("%s = __wolf_in__['%s']\n", clean, clean))
		}
	}

	// User code
	sb.WriteString("\n")
	sb.WriteString(userCode)
	sb.WriteString("\n")

	// Extract output variables — names validated against identifier regex
	if len(outVars) > 0 {
		sb.WriteString("\n__wolf_out__ = {}\n")
		for _, v := range outVars {
			clean := strings.TrimPrefix(v, "$")
			if !isValidPythonIdentifier(clean) {
				return "", fmt.Errorf("ml bridge: invalid output variable name %q", v)
			}
			sb.WriteString(fmt.Sprintf(
				"try:\n    __wolf_out__['%s'] = %s\nexcept NameError:\n    __wolf_out__['%s'] = None\n",
				clean, clean, clean))
		}
		sb.WriteString("print('__WOLF_OUT__:' + json.dumps(__wolf_out__, default=str))\n")
	}

	return sb.String(), nil
}

// findPython locates a Python 3 interpreter.
func findPython() (string, error) {
	candidates := []string{"python3", "python"}
	for _, name := range candidates {
		path, err := exec.LookPath(name)
		if err == nil {
			// Verify it's Python 3
			cmd := exec.Command(path, "--version")
			out, err := cmd.CombinedOutput()
			if err == nil && strings.Contains(string(out), "Python 3") {
				return path, nil
			}
		}
	}
	return "", fmt.Errorf("Python 3 not found. Install Python 3.8+ to use @ml blocks")
}
