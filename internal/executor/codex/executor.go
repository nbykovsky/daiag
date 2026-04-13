package codex

import (
	"context"
	"fmt"
	"os"

	"daiag/internal/executor"
	"daiag/internal/runtime"
)

type Executor struct {
	Runner      executor.Runner
	CommandName string
}

func New() Executor {
	return Executor{
		Runner:      executor.OSRunner{},
		CommandName: "codex",
	}
}

func (e Executor) Run(ctx context.Context, req runtime.TaskRequest) (runtime.TaskResponse, error) {
	outputFile, err := os.CreateTemp("", "daiag-codex-output-*.txt")
	if err != nil {
		return runtime.TaskResponse{}, fmt.Errorf("create codex output file: %w", err)
	}
	outputPath := outputFile.Name()
	if err := outputFile.Close(); err != nil {
		return runtime.TaskResponse{}, fmt.Errorf("close codex output file: %w", err)
	}
	defer os.Remove(outputPath)

	command := executor.Command{
		Name: e.commandName(),
		Args: []string{
			"exec",
			"--skip-git-repo-check",
			"--sandbox", "workspace-write",
			"--full-auto",
			"--color", "never",
			"--model", req.Model,
			"-C", req.ProjectDir,
			"--output-last-message", outputPath,
			"-",
		},
		Dir:   req.ProjectDir,
		Stdin: req.Prompt,
	}

	result, err := e.runner().Run(ctx, command)
	if err != nil {
		return runtime.TaskResponse{}, err
	}

	finalMessage, readErr := os.ReadFile(outputPath)
	if readErr != nil && result.ExitCode == 0 {
		return runtime.TaskResponse{}, fmt.Errorf("read codex output file: %w", readErr)
	}

	return runtime.TaskResponse{
		Stdout:   string(finalMessage),
		Stderr:   result.Stderr,
		ExitCode: result.ExitCode,
	}, nil
}

func (e Executor) runner() executor.Runner {
	if e.Runner != nil {
		return e.Runner
	}
	return executor.OSRunner{}
}

func (e Executor) commandName() string {
	if e.CommandName != "" {
		return e.CommandName
	}
	return "codex"
}
