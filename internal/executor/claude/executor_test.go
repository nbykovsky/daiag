package claude

import (
	"context"
	"testing"

	"daiag/internal/executor"
	"daiag/internal/runtime"
)

func TestExecutorRunBuildsClaudeCommand(t *testing.T) {
	runner := fakeRunner(func(_ context.Context, cmd executor.Command) (executor.Result, error) {
		if cmd.Name != "claude" {
			t.Fatalf("command name = %q, want claude", cmd.Name)
		}
		requireContains(t, cmd.Args, "--print")
		requirePair(t, cmd.Args, "--model", "sonnet")
		requirePair(t, cmd.Args, "--permission-mode", "bypassPermissions")
		requirePair(t, cmd.Args, "--add-dir", "/tmp/work")
		if got := cmd.Args[len(cmd.Args)-1]; got != "Return JSON only." {
			t.Fatalf("prompt arg = %q, want prompt", got)
		}
		return executor.Result{
			Stdout:   `{"ok":true}`,
			Stderr:   "stderr",
			ExitCode: 0,
		}, nil
	})

	exec := Executor{Runner: runner, CommandName: "claude"}
	resp, err := exec.Run(context.Background(), runtime.TaskRequest{
		Model:   "sonnet",
		Prompt:  "Return JSON only.",
		Workdir: "/tmp/work",
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if resp.Stdout != `{"ok":true}` {
		t.Fatalf("stdout = %q, want JSON output", resp.Stdout)
	}
	if resp.Stderr != "stderr" {
		t.Fatalf("stderr = %q, want %q", resp.Stderr, "stderr")
	}
}

type fakeRunner func(context.Context, executor.Command) (executor.Result, error)

func (f fakeRunner) Run(ctx context.Context, cmd executor.Command) (executor.Result, error) {
	return f(ctx, cmd)
}

func requireContains(t *testing.T, args []string, want string) {
	t.Helper()
	for _, arg := range args {
		if arg == want {
			return
		}
	}
	t.Fatalf("args %v do not contain %q", args, want)
}

func requirePair(t *testing.T, args []string, flag, want string) {
	t.Helper()
	if got := argValue(t, args, flag); got != want {
		t.Fatalf("%s = %q, want %q", flag, got, want)
	}
}

func argValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	t.Fatalf("args %v do not contain flag %q", args, flag)
	return ""
}
