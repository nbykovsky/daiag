package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

type Runner interface {
	Run(context.Context, RunConfig) error
}

type Validator interface {
	Validate(context.Context, ValidateConfig) error
}

type Bootstrapper interface {
	Bootstrap(context.Context, BootstrapConfig) error
}

type App struct {
	stdout       io.Writer
	stderr       io.Writer
	runner       Runner
	validator    Validator
	bootstrapper Bootstrapper
}

type RunConfig struct {
	Workflow     string
	WorkflowsLib string
	Inputs       map[string]string
	ProjectDir   string
	RunDir       string
	Verbose      bool
}

type ValidateConfig struct {
	Workflow     string
	WorkflowsLib string
	Inputs       map[string]string
	ProjectDir   string
}

type BootstrapConfig struct {
	Workflow        string
	Description     string
	DescriptionFile string
	ProjectDir      string
	RunDir          string
	WorkflowsLib    string
	Verbose         bool
}

func New(stdout, stderr io.Writer, runner Runner, validator Validator) *App {
	bootstrapper, _ := runner.(Bootstrapper)
	return &App{
		stdout:       stdout,
		stderr:       stderr,
		runner:       runner,
		validator:    validator,
		bootstrapper: bootstrapper,
	}
}

// usageError marks errors that should produce exit code 2 (bad usage / flag parse failure).
type usageError struct{ cause error }

func (e usageError) Error() string { return e.cause.Error() }
func (e usageError) Unwrap() error { return e.cause }

func (a *App) Run(ctx context.Context, args []string) int {
	root := a.buildRoot(ctx)
	root.SetArgs(args)
	root.SetOut(a.stdout)
	root.SetErr(a.stderr)

	err := root.Execute()
	if err == nil {
		return 0
	}

	var ue usageError
	if errors.As(err, &ue) {
		fmt.Fprintf(a.stderr, "error: %v\n\n", err)
		fmt.Fprint(a.stderr, root.UsageString())
		return 2
	}
	fmt.Fprintf(a.stderr, "error: %v\n", err)
	return 1
}

func (a *App) buildRoot(ctx context.Context) *cobra.Command {
	root := &cobra.Command{
		Use:           "daiag",
		Short:         "Orchestrate AI agents through Starlark-defined workflows",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return usageError{err}
	})
	root.AddCommand(a.newRunCmd(ctx))
	root.AddCommand(a.newValidateCmd(ctx))
	root.AddCommand(a.newBootstrapCmd(ctx))
	return root
}

func (a *App) newRunCmd(ctx context.Context) *cobra.Command {
	var cfg RunConfig
	var rawInputs []string

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Execute a workflow",
		Long: `Execute a workflow by ID. Loads the workflow definition from the workflows
library, resolves inputs, and runs each task sequentially.

Flags:
  --workflow       (required) Workflow ID, e.g. "poem" or "code-review".
  --input          Repeatable. Workflow input as key=value, e.g. --input feature=poem.
  --projectdir     Project root. Defaults to the nearest ancestor containing .daiag/.
  --run-dir        Directory to store run artifacts. Defaults to .daiag/runs/<id>/<timestamp>/.
  --workflows-lib  Directory containing workflow definitions. Defaults to <projectdir>/.daiag/workflows/.
  --verbose        Print detailed progress output.

Examples:
  # Run the "poem" workflow with two inputs
  daiag run --workflow poem --input feature=poem --input mode=fast

  # Run with an explicit project and run directory
  daiag run --workflow code-review --projectdir /path/to/repo --run-dir /tmp/run1

  # Run with a shared workflows library
  daiag run --workflow poem --workflows-lib /shared/workflows --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageError{fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))}
			}
			if cfg.Workflow == "" {
				return usageError{errors.New("--workflow is required")}
			}
			inputs, err := parseInputs(rawInputs)
			if err != nil {
				return usageError{err}
			}
			cfg.Inputs = inputs
			return a.runner.Run(ctx, cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.Workflow, "workflow", "", "workflow ID")
	cmd.Flags().StringVar(&cfg.ProjectDir, "projectdir", "", "project directory")
	cmd.Flags().StringVar(&cfg.RunDir, "run-dir", "", "run artifact directory")
	cmd.Flags().StringVar(&cfg.WorkflowsLib, "workflows-lib", "", "workflow library directory")
	cmd.Flags().BoolVar(&cfg.Verbose, "verbose", false, "enable verbose output")
	cmd.Flags().StringArrayVar(&rawInputs, "input", nil, "workflow input as key=value (repeatable)")

	return cmd
}

func (a *App) newValidateCmd(ctx context.Context) *cobra.Command {
	var cfg ValidateConfig
	var rawInputs []string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Parse and validate a workflow without executing it",
		Long: `Parse and validate a workflow definition without executing it.
Useful for CI checks and authoring feedback.

Flags:
  --workflow       (required) Workflow ID to validate.
  --input          Repeatable. Workflow input as key=value (values are not executed).
  --projectdir     Project root. Defaults to the nearest ancestor containing .daiag/.
  --workflows-lib  Directory containing workflow definitions. Defaults to <projectdir>/.daiag/workflows/.

Examples:
  # Validate the "poem" workflow
  daiag validate --workflow poem

  # Validate with explicit inputs and project directory
  daiag validate --workflow code-review --projectdir /path/to/repo --input ticket=PROJ-123`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageError{fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))}
			}
			if cfg.Workflow == "" {
				return usageError{errors.New("--workflow is required")}
			}
			inputs, err := parseInputs(rawInputs)
			if err != nil {
				return usageError{err}
			}
			cfg.Inputs = inputs
			if err := a.validator.Validate(ctx, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "workflow %q is valid\n", cfg.Workflow)
			return nil
		},
	}

	cmd.Flags().StringVar(&cfg.Workflow, "workflow", "", "workflow ID")
	cmd.Flags().StringVar(&cfg.ProjectDir, "projectdir", "", "project directory")
	cmd.Flags().StringVar(&cfg.WorkflowsLib, "workflows-lib", "", "workflow library directory")
	cmd.Flags().StringArrayVar(&rawInputs, "input", nil, "workflow input as key=value (repeatable)")

	return cmd
}

func (a *App) newBootstrapCmd(ctx context.Context) *cobra.Command {
	cfg := BootstrapConfig{Workflow: "workflow_bootstrapper"}
	var rawInputs []string

	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Generate a workflow through the workflow catalog bootstrapper",
		Long: `Generate a new workflow using the workflow catalog bootstrapper.
Exactly one of --description or --description-file must be provided.

Flags:
  --description       Natural-language description of the workflow to generate.
  --description-file  Path to a file containing the workflow description.
  --workflow          Bootstrap workflow ID (default: workflow_bootstrapper).
  --projectdir        Project root. Defaults to the nearest ancestor containing .daiag/.
  --run-dir           Directory to store run artifacts. Defaults to .daiag/runs/<id>/<timestamp>/.
  --workflows-lib     Directory containing workflow definitions. Defaults to <projectdir>/.daiag/workflows/.
  --verbose           Print detailed progress output.

Examples:
  # Bootstrap a workflow from an inline description
  daiag bootstrap --description "summarise a pull request and post a comment"

  # Bootstrap from a requirements file
  daiag bootstrap --description-file requirements.md --projectdir /path/to/repo

  # Use a custom bootstrap workflow with verbose output
  daiag bootstrap --description "review code" --workflow custom_bootstrap --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usageError{fmt.Errorf("unexpected arguments: %s", strings.Join(args, " "))}
			}
			if cfg.Workflow == "" {
				return usageError{errors.New("--workflow must not be empty")}
			}
			hasDesc := cfg.Description != ""
			hasFile := cfg.DescriptionFile != ""
			if hasDesc == hasFile {
				return usageError{errors.New("exactly one of --description or --description-file is required")}
			}
			if a.bootstrapper == nil {
				return errors.New("bootstrap command is unavailable")
			}
			_ = rawInputs // bootstrap does not accept --input; kept for symmetry
			return a.bootstrapper.Bootstrap(ctx, cfg)
		},
	}

	cmd.Flags().StringVar(&cfg.Workflow, "workflow", cfg.Workflow, "bootstrap workflow ID")
	cmd.Flags().StringVar(&cfg.Description, "description", "", "workflow request")
	cmd.Flags().StringVar(&cfg.DescriptionFile, "description-file", "", "workflow request file")
	cmd.Flags().StringVar(&cfg.ProjectDir, "projectdir", "", "project directory")
	cmd.Flags().StringVar(&cfg.RunDir, "run-dir", "", "run artifact directory")
	cmd.Flags().StringVar(&cfg.WorkflowsLib, "workflows-lib", "", "workflow library directory")
	cmd.Flags().BoolVar(&cfg.Verbose, "verbose", false, "enable verbose output")
	cmd.Flags().StringArrayVar(&rawInputs, "input", nil, "workflow input as key=value (repeatable)")

	return cmd
}

func parseInputs(raw []string) (map[string]string, error) {
	m := make(map[string]string, len(raw))
	for _, r := range raw {
		k, v, err := parseKeyValue("--input", r)
		if err != nil {
			return nil, err
		}
		m[k] = v
	}
	return cloneStringMap(m), nil
}

func parseKeyValue(flagName string, raw string) (string, string, error) {
	key, value, ok := strings.Cut(raw, "=")
	if !ok || key == "" || value == "" {
		return "", "", fmt.Errorf("invalid %s %q, expected key=value", flagName, raw)
	}
	return key, value, nil
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}
