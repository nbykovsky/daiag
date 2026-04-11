package runtime

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"daiag/internal/logging"
	"daiag/internal/workflow"
)

func TestEngineRunWorkflowWithLoop(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Read "${SPEC_PATH}" and write to "${POEM_PATH}".`)
	writeFile(t, filepath.Join(baseDir, "agents", "reviewer.md"), `Review "${POEM_PATH}" and write to "${REVIEW_PATH}".`)
	writeFile(t, filepath.Join(baseDir, "docs", "features", "rain", "spec.md"), "Topic: midnight rain\n")

	var logOutput bytes.Buffer
	logger := logging.New(&logOutput)
	logger.Now = func() time.Time {
		return time.Date(2026, 4, 9, 12, 0, 1, 0, time.UTC)
	}

	executor := &fakeExecutor{
		t:       t,
		baseDir: baseDir,
	}

	engine := Engine{
		Executors: map[string]Executor{
			"codex":  executor,
			"claude": executor,
		},
		Logger: logger,
	}

	wf := &workflow.Workflow{
		ID:              "poem",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID: "write_poem",
				Prompt: workflow.Prompt{
					TemplatePath: "agents/writer.md",
					Vars: map[string]workflow.StringExpr{
						"SPEC_PATH": workflow.Literal{Value: "docs/features/rain/spec.md"},
						"POEM_PATH": workflow.Literal{Value: "docs/features/rain/poem.md"},
					},
				},
				Artifacts: map[string]workflow.StringExpr{
					"poem": workflow.Literal{Value: "docs/features/rain/poem.md"},
				},
				ResultKeys: []string{"topic", "line_count", "poem_path"},
			},
			&workflow.RepeatUntil{
				ID:       "extend_until_ready",
				MaxIters: 3,
				Steps: []workflow.Node{
					&workflow.Task{
						ID:       "review_poem",
						Executor: &workflow.ExecutorConfig{CLI: "claude", Model: "sonnet"},
						Prompt: workflow.Prompt{
							TemplatePath: "agents/reviewer.md",
							Vars: map[string]workflow.StringExpr{
								"POEM_PATH": workflow.PathRef{StepID: "write_poem", ArtifactKey: "poem"},
								"REVIEW_PATH": workflow.FormatExpr{
									Template: "docs/features/rain/review-{iter}.txt",
									Args: map[string]workflow.ValueExpr{
										"iter": workflow.LoopIter{LoopID: "extend_until_ready"},
									},
								},
							},
						},
						Artifacts: map[string]workflow.StringExpr{
							"review": workflow.FormatExpr{
								Template: "docs/features/rain/review-{iter}.txt",
								Args: map[string]workflow.ValueExpr{
									"iter": workflow.LoopIter{LoopID: "extend_until_ready"},
								},
							},
						},
						ResultKeys: []string{"outcome", "review_path"},
					},
				},
				Until: workflow.EqPredicate{
					Left:  workflow.JSONRef{StepID: "review_poem", Field: "outcome"},
					Right: workflow.Literal{Value: "ready"},
				},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{
		Workflow:     wf,
		WorkflowPath: "workflows/poem.star",
		BaseDir:      baseDir,
		Workdir:      baseDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	poemPath := filepath.Join(baseDir, "docs", "features", "rain", "poem.md")
	if _, err := os.Stat(poemPath); err != nil {
		t.Fatalf("expected poem file: %v", err)
	}
	for _, filename := range []string{"review-1.txt", "review-2.txt"} {
		reviewPath := filepath.Join(baseDir, "docs", "features", "rain", filename)
		if _, err := os.Stat(reviewPath); err != nil {
			t.Fatalf("expected review file %q: %v", filename, err)
		}
	}

	output := logOutput.String()
	required := []string{
		"workflow start id=poem file=workflows/poem.star",
		"step start id=write_poem cli=codex model=gpt-5.4",
		"step done id=write_poem artifacts=poem",
		"loop iter id=extend_until_ready n=1",
		"step start id=review_poem cli=claude model=sonnet",
		"loop check id=extend_until_ready result=continue",
		"loop iter id=extend_until_ready n=2",
		"loop check id=extend_until_ready result=stop",
		"workflow done id=poem status=success",
	}
	for _, fragment := range required {
		if !strings.Contains(output, fragment) {
			t.Fatalf("log output missing %q:\n%s", fragment, output)
		}
	}
	if !strings.Contains(executor.prompts["review_poem"], "docs/features/rain/poem.md") {
		t.Fatalf("review prompt = %q, want resolved poem path", executor.prompts["review_poem"])
	}
	if !strings.Contains(executor.prompts["review_poem"], "docs/features/rain/review-2.txt") {
		t.Fatalf("review prompt = %q, want resolved iteration-specific review path", executor.prompts["review_poem"])
	}
}

func TestEngineResolvesWorkflowInputs(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "writer.md"), `Read "${SPEC_PATH}" and write to "${POEM_PATH}".`)

	var prompt string
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				prompt = req.Prompt
				writeFile(t, filepath.Join(baseDir, "docs", "features", "rain", "poem.md"), "one\n")
				return TaskResponse{
					Stdout:   `{"ok":true}`,
					ExitCode: 0,
				}, nil
			}),
		},
	}

	wf := &workflow.Workflow{
		ID:              "poem",
		Inputs:          []string{"feature_dir", "spec_path"},
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID: "write_poem",
				Prompt: workflow.Prompt{
					TemplatePath: "agents/writer.md",
					Vars: map[string]workflow.StringExpr{
						"SPEC_PATH": workflow.InputRef{Name: "spec_path"},
						"POEM_PATH": workflow.FormatExpr{
							Template: "{dir}/poem.md",
							Args: map[string]workflow.ValueExpr{
								"dir": workflow.InputRef{Name: "feature_dir"},
							},
						},
					},
				},
				Artifacts: map[string]workflow.StringExpr{
					"poem": workflow.FormatExpr{
						Template: "{dir}/poem.md",
						Args: map[string]workflow.ValueExpr{
							"dir": workflow.InputRef{Name: "feature_dir"},
						},
					},
				},
				ResultKeys: []string{"ok"},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{
		Workflow: wf,
		BaseDir:  baseDir,
		Workdir:  baseDir,
		Inputs: map[string]any{
			"feature_dir": "docs/features/rain",
			"spec_path":   "docs/features/rain/spec.md",
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(prompt, `Read "docs/features/rain/spec.md"`) {
		t.Fatalf("prompt = %q, want resolved spec input", prompt)
	}
	if !strings.Contains(prompt, `write to "docs/features/rain/poem.md"`) {
		t.Fatalf("prompt = %q, want resolved formatted input", prompt)
	}
}

func TestEngineRunsSubworkflowAndExposesOutputs(t *testing.T) {
	baseDir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "child.md"), `literal=${LITERAL} source=${SOURCE_PATH} status=${STATUS}`)
	writeFile(t, filepath.Join(baseDir, "agents", "consume.md"), `child=${CHILD_PATH} source=${SOURCE_PATH}`)

	prompts := make(map[string]string)
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				prompts[req.WorkflowID+"."+req.TaskID] = req.Prompt
				switch req.WorkflowID + "." + req.TaskID {
				case "parent.prepare":
					writeFile(t, filepath.Join(baseDir, "docs", "source.md"), "source\n")
					return TaskResponse{Stdout: `{"status":"ready"}`, ExitCode: 0}, nil
				case "child.child_write":
					writeFile(t, filepath.Join(baseDir, "docs", "child.md"), "child\n")
					return TaskResponse{Stdout: `{"child_status":"accepted"}`, ExitCode: 0}, nil
				case "parent.consume":
					writeFile(t, filepath.Join(baseDir, "docs", "accepted.md"), "accepted\n")
					return TaskResponse{Stdout: `{"ok":true}`, ExitCode: 0}, nil
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %s.%s", req.WorkflowID, req.TaskID)
				}
			}),
		},
	}

	child := &workflow.Workflow{
		ID:              "child",
		Inputs:          []string{"literal", "source_path", "status"},
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID: "child_write",
				Prompt: workflow.Prompt{
					TemplatePath: "agents/child.md",
					Vars: map[string]workflow.StringExpr{
						"LITERAL":     workflow.InputRef{Name: "literal"},
						"SOURCE_PATH": workflow.InputRef{Name: "source_path"},
						"STATUS":      workflow.InputRef{Name: "status"},
					},
				},
				Artifacts:  map[string]workflow.StringExpr{"child": workflow.Literal{Value: "docs/child.md"}},
				ResultKeys: []string{"child_status"},
			},
		},
		OutputArtifacts: map[string]workflow.StringExpr{
			"child":  workflow.PathRef{StepID: "child_write", ArtifactKey: "child"},
			"source": workflow.InputRef{Name: "source_path"},
		},
		OutputResults: map[string]workflow.ValueExpr{
			"child_status": workflow.JSONRef{StepID: "child_write", Field: "child_status"},
		},
	}
	parent := &workflow.Workflow{
		ID:              "parent",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "prepare",
				Prompt:     workflow.Prompt{Inline: "prepare"},
				Artifacts:  map[string]workflow.StringExpr{"source": workflow.Literal{Value: "docs/source.md"}},
				ResultKeys: []string{"status"},
			},
			&workflow.Subworkflow{
				ID:           "child",
				WorkflowPath: "workflows/child.star",
				Workflow:     child,
				Inputs: map[string]workflow.ValueExpr{
					"literal":     workflow.Literal{Value: "literal-value"},
					"source_path": workflow.PathRef{StepID: "prepare", ArtifactKey: "source"},
					"status":      workflow.JSONRef{StepID: "prepare", Field: "status"},
				},
			},
			&workflow.Task{
				ID: "consume",
				Prompt: workflow.Prompt{
					TemplatePath: "agents/consume.md",
					Vars: map[string]workflow.StringExpr{
						"CHILD_PATH":  workflow.PathRef{StepID: "child", ArtifactKey: "child"},
						"SOURCE_PATH": workflow.PathRef{StepID: "child", ArtifactKey: "source"},
					},
				},
				Artifacts: map[string]workflow.StringExpr{
					"child":  workflow.PathRef{StepID: "child", ArtifactKey: "child"},
					"source": workflow.PathRef{StepID: "child", ArtifactKey: "source"},
					"status": workflow.FormatExpr{
						Template: "docs/{status}.md",
						Args: map[string]workflow.ValueExpr{
							"status": workflow.JSONRef{StepID: "child", Field: "child_status"},
						},
					},
				},
				ResultKeys: []string{"ok"},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{
		Workflow: parent,
		BaseDir:  baseDir,
		Workdir:  baseDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	childPrompt := prompts["child.child_write"]
	for _, want := range []string{"literal=literal-value", "source=docs/source.md", "status=ready"} {
		if !strings.Contains(childPrompt, want) {
			t.Fatalf("child prompt = %q, want %q", childPrompt, want)
		}
	}
	parentPrompt := prompts["parent.consume"]
	for _, want := range []string{"child=docs/child.md", "source=docs/source.md"} {
		if !strings.Contains(parentPrompt, want) {
			t.Fatalf("parent prompt = %q, want %q", parentPrompt, want)
		}
	}
}

func TestEngineDoesNotLeakSubworkflowInternalState(t *testing.T) {
	baseDir := t.TempDir()
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				switch req.WorkflowID + "." + req.TaskID {
				case "child.child_write":
					writeFile(t, filepath.Join(baseDir, "docs", "child.md"), "child\n")
					return TaskResponse{Stdout: `{"ok":true}`, ExitCode: 0}, nil
				case "parent.consume":
					return TaskResponse{Stdout: `{"ok":true}`, ExitCode: 0}, nil
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %s.%s", req.WorkflowID, req.TaskID)
				}
			}),
		},
	}

	child := &workflow.Workflow{
		ID:              "child",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "child_write",
				Prompt:     workflow.Prompt{Inline: "child"},
				Artifacts:  map[string]workflow.StringExpr{"child": workflow.Literal{Value: "docs/child.md"}},
				ResultKeys: []string{"ok"},
			},
		},
		OutputArtifacts: map[string]workflow.StringExpr{
			"child": workflow.PathRef{StepID: "child_write", ArtifactKey: "child"},
		},
	}
	parent := &workflow.Workflow{
		ID:              "parent",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Subworkflow{
				ID:           "child",
				WorkflowPath: "workflows/child.star",
				Workflow:     child,
				Inputs:       map[string]workflow.ValueExpr{},
			},
			&workflow.Task{
				ID:         "consume",
				Prompt:     workflow.Prompt{Inline: "consume"},
				Artifacts:  map[string]workflow.StringExpr{"leaked": workflow.PathRef{StepID: "child_write", ArtifactKey: "child"}},
				ResultKeys: []string{"ok"},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{
		Workflow: parent,
		BaseDir:  baseDir,
		Workdir:  baseDir,
	})
	if err == nil || !strings.Contains(err.Error(), `missing artifacts for step "child_write"`) {
		t.Fatalf("Run() error = %v, want missing child internal artifact", err)
	}
}

func TestEngineSubworkflowInLoopUsesLatestOutput(t *testing.T) {
	baseDir := t.TempDir()
	childRuns := 0
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				switch req.WorkflowID + "." + req.TaskID {
				case "child.check":
					childRuns++
					outcome := "not_ready"
					if childRuns >= 2 {
						outcome = "ready"
					}
					writeFile(t, filepath.Join(baseDir, "docs", fmt.Sprintf("child-%d.md", childRuns)), outcome+"\n")
					return TaskResponse{
						Stdout:   fmt.Sprintf(`{"outcome":"%s"}`, outcome),
						ExitCode: 0,
					}, nil
				case "parent.consume":
					writeFile(t, filepath.Join(baseDir, "docs", "ready.md"), "ready\n")
					return TaskResponse{Stdout: `{"ok":true}`, ExitCode: 0}, nil
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %s.%s", req.WorkflowID, req.TaskID)
				}
			}),
		},
	}

	child := &workflow.Workflow{
		ID:              "child",
		Inputs:          []string{"iter"},
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:     "check",
				Prompt: workflow.Prompt{Inline: "check"},
				Artifacts: map[string]workflow.StringExpr{
					"report": workflow.FormatExpr{
						Template: "docs/child-{iter}.md",
						Args: map[string]workflow.ValueExpr{
							"iter": workflow.InputRef{Name: "iter"},
						},
					},
				},
				ResultKeys: []string{"outcome"},
			},
		},
		OutputArtifacts: map[string]workflow.StringExpr{
			"report": workflow.PathRef{StepID: "check", ArtifactKey: "report"},
		},
		OutputResults: map[string]workflow.ValueExpr{
			"outcome": workflow.JSONRef{StepID: "check", Field: "outcome"},
		},
	}
	parent := &workflow.Workflow{
		ID:              "parent",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.RepeatUntil{
				ID:       "review_loop",
				MaxIters: 3,
				Steps: []workflow.Node{
					&workflow.Subworkflow{
						ID:           "child",
						WorkflowPath: "workflows/child.star",
						Workflow:     child,
						Inputs: map[string]workflow.ValueExpr{
							"iter": workflow.LoopIter{LoopID: "review_loop"},
						},
					},
				},
				Until: workflow.EqPredicate{
					Left:  workflow.JSONRef{StepID: "child", Field: "outcome"},
					Right: workflow.Literal{Value: "ready"},
				},
			},
			&workflow.Task{
				ID:     "consume",
				Prompt: workflow.Prompt{Inline: "consume"},
				Artifacts: map[string]workflow.StringExpr{
					"latest": workflow.PathRef{StepID: "child", ArtifactKey: "report"},
					"status": workflow.FormatExpr{
						Template: "docs/{outcome}.md",
						Args: map[string]workflow.ValueExpr{
							"outcome": workflow.JSONRef{StepID: "child", Field: "outcome"},
						},
					},
				},
				ResultKeys: []string{"ok"},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{
		Workflow: parent,
		BaseDir:  baseDir,
		Workdir:  baseDir,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if childRuns != 2 {
		t.Fatalf("childRuns = %d, want 2", childRuns)
	}
}

func TestEngineFailsWhenArtifactMissing(t *testing.T) {
	baseDir := t.TempDir()
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				return TaskResponse{
					Stdout:   `{"ok":true}`,
					ExitCode: 0,
				}, nil
			}),
		},
	}

	wf := &workflow.Workflow{
		ID:              "missing-artifact",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "write_poem",
				Prompt:     workflow.Prompt{Inline: "hello"},
				Artifacts:  map[string]workflow.StringExpr{"poem": workflow.Literal{Value: "docs/poem.md"}},
				ResultKeys: []string{"ok"},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{
		Workflow: wf,
		BaseDir:  baseDir,
		Workdir:  baseDir,
	})
	if err == nil || !strings.Contains(err.Error(), `artifact "poem"`) {
		t.Fatalf("Run() error = %v, want artifact error", err)
	}
}

func TestRenderPromptUsesTemplateDir(t *testing.T) {
	baseDir := t.TempDir()
	moduleDir := filepath.Join(baseDir, "workflows", "lib")
	writeFile(t, filepath.Join(baseDir, "workflows", "agents", "writer.md"), `Read "${POEM_PATH}".`)

	prompt := workflow.Prompt{
		TemplatePath: "../agents/writer.md",
		TemplateDir:  moduleDir,
		Vars: map[string]workflow.StringExpr{
			"POEM_PATH": workflow.Literal{Value: "docs/poem.md"},
		},
	}

	rendered, err := renderPrompt(prompt, baseDir, &state{})
	if err != nil {
		t.Fatalf("renderPrompt() error = %v", err)
	}
	if rendered != `Read "docs/poem.md".` {
		t.Fatalf("renderPrompt() = %q, want resolved template content", rendered)
	}
}

func TestParseResultAcceptsMixedOutput(t *testing.T) {
	stdout := strings.TrimSpace(`
The review is complete.

` + "```json" + `
{"outcome":"ready","line_count":6,"review_path":"docs/features/rain/review-2.txt"}
` + "```")

	result, err := parseResult(stdout, []string{"outcome", "line_count", "review_path"})
	if err != nil {
		t.Fatalf("parseResult() error = %v", err)
	}
	if got := result["outcome"]; got != "ready" {
		t.Fatalf("result[outcome] = %#v, want %q", got, "ready")
	}
	if got := result["review_path"]; got != "docs/features/rain/review-2.txt" {
		t.Fatalf("result[review_path] = %#v, want review path", got)
	}
}

func TestParseResultRejectsOutputWithoutJSONObject(t *testing.T) {
	_, err := parseResult("The task completed successfully.", []string{"outcome"})
	if err == nil || !strings.Contains(err.Error(), "no JSON object found") {
		t.Fatalf("parseResult() error = %v, want no JSON object error", err)
	}
}

type fakeExecutor struct {
	t         *testing.T
	baseDir   string
	prompts   map[string]string
	reviewRun int
}

func (f *fakeExecutor) Run(_ context.Context, req TaskRequest) (TaskResponse, error) {
	if f.prompts == nil {
		f.prompts = make(map[string]string)
	}
	f.prompts[req.TaskID] = req.Prompt

	switch req.TaskID {
	case "write_poem":
		path := filepath.Join(f.baseDir, "docs", "features", "rain", "poem.md")
		writeFile(f.t, path, "one\n")
		return TaskResponse{
			Stdout:   `{"topic":"midnight rain","line_count":4,"poem_path":"docs/features/rain/poem.md"}`,
			ExitCode: 0,
		}, nil
	case "review_poem":
		f.reviewRun++
		path := filepath.Join(f.baseDir, "docs", "features", "rain", fmt.Sprintf("review-%d.txt", f.reviewRun))
		writeFile(f.t, path, fmt.Sprintf("run=%d\n", f.reviewRun))
		outcome := "not_ready"
		if f.reviewRun >= 2 {
			outcome = "ready"
		}
		return TaskResponse{
			Stdout:   fmt.Sprintf(`{"outcome":"%s","review_path":"docs/features/rain/review-%d.txt"}`, outcome, f.reviewRun),
			ExitCode: 0,
		}, nil
	default:
		return TaskResponse{}, fmt.Errorf("unexpected task %q", req.TaskID)
	}
}

type fakeExecutorFunc func(context.Context, TaskRequest) (TaskResponse, error)

func (f fakeExecutorFunc) Run(ctx context.Context, req TaskRequest) (TaskResponse, error) {
	return f(ctx, req)
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
