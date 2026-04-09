package executor

import (
	"bytes"
	"context"
	"os/exec"
)

type Command struct {
	Name  string
	Args  []string
	Dir   string
	Stdin string
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Runner interface {
	Run(context.Context, Command) (Result, error)
}

type OSRunner struct{}

func (OSRunner) Run(ctx context.Context, command Command) (Result, error) {
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = command.Dir
	cmd.Stdin = bytes.NewBufferString(command.Stdin)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}
	if err == nil {
		result.ExitCode = 0
		return result, nil
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		return result, nil
	}

	return Result{}, err
}
