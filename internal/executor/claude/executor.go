package claude

import (
	"context"

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
		CommandName: "claude",
	}
}

func (e Executor) Run(ctx context.Context, req runtime.TaskRequest) (runtime.TaskResponse, error) {
	command := executor.Command{
		Name: e.commandName(),
		Args: []string{
			"--print",
			"--model", req.Model,
			"--permission-mode", "bypassPermissions",
			"--add-dir", req.ProjectDir,
		},
		Dir:   req.ProjectDir,
		Stdin: req.Prompt,
	}

	result, err := e.runner().Run(ctx, command)
	if err != nil {
		return runtime.TaskResponse{}, err
	}

	return runtime.TaskResponse{
		Stdout:   result.Stdout,
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
	return "claude"
}
