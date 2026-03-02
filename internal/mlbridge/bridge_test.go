package mlbridge

import (
	"os/exec"
	"strings"
	"testing"
)

// hasPython3 checks if Python 3 is available for tests.
func hasPython3() bool {
	cmd := exec.Command("python3", "--version")
	out, err := cmd.CombinedOutput()
	return err == nil && strings.Contains(string(out), "Python 3")
}

// --- Bridge Lifecycle ---

func TestNewBridge(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("Expected non-nil bridge")
	}
	if b.IsInitialized() {
		t.Error("Bridge should not be initialized before Init()")
	}
}

func TestInitBridge(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	err := b.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if !b.IsInitialized() {
		t.Error("Bridge should be initialized after Init()")
	}
	if b.PythonPath() == "" {
		t.Error("PythonPath should be set after Init()")
	}
}

func TestShutdownBridge(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	err := b.Shutdown()
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
	if b.IsInitialized() {
		t.Error("Bridge should not be initialized after Shutdown()")
	}
}

// --- Python Execution ---

func TestExecSimple(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	defer b.Shutdown()

	result, err := b.Exec("print('hello from python')", nil, nil)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "hello from python") {
		t.Errorf("Expected stdout 'hello from python', got '%s'", result.Stdout)
	}
}

func TestExecWithInputVars(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	defer b.Shutdown()

	inVars := map[string]interface{}{
		"x": 10,
		"y": 20,
	}

	result, err := b.Exec("print(x + y)", inVars, nil)
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}
	if !strings.Contains(result.Stdout, "30") {
		t.Errorf("Expected stdout '30', got '%s'", result.Stdout)
	}
}

func TestExecWithOutputVars(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	defer b.Shutdown()

	inVars := map[string]interface{}{
		"items": []int{1, 2, 3, 4, 5},
	}

	result, err := b.Exec("total = sum(items)\navg = total / len(items)", inVars, []string{"total", "avg"})
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	if total, ok := result.Output["total"]; !ok {
		t.Error("Expected 'total' in output")
	} else {
		// JSON numbers come back as float64
		if totalF, ok := total.(float64); ok {
			if totalF != 15 {
				t.Errorf("Expected total=15, got %v", totalF)
			}
		}
	}

	if avg, ok := result.Output["avg"]; !ok {
		t.Error("Expected 'avg' in output")
	} else {
		if avgF, ok := avg.(float64); ok {
			if avgF != 3.0 {
				t.Errorf("Expected avg=3.0, got %v", avgF)
			}
		}
	}
}

func TestExecWithStringVars(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	defer b.Shutdown()

	inVars := map[string]interface{}{
		"name": "Wolf",
	}

	result, err := b.Exec("greeting = f'Hello, {name}!'", inVars, []string{"greeting"})
	if err != nil {
		t.Fatalf("Exec failed: %v", err)
	}

	if greeting, ok := result.Output["greeting"]; !ok {
		t.Error("Expected 'greeting' in output")
	} else if greeting != "Hello, Wolf!" {
		t.Errorf("Expected 'Hello, Wolf!', got '%v'", greeting)
	}
}

// --- Async Execution ---

func TestExecAsync(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	defer b.Shutdown()

	ch := b.ExecAsync("result = 42 * 2", nil, []string{"result"})
	res := <-ch

	if res.Error != nil {
		t.Fatalf("Async exec failed: %v", res.Error)
	}
	if val, ok := res.Output["result"]; !ok {
		t.Error("Expected 'result' in output")
	} else if valF, ok := val.(float64); ok && valF != 84 {
		t.Errorf("Expected 84, got %v", valF)
	}
}

// --- Model Cache ---

func TestModelCache(t *testing.T) {
	b := New()

	model, err := b.LoadModel("sentiment")
	if err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}
	if model.Name != "sentiment" {
		t.Errorf("Expected 'sentiment', got '%s'", model.Name)
	}
	if !model.Loaded {
		t.Error("Model should be loaded")
	}

	// Second load should return cached
	model2, err := b.LoadModel("sentiment")
	if err != nil {
		t.Fatalf("Second LoadModel failed: %v", err)
	}
	if model2 != model {
		t.Error("Expected cached model to be same instance")
	}

	// Get model
	got, ok := b.GetModel("sentiment")
	if !ok {
		t.Error("Expected to find cached model")
	}
	if got.Name != "sentiment" {
		t.Error("Expected model name 'sentiment'")
	}

	// Missing model
	_, ok = b.GetModel("nonexistent")
	if ok {
		t.Error("Expected missing model to return false")
	}
}

// --- Error Handling ---

func TestExecPythonError(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	defer b.Shutdown()

	_, err := b.Exec("raise ValueError('test error')", nil, nil)
	if err == nil {
		t.Error("Expected error from Python exception")
	}
}

func TestExecSyntaxError(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	b := New()
	b.Init()
	defer b.Shutdown()

	_, err := b.Exec("def broken(:", nil, nil)
	if err == nil {
		t.Error("Expected error from Python syntax error")
	}
}

// --- findPython ---

func TestFindPython(t *testing.T) {
	if !hasPython3() {
		t.Skip("Python 3 not available")
	}

	path, err := findPython()
	if err != nil {
		t.Fatalf("findPython failed: %v", err)
	}
	if path == "" {
		t.Error("Expected non-empty Python path")
	}
}

// --- Build Wrapper ---

func TestBuildWrapperNoVars(t *testing.T) {
	b := New()
	wrapper := b.buildWrapper("print('hello')", nil, nil)
	if !strings.Contains(wrapper, "print('hello')") {
		t.Error("Wrapper should contain user code")
	}
	if !strings.Contains(wrapper, "import json") {
		t.Error("Wrapper should import json")
	}
}

func TestBuildWrapperWithInVars(t *testing.T) {
	b := New()
	inVars := map[string]interface{}{"x": 42}
	wrapper := b.buildWrapper("print(x)", inVars, nil)
	if !strings.Contains(wrapper, "__wolf_in__") {
		t.Error("Wrapper should inject input vars")
	}
	if !strings.Contains(wrapper, "x = __wolf_in__['x']") {
		t.Error("Wrapper should assign x from input")
	}
}

func TestBuildWrapperWithOutVars(t *testing.T) {
	b := New()
	wrapper := b.buildWrapper("result = 42", nil, []string{"result"})
	if !strings.Contains(wrapper, "__WOLF_OUT__") {
		t.Error("Wrapper should output vars marker")
	}
	if !strings.Contains(wrapper, "__wolf_out__['result']") {
		t.Error("Wrapper should extract result")
	}
}

func TestBuildWrapperStripsDollar(t *testing.T) {
	b := New()
	wrapper := b.buildWrapper("result = 42", nil, []string{"$result"})
	if !strings.Contains(wrapper, "__wolf_out__['result']") {
		t.Error("Wrapper should strip $ prefix from output var names")
	}
}
