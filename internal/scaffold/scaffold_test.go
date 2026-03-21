package scaffold

import (
	"os"
	"path/filepath"
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
		filepath.Join(name, "config"),
	}
	for _, dir := range dirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory %s", dir)
		}
	}

	// Check files
	files := []string{
		filepath.Join(name, "src", "main.wolf"),
		filepath.Join(name, "config", "wolf.config.json"),
		filepath.Join(name, "wolf.python"),
		filepath.Join(name, ".gitignore"),
		filepath.Join(name, "README.md"),
	}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file %s", file)
		}
	}

	// Check main.wolf content
	data, _ := os.ReadFile(filepath.Join(name, "src", "main.wolf"))
	content := string(data)
	if len(content) == 0 {
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

	// Check files
	files := []string{
		filepath.Join(name, "public", "index.wolf"),
		filepath.Join(name, "docker-compose.yml"),
		filepath.Join(name, "config", "wolf.config.json"),
	}
	for _, file := range files {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("Expected file %s", file)
		}
	}
}
