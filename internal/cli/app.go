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
  daiag run --workflow <path> [--param key=value] [--workdir <path>] [--verbose]

Commands:
  run     Execute a workflow
`

type Runner interface {
	Run(context.Context, RunConfig) error
}

type App struct {
	stdout io.Writer
	stderr io.Writer
	runner Runner
}

type RunConfig struct {
	Workflow string
	Params   map[string]string
	Workdir  string
	Verbose  bool
}

func New(stdout, stderr io.Writer, runner Runner) *App {
	return &App{
		stdout: stdout,
		stderr: stderr,
		runner: runner,
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
	var params multiFlag

	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&cfg.Workflow, "workflow", "", "path to workflow file")
	fs.StringVar(&cfg.Workdir, "workdir", "", "working directory")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "enable verbose output")
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

	cfg.Params = make(map[string]string, len(params))
	for _, raw := range params {
		key, value, err := parseKeyValue(raw)
		if err != nil {
			return RunConfig{}, err
		}
		cfg.Params[key] = value
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

func parseKeyValue(raw string) (string, string, error) {
	key, value, ok := strings.Cut(raw, "=")
	if !ok || key == "" || value == "" {
		return "", "", fmt.Errorf("invalid --param %q, expected key=value", raw)
	}
	return key, value, nil
}
