package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wolflang/wolf/internal/compiler"
	"github.com/wolflang/wolf/internal/pythonenv"
	"github.com/wolflang/wolf/internal/scaffold"
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

	buildCmd := &cobra.Command{
		Use:   "build [file]",
		Short: "Compile a Wolf source file to a native binary",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("cannot read file: %w", err)
			}

			projectRoot, _ := os.Getwd()
			c, err := compiler.NewWithConfig(projectRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wolf: config warning: %v\n", err)
				c = compiler.New()
			}
			c.Verbose = verbose
			c.StrictMode = strict

			result, err := c.Build(string(source), args[0])
			if err != nil {
				for _, e := range result.Errors {
					fmt.Fprintln(os.Stderr, e)
				}
				return err
			}

			fmt.Printf("wolf: built → %s\n", result.OutputPath)
			return nil
		},
	}

	runCmd := &cobra.Command{
		Use:   "run [file]",
		Short: "Compile and run a Wolf source file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("cannot read file: %w", err)
			}

			projectRoot, _ := os.Getwd()
			c, err := compiler.NewWithConfig(projectRoot)
			if err != nil {
				fmt.Fprintf(os.Stderr, "wolf: config warning: %v\n", err)
				c = compiler.New()
			}
			c.Verbose = verbose
			c.StrictMode = strict

			return c.Run(string(source), args[0])
		},
	}

	fmtCmd := &cobra.Command{
		Use:   "fmt [file]",
		Short: "Format a Wolf source file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("wolf: formatting %s...\n", args[0])
			// TODO: implement formatter
			return nil
		},
	}

	testCmd := &cobra.Command{
		Use:   "test [file|dir]",
		Short: "Run Wolf test files",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("wolf: running tests...")
			// TODO: implement test runner
			return nil
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
	newCmd := &cobra.Command{
		Use:   "new [name]",
		Short: "Create a new Wolf project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			fmt.Printf("wolf: creating project '%s'...\n", name)
			if err := scaffold.Project(name); err != nil {
				return err
			}
			fmt.Printf("wolf: project '%s' created ✓\n", name)
			fmt.Printf("  cd %s && wolf run src/main.wolf\n", name)
			return nil
		},
	}

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

	rootCmd.AddCommand(buildCmd, runCmd, fmtCmd, testCmd, pythonCmd, newCmd, generateCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
