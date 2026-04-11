package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

const usageText = `Usage:
  daiag run --workflow <workflow-id> --workdir <path> [--workflows-lib <dir>] [--input key=value] [--param key=value] [--verbose]
  daiag validate --workflow <workflow-id> [--workflows-lib <dir>]

Commands:
  run       Execute a workflow
  validate  Parse and validate a workflow without executing it
`

type Runner interface {
	Run(context.Context, RunConfig) error
}

type Validator interface {
	Validate(context.Context, ValidateConfig) error
}

type App struct {
	stdout    io.Writer
	stderr    io.Writer
	runner    Runner
	validator Validator
}

type RunConfig struct {
	Workflow     string
	WorkflowsLib string
	Inputs       map[string]string
	Params       map[string]string
	Workdir      string
	Verbose      bool
}

type ValidateConfig struct {
	Workflow     string
	WorkflowsLib string
}

func New(stdout, stderr io.Writer, runner Runner, validator Validator) *App {
	return &App{
		stdout:    stdout,
		stderr:    stderr,
		runner:    runner,
		validator: validator,
	}
}

func (a *App) Run(ctx context.Context, args []string) int {
	if len(args) == 0 {
		a.printUsage(a.stderr)
		return 2
	}

	switch args[0] {
	case "run":
		cfg, err := parseRunArgs(args[1:])
		if err != nil {
			fmt.Fprintf(a.stderr, "error: %v\n\n", err)
			a.printUsage(a.stderr)
			return 2
		}
		if err := a.runner.Run(ctx, cfg); err != nil {
			fmt.Fprintf(a.stderr, "error: %v\n", err)
			return 1
		}
		return 0
	case "validate":
		cfg, err := parseValidateArgs(args[1:])
		if err != nil {
			fmt.Fprintf(a.stderr, "error: %v\n\n", err)
			a.printUsage(a.stderr)
			return 2
		}
		if err := a.validator.Validate(ctx, cfg); err != nil {
			fmt.Fprintf(a.stderr, "error: %v\n", err)
			return 1
		}
		fmt.Fprintf(a.stdout, "workflow %q is valid\n", cfg.Workflow)
		return 0
	case "help", "-h", "--help":
		a.printUsage(a.stdout)
		return 0
	default:
		fmt.Fprintf(a.stderr, "error: unknown command %q\n\n", args[0])
		a.printUsage(a.stderr)
		return 2
	}
}

func (a *App) printUsage(w io.Writer) {
	io.WriteString(w, usageText)
}

func parseRunArgs(args []string) (RunConfig, error) {
	var cfg RunConfig
	var inputs multiFlag
	var params multiFlag

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.Workflow, "workflow", "", "workflow ID")
	fs.StringVar(&cfg.WorkflowsLib, "workflows-lib", "", "workflow library directory")
	fs.StringVar(&cfg.Workdir, "workdir", "", "working directory")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "enable verbose output")
	fs.Var(&inputs, "input", "workflow input in key=value form")
	fs.Var(&params, "param", "workflow parameter in key=value form")

	if err := fs.Parse(args); err != nil {
		return RunConfig{}, err
	}
	if fs.NArg() > 0 {
		return RunConfig{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if cfg.Workflow == "" {
		return RunConfig{}, errors.New("--workflow is required")
	}

	inputMap := make(map[string]string, len(inputs))
	for _, raw := range inputs {
		key, value, err := parseKeyValue("--input", raw)
		if err != nil {
			return RunConfig{}, err
		}
		inputMap[key] = value
	}
	paramMap := make(map[string]string, len(params))
	for _, raw := range params {
		key, value, err := parseKeyValue("--param", raw)
		if err != nil {
			return RunConfig{}, err
		}
		paramMap[key] = value
	}

	merged := cloneStringMap(inputMap)
	for key, value := range paramMap {
		if existing, ok := inputMap[key]; ok && existing != value {
			return RunConfig{}, fmt.Errorf("conflicting workflow input %q from --input and --param", key)
		}
		merged[key] = value
	}
	if cfg.Workdir == "" {
		return RunConfig{}, errors.New("--workdir is required")
	}
	cfg.Inputs = cloneStringMap(merged)
	cfg.Params = cloneStringMap(merged)

	return cfg, nil
}

func parseValidateArgs(args []string) (ValidateConfig, error) {
	var cfg ValidateConfig
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfg.Workflow, "workflow", "", "workflow ID")
	fs.StringVar(&cfg.WorkflowsLib, "workflows-lib", "", "workflow library directory")
	if err := fs.Parse(args); err != nil {
		return ValidateConfig{}, err
	}
	if fs.NArg() > 0 {
		return ValidateConfig{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if cfg.Workflow == "" {
		return ValidateConfig{}, errors.New("--workflow is required")
	}
	return cfg, nil
}

type multiFlag []string

func (m *multiFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *multiFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
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
