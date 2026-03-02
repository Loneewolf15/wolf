// Package mlbridge provides the Wolf-to-Python ML bridge.
// It enables @ml blocks to execute Python code from Wolf programs,
// bridging variables between the two runtimes.
//
// The initial implementation uses subprocess execution (os/exec)
// for portability. A future cgo+libpython implementation can be
// swapped in for production performance.
package mlbridge

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	wrapper := b.buildWrapper(pythonSrc, inVars, outVars)

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
func (b *Bridge) buildWrapper(userCode string, inVars map[string]interface{}, outVars []string) string {
	var sb strings.Builder

	sb.WriteString("import json, sys\n")

	// Inject input variables
	if len(inVars) > 0 {
		inJSON, _ := json.Marshal(inVars)
		sb.WriteString(fmt.Sprintf("__wolf_in__ = json.loads('%s')\n", string(inJSON)))
		for k := range inVars {
			sb.WriteString(fmt.Sprintf("%s = __wolf_in__['%s']\n", k, k))
		}
	}

	// User code
	sb.WriteString("\n")
	sb.WriteString(userCode)
	sb.WriteString("\n")

	// Extract output variables
	if len(outVars) > 0 {
		sb.WriteString("\n__wolf_out__ = {}\n")
		for _, v := range outVars {
			// Strip $ prefix if present
			clean := v
			if strings.HasPrefix(clean, "$") {
				clean = clean[1:]
			}
			sb.WriteString(fmt.Sprintf("try:\n    __wolf_out__['%s'] = %s\nexcept NameError:\n    __wolf_out__['%s'] = None\n", clean, clean, clean))
		}
		sb.WriteString("print('__WOLF_OUT__:' + json.dumps(__wolf_out__, default=str))\n")
	}

	return sb.String()
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
