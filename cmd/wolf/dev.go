package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wolflang/wolf/internal/compiler"
	"github.com/wolflang/wolf/internal/config"
	"github.com/wolflang/wolf/internal/dashboard"
)

var devCmd = &cobra.Command{
	Use:   "dev [file]",
	Short: "Start the developer server with live reload on file change",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := ""
		if len(args) > 0 {
			filename = args[0]
		} else {
			if _, err := os.Stat("src/main.wolf"); err == nil {
				filename = "src/main.wolf"
			} else if _, err := os.Stat("main.wolf"); err == nil {
				filename = "main.wolf"
			} else {
				return fmt.Errorf("no input file specified. Please provide a file or create src/main.wolf")
			}
		}

		if !filepath.IsAbs(filename) {
			cwd, _ := os.Getwd()
			filename = filepath.Join(cwd, filename)
		}

		outDir := filepath.Join(filepath.Dir(filename), "wolf_out")

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("wolf: cannot determine working directory: %w", err)
		}

		// Use NewWithConfig which automatically loads the config and sets ProjectRoot
		c, err := compiler.NewWithConfig(cwd)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wolf: config warning: %v\n", err)
			c = compiler.New()
			c.ProjectRoot = cwd
		}

		// Load config if exists
		cfg := c.Config
		if cfg == nil {
			cfg = &config.WolfConfig{
				Target: config.TargetConfig{Mode: "script", Arch: "native"},
				DB:     config.DBConfig{Driver: "mysql", PoolSize: 10, PoolMinIdle: 2, PoolTimeout: 5000, MaxRetries: 3},
				Server: config.ServerConfig{MaxConcurrent: 1024, MaxRequestSize: 65536, MaxUploads: 8},
				App:    config.AppConfig{Env: "development", Debug: true},
			}
			c.Config = cfg
		}

		isVerbose, _ := cmd.Flags().GetBool("verbose")
		c.Verbose = isVerbose
		c.OutDir = outDir

		// Zero-Config Execution Heuristic — detect if this is an API or script
		configPath := filepath.Join(c.ProjectRoot, "wolf.config")
		isAPI := cfg.Target.Mode == "api"
		if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
			content, _ := os.ReadFile(filename)
			source := string(content)
			if strings.Contains(source, "wolf_http_serve") || strings.Contains(source, "route(") {
				cfg.Target.Mode = "api"
				isAPI = true
			} else {
				cfg.Target.Mode = "script"
				isAPI = false
			}
		}

		// Project root for watching
		watchRoot := c.ProjectRoot

		if isAPI {
			// ── HMR mode: compile as shared .so, run via wolf_host, send SIGUSR1 on change
			return runHMRMode(c, cfg, filename, outDir, watchRoot, isVerbose)
		}

		// ── Script mode: compile binary, run it, kill+restart on change
		return runScriptMode(c, filename, outDir, watchRoot)
	},
}

// walkWolfFiles returns a map of absolute path → mtime for all .wolf files
// found under root, skipping hidden directories, wolf_out, and node_modules.
func walkWolfFiles(root string) (map[string]time.Time, error) {
	mtimes := make(map[string]time.Time)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable entries gracefully
		}
		if info.IsDir() {
			base := info.Name()
			if strings.HasPrefix(base, ".") || base == "wolf_out" || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(info.Name(), ".wolf") {
			mtimes[path] = info.ModTime()
		}
		return nil
	})
	return mtimes, err
}

// hasChanges compares current file mtimes against the last snapshot.
// Returns changed=true plus the new snapshot if any file was added, removed, or modified.
func hasChanges(root string, lastMtimes map[string]time.Time) (bool, map[string]time.Time, error) {
	current, err := walkWolfFiles(root)
	if err != nil {
		return false, lastMtimes, err
	}
	// Check for modifications and new files
	for path, mtime := range current {
		if last, ok := lastMtimes[path]; !ok || mtime.After(last) {
			return true, current, nil
		}
	}
	// Check for deleted files
	if len(current) != len(lastMtimes) {
		return true, current, nil
	}
	return false, current, nil
}

// runScriptMode runs the compiled binary and restarts it whenever any .wolf file changes.
func runScriptMode(c *compiler.Compiler, filename, outDir, watchRoot string) error {
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	binaryPath := filepath.Join(outDir, baseName)

	// Initial build
	fmt.Printf("wolf dev: building %s...\n", filepath.Base(filename))
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}
	if _, err := c.Build(string(content), filename); err != nil {
		fmt.Fprintf(os.Stderr, "wolf dev: initial build failed:\n%v\n", err)
		// Don't exit — keep watching so developer can fix and retry
	}

	// Snapshot initial file tree
	lastMtimes, _ := walkWolfFiles(watchRoot)

	var mu sync.Mutex
	var runningCmd *exec.Cmd

	startBinary := func() {
		mu.Lock()
		defer mu.Unlock()
		if runningCmd != nil && runningCmd.Process != nil {
			_ = runningCmd.Process.Kill()
			_ = runningCmd.Wait()
			runningCmd = nil
		}
		if _, err := os.Stat(binaryPath); err != nil {
			return // binary doesn't exist yet (build may have failed)
		}
		cmd := exec.Command(binaryPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "wolf dev: failed to start binary: %v\n", err)
			return
		}
		runningCmd = cmd
		// Reap in background so we don't block the watcher
		go func() {
			_ = cmd.Wait()
		}()
	}

	stopBinary := func() {
		mu.Lock()
		defer mu.Unlock()
		if runningCmd != nil && runningCmd.Process != nil {
			_ = runningCmd.Process.Kill()
			_ = runningCmd.Wait()
			runningCmd = nil
		}
	}

	// Start the binary initially
	startBinary()

	fmt.Printf("wolf dev: watching %s for changes (script mode)...\n", watchRoot)

	// Polling watcher
	for {
		time.Sleep(200 * time.Millisecond)

		changed, newMtimes, err := hasChanges(watchRoot, lastMtimes)
		if err != nil {
			continue
		}
		if !changed {
			continue
		}

		lastMtimes = newMtimes
		fmt.Printf("\nwolf dev: change detected, rebuilding...\n")

		// Stop current process
		stopBinary()

		// Rebuild
		newContent, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "wolf dev: failed to read file: %v\n", err)
			continue
		}
		_, buildErr := c.Build(string(newContent), filename)
		if buildErr != nil {
			fmt.Fprintf(os.Stderr, "wolf dev: rebuild failed:\n%v\n", buildErr)
			fmt.Printf("wolf dev: watching for more changes...\n")
			continue
		}

		fmt.Printf("wolf dev: rebuilt successfully, restarting...\n")
		startBinary()
	}
}

// runHMRMode is the original HMR path for API/server targets.
// It compiles to a .so, runs wolf_host, and sends SIGUSR1 on any file change.
func runHMRMode(c *compiler.Compiler, cfg *config.WolfConfig, filename, outDir, watchRoot string, isVerbose bool) error {
	hostPath := filepath.Join(outDir, "wolf_host")

	// Force Shared mode for HMR
	cfg.Target.Shared = true

	// Start Observability Dashboard in background
	targetPort := 2006
	if cfg.Server.Port != 0 {
		targetPort = cfg.Server.Port
	}
	dashboard.Start(targetPort)

	// Initial compile
	fmt.Printf("wolf dev: compiling for HMR...\n")
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	if _, err = c.Build(string(content), filename); err != nil {
		fmt.Fprintf(os.Stderr, "Initial build failed:\n%v\n", err)
		os.Exit(1)
	}

	// Start Host Shell
	baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	appSoPath := filepath.Join(outDir, baseName+".so")
	hostCmd := exec.Command(hostPath, filename)
	hostCmd.Stdout = os.Stdout
	hostCmd.Stderr = os.Stderr
	hostCmd.Env = append(os.Environ(), "WOLF_APP_SO_PATH="+appSoPath)
	if err := hostCmd.Start(); err != nil {
		return fmt.Errorf("failed to start host shell: %w", err)
	}

	fmt.Printf("wolf dev: watching %s for changes (HMR mode)...\n", watchRoot)

	// Snapshot initial file tree
	lastMtimes, _ := walkWolfFiles(watchRoot)

	go func() {
		for {
			time.Sleep(200 * time.Millisecond)

			changed, newMtimes, err := hasChanges(watchRoot, lastMtimes)
			if err != nil {
				continue
			}
			if !changed {
				continue
			}
			lastMtimes = newMtimes

			fmt.Printf("\nwolf dev: change detected, recompiling...\n")

			newContent, err := os.ReadFile(filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to read file: %v\n", err)
				continue
			}
			_, err = c.Build(string(newContent), filename)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Rebuild failed:\n%v\n", err)
				continue
			}

			// Signal host shell to hot-reload
			if hostCmd.Process != nil {
				if runtime.GOOS == "windows" {
					hostCmd.Process.Signal(os.Interrupt)
				} else {
					hostCmd.Process.Signal(syscall.SIGUSR1)
				}
			}
		}
	}()

	// Wait for host shell to exit
	if err := hostCmd.Wait(); err != nil {
		return fmt.Errorf("host shell exited with error: %w", err)
	}
	return nil
}

func getModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
