package pythonenv

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// hasPython3 checks if Python 3 is available.
func hasPython3() bool {
	cmd := exec.Command("python3", "--version")
	out, err := cmd.CombinedOutput()
	return err == nil && strings.Contains(string(out), "Python 3")
}

// canCreateVenv checks if python3-venv module is available.
func canCreateVenv() bool {
	if !hasPython3() {
		return false
	}
	tmpDir, err := os.MkdirTemp("", "wolf-venv-check-*")
	if err != nil {
		return false
	}
	defer os.RemoveAll(tmpDir)
	venvPath := filepath.Join(tmpDir, "test-venv")
	cmd := exec.Command("python3", "-m", "venv", venvPath)
	return cmd.Run() == nil
}

// --- Config Tests ---

func TestNewManager(t *testing.T) {
	m := NewManager("/tmp/wolf-test")
	if m == nil {
		t.Fatal("Expected non-nil manager")
	}
	if m.projectDir != "/tmp/wolf-test" {
		t.Errorf("Expected project dir '/tmp/wolf-test', got '%s'", m.projectDir)
	}
}

func TestLoadConfigDefault(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	cfg, err := m.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.PythonVersion != "3" {
		t.Errorf("Expected default Python version '3', got '%s'", cfg.PythonVersion)
	}
	if !cfg.AutoActivate {
		t.Error("Expected auto_activate to default to true")
	}
	if cfg.VenvPath != ".wolf-venv" {
		t.Errorf("Expected venv_path '.wolf-venv', got '%s'", cfg.VenvPath)
	}
}

func TestInitAndSaveConfig(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)

	cfg, err := m.InitConfig()
	if err != nil {
		t.Fatalf("InitConfig failed: %v", err)
	}
	if cfg.PythonVersion != "3" {
		t.Errorf("Expected version '3', got '%s'", cfg.PythonVersion)
	}

	// Check file was written
	data, err := os.ReadFile(filepath.Join(tmpDir, "wolf.python"))
	if err != nil {
		t.Fatalf("Failed to read wolf.python: %v", err)
	}

	var loaded Config
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("Failed to parse saved config: %v", err)
	}
	if loaded.PythonVersion != "3" {
		t.Error("Saved config should have python_version '3'")
	}
}

func TestLoadConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := Config{
		PythonVersion: "3.11",
		Packages:      []PackageSpec{{Name: "numpy", Version: ">=1.21"}},
		VenvPath:      ".venv",
		AutoActivate:  true,
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	os.WriteFile(filepath.Join(tmpDir, "wolf.python"), data, 0644)

	m := NewManager(tmpDir)
	loaded, err := m.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if loaded.PythonVersion != "3.11" {
		t.Errorf("Expected '3.11', got '%s'", loaded.PythonVersion)
	}
	if len(loaded.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(loaded.Packages))
	}
	if loaded.Packages[0].Name != "numpy" {
		t.Error("Expected package 'numpy'")
	}
}

// --- PackageSpec ---

func TestPackageSpecString(t *testing.T) {
	tests := []struct {
		spec     PackageSpec
		expected string
	}{
		{PackageSpec{Name: "numpy"}, "numpy"},
		{PackageSpec{Name: "numpy", Version: ">=1.21"}, "numpy>=1.21"},
		{PackageSpec{Name: "torch", Version: "==2.0.0"}, "torch==2.0.0"},
	}

	for _, tt := range tests {
		got := tt.spec.String()
		if got != tt.expected {
			t.Errorf("PackageSpec.String() = %q, expected %q", got, tt.expected)
		}
	}
}

// --- Add/Remove ---

func TestAddPackageToConfig(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()

	// Add without venv (just config update)
	err := m.Add("numpy", ">=1.21")
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	if len(m.config.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(m.config.Packages))
	}
	if m.config.Packages[0].Name != "numpy" {
		t.Error("Expected package 'numpy'")
	}

	// Verify persisted
	data, _ := os.ReadFile(filepath.Join(tmpDir, "wolf.python"))
	var saved Config
	json.Unmarshal(data, &saved)
	if len(saved.Packages) != 1 || saved.Packages[0].Name != "numpy" {
		t.Error("Package not persisted to wolf.python")
	}
}

func TestAddDuplicateUpdatesVersion(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()

	m.Add("numpy", ">=1.21")
	m.Add("numpy", ">=1.24")

	if len(m.config.Packages) != 1 {
		t.Fatalf("Expected 1 package (deduped), got %d", len(m.config.Packages))
	}
	if m.config.Packages[0].Version != ">=1.24" {
		t.Errorf("Expected updated version '>=1.24', got '%s'", m.config.Packages[0].Version)
	}
}

func TestRemovePackageFromConfig(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()
	m.Add("numpy", "")
	m.Add("pandas", "")

	err := m.Remove("numpy")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if len(m.config.Packages) != 1 {
		t.Fatalf("Expected 1 package, got %d", len(m.config.Packages))
	}
	if m.config.Packages[0].Name != "pandas" {
		t.Error("Expected 'pandas' to remain")
	}
}

func TestRemoveNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()

	err := m.Remove("nonexistent")
	if err == nil {
		t.Error("Expected error for removing nonexistent package")
	}
}

// --- Venv Operations ---

func TestVenvDoesNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	if m.VenvExists() {
		t.Error("Venv should not exist in empty tmpdir")
	}
}

func TestInstallCreatesVenv(t *testing.T) {
	if !canCreateVenv() {
		t.Skip("python3-venv not available")
	}

	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()

	err := m.Install()
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if !m.VenvExists() {
		t.Error("Venv should exist after Install")
	}

	// Check python binary exists
	if _, err := os.Stat(m.PythonPath()); os.IsNotExist(err) {
		t.Error("Python binary should exist in venv")
	}
}

func TestPaths(t *testing.T) {
	m := NewManager("/tmp/myproject")
	m.LoadConfig()
	if !strings.Contains(m.PythonPath(), ".wolf-venv/bin/python") {
		t.Errorf("Unexpected python path: %s", m.PythonPath())
	}
	if !strings.Contains(m.PipPath(), ".wolf-venv/bin/pip") {
		t.Errorf("Unexpected pip path: %s", m.PipPath())
	}
}

func TestShellCommand(t *testing.T) {
	m := NewManager("/tmp/myproject")
	m.LoadConfig()
	shell := m.Shell()
	if !strings.Contains(shell, "source") || !strings.Contains(shell, "activate") {
		t.Errorf("Expected shell activation command, got '%s'", shell)
	}
}

// --- Lock File ---

func TestGenerateLock(t *testing.T) {
	if !canCreateVenv() {
		t.Skip("python3-venv not available")
	}

	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()
	m.Install()

	err := m.GenerateLock()
	if err != nil {
		t.Fatalf("GenerateLock failed: %v", err)
	}

	lockPath := filepath.Join(tmpDir, "wolf.python.lock")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal("Lock file should exist")
	}

	var lock LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		t.Fatalf("Failed to parse lock file: %v", err)
	}

	if !strings.Contains(lock.PythonVersion, "Python 3") {
		t.Errorf("Expected Python 3 in lock, got '%s'", lock.PythonVersion)
	}
	if lock.GeneratedAt == "" {
		t.Error("Expected generated_at timestamp")
	}
}

// --- Check ---

func TestCheckAllInstalled(t *testing.T) {
	if !canCreateVenv() {
		t.Skip("python3-venv not available")
	}

	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()
	m.Install()

	// pip is always installed, add it to config
	m.Add("pip", "")

	issues, err := m.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(issues) > 0 {
		t.Errorf("Expected no issues, got: %v", issues)
	}
}

func TestCheckMissingPackage(t *testing.T) {
	if !canCreateVenv() {
		t.Skip("python3-venv not available")
	}

	tmpDir := t.TempDir()
	m := NewManager(tmpDir)
	m.InitConfig()
	m.Install()

	// Add a package to config without installing it
	m.config.Packages = append(m.config.Packages, PackageSpec{Name: "nonexistent-wolf-package-xyz"})

	issues, err := m.Check()
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	if len(issues) == 0 {
		t.Error("Expected missing package issue")
	}
}
