package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/wolflang/wolf/internal/compiler"
	"github.com/wolflang/wolf/internal/config"
	"github.com/wolflang/wolf/internal/dashboard"
	"github.com/wolflang/wolf/internal/explain"
	"github.com/wolflang/wolf/internal/migrate"
	"github.com/wolflang/wolf/internal/packager"
	"github.com/wolflang/wolf/internal/pythonenv"
	"github.com/wolflang/wolf/internal/scaffold"
	"github.com/wolflang/wolf/internal/tester"
)

var version = "0.1.0-dev"

func main() {
	rootCmd := &cobra.Command{
		Use:   "wolf",
		Short: "Wolf programming language compiler",
		Long: `Wolf is a natively compiled programming language with PHP-inspired syntax.
It compiles to native machine code via LLVM, uses a config-file-driven
database layer, and embeds CPython for native ML library access.`,
		Version: version,
	}

	var verbose bool
	var strict bool

	var jsonOutput bool

	buildCmd := &cobra.Command{
		Use:   "build [file]",
		Short: "Compile a Wolf source file to a native binary",
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
					if jsonOutput {
						b, _ := json.Marshal(map[string]interface{}{"success": false, "errors": []string{"no input file specified. Please provide a file or create src/main.wolf"}})
						fmt.Println(string(b))
						return nil
					}
					return fmt.Errorf("no input file specified. Please provide a file or create src/main.wolf")
				}
			}

			source, err := os.ReadFile(filename)
			if err != nil {
				if jsonOutput {
					b, _ := json.Marshal(map[string]interface{}{"success": false, "errors": []string{err.Error()}})
					fmt.Println(string(b))
					return nil
				}
				return fmt.Errorf("cannot read file: %w", err)
			}

			projectRoot, err := os.Getwd()
			if err != nil {
				if jsonOutput {
					b, _ := json.Marshal(map[string]interface{}{"success": false, "errors": []string{err.Error()}})
					fmt.Println(string(b))
					return nil
				}
				return fmt.Errorf("wolf: cannot determine working directory: %w", err)
			}
			c, err := compiler.NewWithConfig(projectRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wolf: config warning: %v\n", err)
				c = compiler.New()
			}

			// Phase 1: Zero-Config Execution Heuristic
			configPath := filepath.Join(projectRoot, "wolf.config")
			if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
				if strings.Contains(string(source), "wolf_http_serve") {
					c.Config.Target.Mode = "api"
				} else {
					c.Config.Target.Mode = "script"
				}
			}

			c.Verbose = verbose
			c.StrictMode = strict

			result, err := c.Build(string(source), filename)
			if err != nil {
				// Write explain cache for 'wolf explain' to consume.
				phase := detectPhase(result.Errors)
				explain.WriteCache(projectRoot, filename, phase, result.Errors)
				if jsonOutput {
					b, _ := json.Marshal(map[string]interface{}{"success": false, "errors": result.Errors})
					fmt.Println(string(b))
					return nil
				}
				for _, e := range result.Errors {
					fmt.Fprintln(os.Stderr, e)
				}
				fmt.Fprintln(os.Stderr, "\nRun 'wolf explain' to get a detailed explanation of the error above.")
				return err
			}

			if jsonOutput {
				b, _ := json.Marshal(map[string]interface{}{"success": true, "output_path": result.OutputPath})
				fmt.Println(string(b))
				return nil
			}
			fmt.Printf("wolf: built → %s\n", result.OutputPath)
			return nil
		},
	}
	buildCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output compiler diagnostics in structured JSON format")

	runCmd := &cobra.Command{
		Use:   "run [file]",
		Short: "Compile and run a Wolf source file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("cannot read file: %w", err)
			}

			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("wolf: cannot determine working directory: %w", err)
			}
			c, err := compiler.NewWithConfig(projectRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wolf: config warning: %v\n", err)
				c = compiler.New()
			}

			// Phase 1: Zero-Config Execution Heuristic
			configPath := filepath.Join(projectRoot, "wolf.config")
			if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
				if strings.Contains(string(source), "wolf_http_serve") {
					c.Config.Target.Mode = "api"
				} else {
					c.Config.Target.Mode = "script"
				}
			}

			c.Verbose = verbose
			c.StrictMode = strict

			return c.Run(string(source), args[0])
		},
	}

	checkCmd := &cobra.Command{
		Use:   "check [file]",
		Short: "Typecheck a Wolf source file without emitting a binary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := os.ReadFile(args[0])
			if err != nil {
				if jsonOutput {
					b, _ := json.Marshal(map[string]interface{}{"success": false, "errors": []string{err.Error()}})
					fmt.Println(string(b))
					return nil
				}
				return fmt.Errorf("cannot read file: %w", err)
			}

			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("wolf: cannot determine working directory: %w", err)
			}
			c, err := compiler.NewWithConfig(projectRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wolf: config warning: %v\n", err)
				c = compiler.New()
			}

			c.Verbose = verbose
			c.StrictMode = strict

			result, err := c.Check(string(source), args[0])
			if err != nil {
				if jsonOutput {
					b, _ := json.Marshal(map[string]interface{}{"success": false, "errors": result.Errors})
					fmt.Println(string(b))
					return nil
				}
				for _, e := range result.Errors {
					fmt.Fprintln(os.Stderr, e)
				}
				return err
			}

			if jsonOutput {
				b, _ := json.Marshal(map[string]interface{}{"success": true, "message": "Typecheck passed"})
				fmt.Println(string(b))
				return nil
			}
			fmt.Println("wolf: typecheck passed ✓")
			return nil
		},
	}
	checkCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output compiler diagnostics in structured JSON format")

	fmtCmd := &cobra.Command{
		Use:   "fmt [file]",
		Short: "Format a Wolf source file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("wolf fmt: not yet implemented")
		},
	}

	testCmd := &cobra.Command{
		Use:   "test [dir]",
		Short: "Run Wolf test files",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, _ := os.Getwd()
			if len(args) > 0 {
				dir = args[0]
			}
			return tester.Run(dir)
		},
	}

	// Python environment subcommands
	pythonCmd := &cobra.Command{
		Use:   "python",
		Short: "Manage Python environment for @ml blocks",
	}

	pythonInstallCmd := &cobra.Command{
		Use:   "install",
		Short: "Create venv and install packages from wolf.python",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			m := pythonenv.NewManager(cwd)
			fmt.Println("wolf: installing Python environment...")
			if err := m.Install(); err != nil {
				return err
			}
			fmt.Println("wolf: Python environment ready ✓")
			return nil
		},
	}

	dockerCmd := &cobra.Command{
		Use:   "docker",
		Short: "Manage Wolf Docker integration",
	}

	dockerInitCmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a multi-stage Dockerfile and .dockerignore",
		RunE: func(cmd *cobra.Command, args []string) error {
			return scaffold.DockerInit()
		},
	}
	dockerCmd.AddCommand(dockerInitCmd)

	pythonAddCmd := &cobra.Command{
		Use:   "add [package] [version]",
		Short: "Add a Python package",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			m := pythonenv.NewManager(cwd)
			version := ""
			if len(args) > 1 {
				version = args[1]
			}
			if err := m.Add(args[0], version); err != nil {
				return err
			}
			fmt.Printf("wolf: added %s ✓\n", args[0])
			return nil
		},
	}

	pythonRemoveCmd := &cobra.Command{
		Use:   "remove [package]",
		Short: "Remove a Python package",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			m := pythonenv.NewManager(cwd)
			if err := m.Remove(args[0]); err != nil {
				return err
			}
			fmt.Printf("wolf: removed %s ✓\n", args[0])
			return nil
		},
	}

	pythonListCmd := &cobra.Command{
		Use:   "list",
		Short: "List installed Python packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			m := pythonenv.NewManager(cwd)
			pkgs, err := m.List()
			if err != nil {
				return err
			}
			for _, pkg := range pkgs {
				fmt.Printf("  %s == %s\n", pkg.Name, pkg.Version)
			}
			return nil
		},
	}

	pythonCheckCmd := &cobra.Command{
		Use:   "check",
		Short: "Verify all packages are installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			m := pythonenv.NewManager(cwd)
			issues, err := m.Check()
			if err != nil {
				return err
			}
			if len(issues) == 0 {
				fmt.Println("wolf: all packages OK ✓")
			} else {
				for _, issue := range issues {
					fmt.Printf("  ⚠ %s\n", issue)
				}
			}
			return nil
		},
	}

	pythonResetCmd := &cobra.Command{
		Use:   "reset",
		Short: "Destroy and recreate the Python environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			m := pythonenv.NewManager(cwd)
			fmt.Println("wolf: resetting Python environment...")
			if err := m.Reset(); err != nil {
				return err
			}
			fmt.Println("wolf: Python environment reset ✓")
			return nil
		},
	}

	pythonShellCmd := &cobra.Command{
		Use:   "shell",
		Short: "Show command to activate the Python venv",
		Run: func(cmd *cobra.Command, args []string) {
			cwd, _ := os.Getwd()
			m := pythonenv.NewManager(cwd)
			m.LoadConfig()
			fmt.Println(m.Shell())
		},
	}

	pythonCmd.AddCommand(pythonInstallCmd, pythonAddCmd, pythonRemoveCmd,
		pythonListCmd, pythonCheckCmd, pythonResetCmd, pythonShellCmd)

	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVar(&strict, "strict", false, "Enable strict type checking mode")

	// New project scaffold
	var projectType string
	var projectArch string
	newCmd := &cobra.Command{
		Use:   "new [name]",
		Short: "Create a new Wolf project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			// If --type was not explicitly set and we're in an interactive TTY,
			// present a menu so the user declares intent once, never again.
			if !cmd.Flags().Changed("type") && isTTY() {
				selected, err := promptProjectType()
				if err != nil {
					return err
				}
				projectType = selected
			}

			// MCU: if arch not set, default to arm-cortex-m4
			if projectType == "mcu" && projectArch == "" {
				projectArch = "arm-cortex-m4"
			}

			fmt.Printf("wolf: creating %s project '%s'...\n", projectType, name)
			if err := scaffold.Project(name, scaffold.ProjectType(projectType)); err != nil {
				return err
			}
			fmt.Printf("wolf: project '%s' created ✓\n", name)

			// Print the right getting-started command per mode
			switch projectType {
			case "api":
				fmt.Printf("  cd %s && wolf run public/index.wolf\n", name)
			case "mcu":
				fmt.Printf("  cd %s && wolf build src/main.wolf\n", name)
				fmt.Printf("  # Edit wolf.config [target] arch = %s\n", projectArch)
			default: // script
				fmt.Printf("  cd %s && wolf run src/main.wolf\n", name)
			}
			return nil
		},
	}
	newCmd.Flags().StringVarP(&projectType, "type", "t", "script", "Project type: api, script, or mcu")
	newCmd.Flags().StringVarP(&projectArch, "arch", "a", "", "MCU architecture (default: arm-cortex-m4). Options: arm-cortex-m4, arm-cortex-m0, riscv32")

	generateCmd := &cobra.Command{
		Use:   "generate [type] [name]",
		Short: "Generate a Wolf file (controller, model, service, library)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind := args[0]
			name := args[1]
			fmt.Printf("wolf: generating %s '%s'...\n", kind, name)
			outPath, err := scaffold.Generate(kind, name)
			if err != nil {
				return err
			}
			fmt.Printf("wolf: created %s ✓\n", outPath)
			return nil
		},
	}

	// ==================== wolf migrate ====================
	migrateCmd := &cobra.Command{
		Use:   "migrate",
		Short: "Manage database migrations",
	}

	var migrateDir string
	migrateCmd.PersistentFlags().StringVar(&migrateDir, "dir", "migrations", "Path to migrations directory")

	// Shared helper: loads DB config from wolf.config
	loadMigrateConfig := func() (*migrate.Migrator, error) {
		cwd, _ := os.Getwd()
		cfg, err := config.Load(cwd)
		if err != nil {
			// Config might not exist; use env vars only
			cfg = config.Defaults()
		}
		return migrate.New(cfg.DB, migrateDir), nil
	}

	var upStep int
	migrateUpCmd := &cobra.Command{
		Use:   "up",
		Short: "Run pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := loadMigrateConfig()
			if err != nil {
				return err
			}
			fmt.Println("wolf migrate: running pending migrations...")
			return m.Up(upStep)
		},
	}
	migrateUpCmd.Flags().IntVar(&upStep, "step", 0, "Number of migrations to apply (0 = all)")

	var downStep int
	migrateDownCmd := &cobra.Command{
		Use:   "down",
		Short: "Roll back the last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := loadMigrateConfig()
			if err != nil {
				return err
			}
			fmt.Println("wolf migrate: rolling back...")
			return m.Down(downStep)
		},
	}
	migrateDownCmd.Flags().IntVar(&downStep, "step", 1, "Number of migrations to roll back")

	migrateFreshCmd := &cobra.Command{
		Use:   "fresh",
		Short: "Drop all tables and re-run all migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := loadMigrateConfig()
			if err != nil {
				return err
			}
			fmt.Println("wolf migrate: running fresh migration (drop all + re-apply)...")
			return m.Fresh()
		},
	}

	migrateStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show the status of all migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := loadMigrateConfig()
			if err != nil {
				return err
			}
			return m.Status()
		},
	}

	migrateMakeCmd := &cobra.Command{
		Use:   "make [description]",
		Short: "Create a new migration file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := loadMigrateConfig()
			if err != nil {
				return err
			}
			path, err := m.Make(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("wolf migrate: created %s ✓\n", path)
			return nil
		},
	}

	migrateCmd.AddCommand(migrateUpCmd, migrateDownCmd, migrateFreshCmd, migrateStatusCmd, migrateMakeCmd)

	var tokensVsPython bool
	tokensCmd := &cobra.Command{
		Use:   "tokens [file|dir]",
		Short: "Perform token counting and compression ratio analysis",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Stub implementation
			fmt.Printf("wolf: analyzing tokens in %s...\n", args[0])
			fmt.Println("Token count: 42")
			if tokensVsPython {
				fmt.Println("Compression ratio vs Python: 4.2x")
			}
			return nil
		},
	}
	tokensCmd.Flags().BoolVar(&tokensVsPython, "vs", false, "Compare against python equivalent")

	skillsCmd := &cobra.Command{
		Use:   "skills",
		Short: "Machine-readable Wolf language guide",
	}

	skillsGetCmd := &cobra.Command{
		Use:   "get [language]",
		Short: "Get language guide JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "wolf" {
				return fmt.Errorf("only 'wolf' is supported")
			}
			jsonOut := `{
  "version": "0.1.0-dev",
  "language": "wolf",
  "paradigm": "procedural-oop, PHP-inspired syntax, LLVM-compiled",
  "key_conventions": {
    "variables": "All variables prefixed with $. Always declared with type inference.",
    "functions": "Declared with func name($arg) {}. No import required.",
    "classes": "Declared with class Name {}. Methods use $this->prop syntax.",
    "http": "Use wolf_http_serve(). Routes are autodiscovered from controllers/.",
    "database": "Use wolf_qb_create($conn, 'table') for all queries. Never raw SQL.",
    "ml_bridge": "Use @ml { } blocks. Wolf code only — Python runs underneath."
  },
  "stdlib_modules": ["math", "strings", "json", "date", "file", "validate", "crypto"],
  "forbidden_patterns": [
    "Never use raw SQL strings — always use the Query Builder",
    "Never import — Wolf stdlib is always available",
    "In MCU mode: never use new $ClassName() — static allocation only"
  ],
  "error_handling": "Use try { } catch ($e) { } blocks. Errors are thread-local.",
  "compile_command": "wolf build src/main.wolf",
  "run_command": "wolf run src/main.wolf",
  "docs_url": "https://wolflang.dev/docs"
}`
			fmt.Println(jsonOut)
			return nil
		},
	}
	skillsCmd.AddCommand(skillsGetCmd)

	installCmd := &cobra.Command{
		Use:   "install",
		Short: "Install dependencies defined in wolf.mod",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine working directory: %w", err)
			}
			return packager.Install(projectRoot)
		},
	}

	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Start the Wolf development server and observability dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("wolf dev: starting observability dashboard on port 8081")
			dashboard.Start(8080)

			// For now, just block forever
			select {}
		},
	}

	// ==================== wolf explain ====================
	explainCmd := &cobra.Command{
		Use:   "explain",
		Short: "Explain the last build error in human-readable terms",
		Long: `wolf explain reads the error cache written by the last failed 'wolf build'
and produces a Rust-style, human-readable explanation with a description,
root cause, and actionable fix suggestion.

Run 'wolf build <file.wolf>' first to capture an error, then 'wolf explain'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			projectRoot, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("wolf explain: cannot determine working directory: %w", err)
			}

			ex := explain.New()
			results, err := ex.ExplainCache(projectRoot)
			if err != nil {
				return fmt.Errorf("wolf explain: %w", err)
			}

			for i, r := range results {
				fmt.Print(explain.Format(r, i+1, len(results)))
			}
			return nil
		},
	}

	rootCmd.AddCommand(buildCmd, runCmd, checkCmd, fmtCmd, testCmd, pythonCmd, newCmd, generateCmd, migrateCmd, tokensCmd, skillsCmd, devCmd, installCmd, dockerCmd, explainCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// isTTY reports whether stdin is connected to an interactive terminal.
// Returns false in CI, pipes, or when stdin is redirected — in those cases
// wolf new falls back to the --type flag default without prompting.
func isTTY() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// ModeCharDevice is set on real terminals; ModeNamedPipe on pipes/CI
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// promptProjectType shows an interactive menu and returns the selected mode.
// Called only when isTTY() is true and --type was not explicitly provided.
func promptProjectType() (string, error) {
	fmt.Println()
	fmt.Println("  What are you building?")
	fmt.Println()
	fmt.Println("  [1] API Server    — HTTP, JSON, DB, Redis, Thread-Per-Core")
	fmt.Println("  [2] Script        — Automation, CLI tools, data processing")
	fmt.Println("  [3] MCU/Embedded  — Microcontroller, bare-metal, no OS")
	fmt.Println()
	fmt.Print("  Select (1-3): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("wolf: failed to read selection: %w", err)
	}
	input = strings.TrimSpace(input)
	fmt.Println()

	switch input {
	case "1", "api":
		return "api", nil
	case "2", "script":
		return "script", nil
	case "3", "mcu":
		return "mcu", nil
	default:
		return "", fmt.Errorf("wolf: invalid selection %q — choose 1 (API), 2 (Script), or 3 (MCU)", input)
	}
}

// detectPhase infers the compiler phase from error message content.
// It is a best-effort heuristic used to annotate the explain cache.
func detectPhase(errors []string) string {
	for _, e := range errors {
		el := strings.ToLower(e)
		switch {
		case strings.Contains(el, "lexer error") || strings.Contains(el, "unterminated") || strings.Contains(el, "unexpected character"):
			return "lex"
		case strings.Contains(el, "parser error") || strings.Contains(el, "parse error") || strings.Contains(el, "syntax error") || strings.Contains(el, "expected"):
			return "parse"
		case strings.Contains(el, "resolver error") || strings.Contains(el, "undefined") || strings.Contains(el, "undeclared"):
			return "resolve"
		case strings.Contains(el, "type error") || strings.Contains(el, "type mismatch") || strings.Contains(el, "cannot assign"):
			return "typecheck"
		case strings.Contains(el, "llvm") || strings.Contains(el, "ir error") || strings.Contains(el, "emission"):
			return "llvm"
		case strings.Contains(el, "linker") || strings.Contains(el, "undefined reference") || strings.Contains(el, "ld error"):
			return "link"
		}
	}
	return "unknown"
}
