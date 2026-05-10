package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectCreation(t *testing.T) {
	tmpDir := t.TempDir()
	name := filepath.Join(tmpDir, "test-app")

	err := Project(name, TypeScript)
	if err != nil {
		t.Fatalf("Project() failed: %v", err)
	}

	// Check directories
	dirs := []string{
		name,
		filepath.Join(name, "src"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s", dir)
		}
	}

	// Check files — wolf.config is now in project root (not config/wolf.config.json)
	files := []string{
		filepath.Join(name, "src", "main.wolf"),
		filepath.Join(name, "wolf.config"),
		filepath.Join(name, "wolf.python"),
		filepath.Join(name, ".gitignore"),
		filepath.Join(name, "README.md"),
	}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file %s", file)
		}
	}

	// Check wolf.config contains [target] mode=script
	configData, _ := os.ReadFile(filepath.Join(name, "wolf.config"))
	if !strings.Contains(string(configData), "mode = script") {
		t.Error("wolf.config should contain 'mode = script' for script projects")
	}

	// Check main.wolf content
	data, _ := os.ReadFile(filepath.Join(name, "src", "main.wolf"))
	if len(data) == 0 {
		t.Error("main.wolf should not be empty")
	}
}

func TestProjectEmptyName(t *testing.T) {
	err := Project("", TypeScript)
	if err == nil {
		t.Error("Expected error for empty name")
	}
}

func TestAPIProjectCreation(t *testing.T) {
	tmpDir := t.TempDir()
	name := filepath.Join(tmpDir, "test-api")

	err := Project(name, TypeAPI)
	if err != nil {
		t.Fatalf("Project(API) failed: %v", err)
	}

	// Check directories
	dirs := []string{
		name,
		filepath.Join(name, "controllers"),
		filepath.Join(name, "models"),
		filepath.Join(name, "libraries"),
		filepath.Join(name, "public"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s", dir)
		}
	}

	// Check files — wolf.config is now TOML/INI in project root, not JSON in config/
	files := []string{
		filepath.Join(name, "public", "index.wolf"),
		filepath.Join(name, "docker-compose.yml"),
		filepath.Join(name, "wolf.config"),
	}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file %s", file)
		}
	}

	// Check wolf.config contains [target] mode=api and workers = 0
	configData, _ := os.ReadFile(filepath.Join(name, "wolf.config"))
	cfg := string(configData)
	if !strings.Contains(cfg, "mode = api") {
		t.Error("API wolf.config should contain 'mode = api'")
	}
	if !strings.Contains(cfg, "workers = 0") {
		t.Error("API wolf.config should set workers = 0 (auto-detect)")
	}
}

func TestMCUProjectCreation(t *testing.T) {
	tmpDir := t.TempDir()
	name := filepath.Join(tmpDir, "test-mcu")

	err := Project(name, TypeMCU)
	if err != nil {
		t.Fatalf("Project(MCU) failed: %v", err)
	}

	// Check MCU-specific directories
	dirs := []string{
		name,
		filepath.Join(name, "src"),
		filepath.Join(name, "hal"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s", dir)
		}
	}

	// Check files — no docker-compose, no wolf.python on bare metal
	files := []string{
		filepath.Join(name, "src", "main.wolf"),
		filepath.Join(name, "hal", "board.wolf"),
		filepath.Join(name, "wolf.config"),
		filepath.Join(name, "Makefile"),
		filepath.Join(name, ".gitignore"),
		filepath.Join(name, "README.md"),
	}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file %s", file)
		}
	}

	// Should NOT have docker-compose or wolf.python (not relevant on bare metal)
	noFiles := []string{
		filepath.Join(name, "docker-compose.yml"),
		filepath.Join(name, "wolf.python"),
	}
	for _, file := range noFiles {
		if _, err := os.Stat(file); err == nil {
			t.Errorf("MCU project should NOT contain %s", file)
		}
	}

	// Check wolf.config has correct MCU target
	configData, _ := os.ReadFile(filepath.Join(name, "wolf.config"))
	cfg := string(configData)
	if !strings.Contains(cfg, "mode = mcu") {
		t.Error("MCU wolf.config should contain 'mode = mcu'")
	}
	if !strings.Contains(cfg, "arm-cortex-m4") {
		t.Error("MCU wolf.config should specify arch = arm-cortex-m4")
	}

	// Check main.wolf has the cooperative loop pattern
	mainData, _ := os.ReadFile(filepath.Join(name, "src", "main.wolf"))
	if !strings.Contains(string(mainData), "wolf_mcu_run") {
		t.Error("MCU main.wolf should contain wolf_mcu_run() call")
	}
}
