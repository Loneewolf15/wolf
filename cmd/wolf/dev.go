package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/wolflang/wolf/internal/compiler"
	"github.com/wolflang/wolf/internal/config"
	"github.com/wolflang/wolf/internal/dashboard"
)

var devCmd = &cobra.Command{
	Use:   "dev [file]",
	Short: "Start the developer server with Hot Module Replacement (HMR)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filename := args[0]
		if !filepath.IsAbs(filename) {
			cwd, _ := os.Getwd()
			filename = filepath.Join(cwd, filename)
		}

		outDir := filepath.Join(filepath.Dir(filename), "wolf_out")
		hostPath := filepath.Join(outDir, "wolf_host")

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
			// Fallback if somehow still nil
			cfg = &config.WolfConfig{
				Target: config.TargetConfig{Mode: "script", Arch: "native"},
				DB:     config.DBConfig{Driver: "mysql", PoolSize: 10, PoolMinIdle: 2, PoolTimeout: 5000, MaxRetries: 3},
				Server: config.ServerConfig{MaxConcurrent: 1024, MaxRequestSize: 65536, MaxUploads: 8},
				App:    config.AppConfig{Env: "development", Debug: true},
			}
			c.Config = cfg
		}

		// Zero-Config Execution Heuristic
		configPath := filepath.Join(c.ProjectRoot, "wolf.config")
		if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
			content, _ := os.ReadFile(filename)
			source := string(content)
			if strings.Contains(source, "wolf_http_serve") || strings.Contains(source, "route(") {
				cfg.Target.Mode = "api"
			} else {
				cfg.Target.Mode = "script"
			}
		}

		// Force Shared mode for HMR
		cfg.Target.Shared = true

		// Start Observability Dashboard in background
		targetPort := 2006
		if cfg.Server.Port != 0 {
			targetPort = cfg.Server.Port
		}
		dashboard.Start(targetPort)

		// Initial compile
		fmt.Printf("wolf: compiling for HMR...\n")

		isVerbose, _ := cmd.Flags().GetBool("verbose")
		c.Verbose = isVerbose
		c.OutDir = outDir

		content, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read source file: %w", err)
		}

		_, err = c.Build(string(content), filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Initial build failed:\n%v\n", err)
			os.Exit(1)
		}

		// Start Host Shell
		baseName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		appSoPath := filepath.Join(outDir, baseName+".so")
		hostCmd := exec.Command(hostPath, args[0])
		hostCmd.Stdout = os.Stdout
		hostCmd.Stderr = os.Stderr
		hostCmd.Env = append(os.Environ(), "WOLF_APP_SO_PATH="+appSoPath)
		if err := hostCmd.Start(); err != nil {
			return fmt.Errorf("failed to start host shell: %w", err)
		}

		fmt.Printf("wolf dev: watching %s for changes...\n", filename)

		// Simple polling watcher (since fsnotify is not in go.mod)
		lastMod, _ := getModTime(filename)

		go func() {
			for {
				time.Sleep(200 * time.Millisecond)
				mod, err := getModTime(filename)
				if err != nil {
					continue
				}
				if mod.After(lastMod) {
					lastMod = mod
					fmt.Printf("\nwolf dev: change detected, recompiling...\n")

					// Recompile
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

					// Send signal to host shell
					if hostCmd.Process != nil {
						if runtime.GOOS == "windows" {
							hostCmd.Process.Signal(os.Interrupt)
						} else {
							hostCmd.Process.Signal(syscall.SIGUSR1)
						}
					}
				}
			}
		}()

		// Wait for host shell to exit
		err = hostCmd.Wait()
		if err != nil {
			return fmt.Errorf("host shell exited with error: %w", err)
		}

		return nil
	},
}

func getModTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
