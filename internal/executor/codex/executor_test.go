package codex

import (
	"context"
	"os"
	"testing"

	"daiag/internal/executor"
	"daiag/internal/runtime"
)

func TestExecutorRunBuildsCodexCommand(t *testing.T) {
	runner := &fakeRunner{
		t: t,
		run: func(_ context.Context, cmd executor.Command) (executor.Result, error) {
			if cmd.Name != "codex" {
				t.Fatalf("command name = %q, want codex", cmd.Name)
			}
			requireContains(t, cmd.Args, "exec")
			requirePair(t, cmd.Args, "--model", "gpt-5.4")
			requirePair(t, cmd.Args, "-C", "/tmp/work")
			requireContains(t, cmd.Args, "--skip-git-repo-check")
			requireContains(t, cmd.Args, "--full-auto")

			outputPath := argValue(t, cmd.Args, "--output-last-message")
			if err := os.WriteFile(outputPath, []byte(`{"ok":true}`), 0o644); err != nil {
				t.Fatalf("WriteFile(%q): %v", outputPath, err)
			}

			if cmd.Stdin != "Return JSON only." {
				t.Fatalf("stdin = %q, want prompt", cmd.Stdin)
			}

			return executor.Result{ExitCode: 0, Stderr: "stderr"}, nil
		},
	}

	exec := Executor{Runner: runner, CommandName: "codex"}
	resp, err := exec.Run(context.Background(), runtime.TaskRequest{
		Model:      "gpt-5.4",
		Prompt:     "Return JSON only.",
		ProjectDir: "/tmp/work",
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

type fakeRunner struct {
	t   *testing.T
	run func(context.Context, executor.Command) (executor.Result, error)
}

func (f *fakeRunner) Run(ctx context.Context, cmd executor.Command) (executor.Result, error) {
	return f.run(ctx, cmd)
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
