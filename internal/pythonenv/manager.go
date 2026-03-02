// Package pythonenv manages Wolf's Python virtual environment.
// It handles wolf.python config parsing, venv creation, package
// installation, and auto-activation for @ml block execution.
package pythonenv

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Config represents the wolf.python configuration file.
type Config struct {
	PythonVersion string            `json:"python_version"`
	Packages      []PackageSpec     `json:"packages"`
	VenvPath      string            `json:"venv_path"`
	AutoActivate  bool              `json:"auto_activate"`
	Env           map[string]string `json:"env,omitempty"`
}

// PackageSpec represents a Python package requirement.
type PackageSpec struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"` // e.g. ">=1.0", "==2.3.1"
}

// String returns pip-compatible requirement string.
func (p PackageSpec) String() string {
	if p.Version != "" {
		return p.Name + p.Version
	}
	return p.Name
}

// LockFile represents the wolf.python.lock file.
type LockFile struct {
	GeneratedAt   string      `json:"generated_at"`
	PythonVersion string      `json:"python_version"`
	PythonPath    string      `json:"python_path"`
	Packages      []LockedPkg `json:"packages"`
}

// LockedPkg is a resolved package with exact version.
type LockedPkg struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Manager handles Python environment operations.
type Manager struct {
	projectDir string
	config     *Config
	venvDir    string
}

// NewManager creates a new Python environment manager.
func NewManager(projectDir string) *Manager {
	return &Manager{
		projectDir: projectDir,
		venvDir:    filepath.Join(projectDir, ".wolf-venv"),
	}
}

// ========== Config Operations ==========

// LoadConfig reads and parses wolf.python from the project directory.
func (m *Manager) LoadConfig() (*Config, error) {
	configPath := filepath.Join(m.projectDir, "wolf.python")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config
			m.config = &Config{
				PythonVersion: "3",
				AutoActivate:  true,
				VenvPath:      ".wolf-venv",
			}
			return m.config, nil
		}
		return nil, fmt.Errorf("failed to read wolf.python: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse wolf.python: %w", err)
	}

	if cfg.VenvPath == "" {
		cfg.VenvPath = ".wolf-venv"
	}
	if cfg.PythonVersion == "" {
		cfg.PythonVersion = "3"
	}

	m.config = &cfg
	m.venvDir = filepath.Join(m.projectDir, cfg.VenvPath)
	return &cfg, nil
}

// SaveConfig writes the config to wolf.python.
func (m *Manager) SaveConfig() error {
	if m.config == nil {
		return fmt.Errorf("no config loaded")
	}
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	configPath := filepath.Join(m.projectDir, "wolf.python")
	return os.WriteFile(configPath, data, 0644)
}

// InitConfig creates a new wolf.python config with defaults.
func (m *Manager) InitConfig() (*Config, error) {
	m.config = &Config{
		PythonVersion: "3",
		Packages:      []PackageSpec{},
		VenvPath:      ".wolf-venv",
		AutoActivate:  true,
	}
	if err := m.SaveConfig(); err != nil {
		return nil, err
	}
	return m.config, nil
}

// ========== Venv Operations ==========

// Install creates the virtual environment and installs packages.
func (m *Manager) Install() error {
	if m.config == nil {
		if _, err := m.LoadConfig(); err != nil {
			return err
		}
	}

	// Create venv if it doesn't exist
	if !m.VenvExists() {
		pythonCmd := "python" + m.config.PythonVersion
		cmd := exec.Command(pythonCmd, "-m", "venv", m.venvDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			// Fallback to python3
			cmd = exec.Command("python3", "-m", "venv", m.venvDir)
			if out, err := cmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to create venv: %s\n%s", err, string(out))
			}
			_ = out
		} else {
			_ = out
		}
	}

	// Install packages
	if len(m.config.Packages) > 0 {
		if err := m.installPackages(m.config.Packages); err != nil {
			return err
		}
	}

	// Generate lock file
	return m.GenerateLock()
}

// VenvExists checks if the virtual environment exists.
func (m *Manager) VenvExists() bool {
	pipPath := m.PipPath()
	_, err := os.Stat(pipPath)
	return err == nil
}

// PythonPath returns the path to the venv's Python interpreter.
func (m *Manager) PythonPath() string {
	return filepath.Join(m.venvDir, "bin", "python")
}

// PipPath returns the path to the venv's pip.
func (m *Manager) PipPath() string {
	return filepath.Join(m.venvDir, "bin", "pip")
}

// VenvDir returns the virtual environment directory.
func (m *Manager) VenvDir() string {
	return m.venvDir
}

// Add adds a package to the config and installs it.
func (m *Manager) Add(name, version string) error {
	if m.config == nil {
		if _, err := m.LoadConfig(); err != nil {
			return err
		}
	}

	// Check if already in config
	for i, pkg := range m.config.Packages {
		if pkg.Name == name {
			m.config.Packages[i].Version = version
			if err := m.SaveConfig(); err != nil {
				return err
			}
			return m.installPackages([]PackageSpec{{Name: name, Version: version}})
		}
	}

	// Add new package
	spec := PackageSpec{Name: name, Version: version}
	m.config.Packages = append(m.config.Packages, spec)
	if err := m.SaveConfig(); err != nil {
		return err
	}

	if m.VenvExists() {
		return m.installPackages([]PackageSpec{spec})
	}
	return nil
}

// Remove removes a package from the config and uninstalls it.
func (m *Manager) Remove(name string) error {
	if m.config == nil {
		if _, err := m.LoadConfig(); err != nil {
			return err
		}
	}

	found := false
	var newPkgs []PackageSpec
	for _, pkg := range m.config.Packages {
		if pkg.Name == name {
			found = true
		} else {
			newPkgs = append(newPkgs, pkg)
		}
	}

	if !found {
		return fmt.Errorf("package '%s' not found in wolf.python", name)
	}

	m.config.Packages = newPkgs
	if err := m.SaveConfig(); err != nil {
		return err
	}

	// Uninstall from venv
	if m.VenvExists() {
		cmd := exec.Command(m.PipPath(), "uninstall", "-y", name)
		cmd.CombinedOutput()
	}

	return nil
}

// List returns the list of installed packages in the venv.
func (m *Manager) List() ([]LockedPkg, error) {
	if !m.VenvExists() {
		return nil, fmt.Errorf("no virtual environment found. Run 'wolf python install' first")
	}

	cmd := exec.Command(m.PipPath(), "list", "--format=json", "--disable-pip-version-check")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("pip list failed: %w", err)
	}

	var pkgs []LockedPkg
	if err := json.Unmarshal(out, &pkgs); err != nil {
		return nil, fmt.Errorf("failed to parse pip output: %w", err)
	}

	return pkgs, nil
}

// Check verifies that all configured packages are installed.
func (m *Manager) Check() ([]string, error) {
	if m.config == nil {
		if _, err := m.LoadConfig(); err != nil {
			return nil, err
		}
	}

	if !m.VenvExists() {
		return []string{"virtual environment not found"}, nil
	}

	installed, err := m.List()
	if err != nil {
		return nil, err
	}

	installedMap := make(map[string]string)
	for _, pkg := range installed {
		installedMap[strings.ToLower(pkg.Name)] = pkg.Version
	}

	var issues []string
	for _, pkg := range m.config.Packages {
		if _, ok := installedMap[strings.ToLower(pkg.Name)]; !ok {
			issues = append(issues, fmt.Sprintf("missing: %s", pkg.Name))
		}
	}

	return issues, nil
}

// Reset destroys and recreates the virtual environment.
func (m *Manager) Reset() error {
	if err := os.RemoveAll(m.venvDir); err != nil {
		return fmt.Errorf("failed to remove venv: %w", err)
	}
	return m.Install()
}

// Shell returns the command to activate the venv shell.
func (m *Manager) Shell() string {
	return fmt.Sprintf("source %s/bin/activate", m.venvDir)
}

// ========== Lock File ==========

// GenerateLock creates the wolf.python.lock file from installed packages.
func (m *Manager) GenerateLock() error {
	if !m.VenvExists() {
		return fmt.Errorf("no venv to lock")
	}

	pkgs, err := m.List()
	if err != nil {
		return err
	}

	// Get Python version
	cmd := exec.Command(m.PythonPath(), "--version")
	verOut, _ := cmd.CombinedOutput()
	pyVer := strings.TrimSpace(string(verOut))

	lock := &LockFile{
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		PythonVersion: pyVer,
		PythonPath:    m.PythonPath(),
		Packages:      pkgs,
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}

	lockPath := filepath.Join(m.projectDir, "wolf.python.lock")
	return os.WriteFile(lockPath, data, 0644)
}

// ========== Internal ==========

func (m *Manager) installPackages(pkgs []PackageSpec) error {
	var args []string
	args = append(args, "install")
	for _, pkg := range pkgs {
		args = append(args, pkg.String())
	}

	cmd := exec.Command(m.PipPath(), args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pip install failed: %s\n%s", err, string(out))
	}

	return nil
}
