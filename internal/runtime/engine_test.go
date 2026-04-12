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

func TestEngineResolvesWorkdirExpressionAndStoresAbsoluteArtifactPaths(t *testing.T) {
	baseDir := t.TempDir()
	workdir := t.TempDir()
	writeFile(t, filepath.Join(baseDir, "agents", "write.md"), `out=${OUT_PATH}`)
	writeFile(t, filepath.Join(baseDir, "agents", "consume.md"), `poem=${POEM_PATH}`)

	prompts := make(map[string]string)
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				if req.Workdir != workdir {
					return TaskResponse{}, fmt.Errorf("Workdir = %q, want %q", req.Workdir, workdir)
				}
				prompts[req.TaskID] = req.Prompt
				switch req.TaskID {
				case "write_poem":
					writeFile(t, filepath.Join(workdir, "docs", "poem.md"), "one\n")
				case "consume_poem":
					writeFile(t, filepath.Join(workdir, "docs", "summary.md"), "summary\n")
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %q", req.TaskID)
				}
				return TaskResponse{Stdout: `{"ok":true}`, ExitCode: 0}, nil
			}),
		},
	}

	wf := &workflow.Workflow{
		ID:              "poem",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID: "write_poem",
				Prompt: workflow.Prompt{
					TemplatePath: "agents/write.md",
					Vars: map[string]workflow.StringExpr{
						"OUT_PATH": workflow.FormatExpr{
							Template: "{workdir}/explicit.md",
							Args: map[string]workflow.ValueExpr{
								"workdir": workflow.WorkdirRef{},
							},
						},
					},
				},
				Artifacts:  map[string]workflow.StringExpr{"poem": workflow.Literal{Value: "docs/poem.md"}},
				ResultKeys: []string{"ok"},
			},
			&workflow.Task{
				ID: "consume_poem",
				Prompt: workflow.Prompt{
					TemplatePath: "agents/consume.md",
					Vars: map[string]workflow.StringExpr{
						"POEM_PATH": workflow.PathRef{StepID: "write_poem", ArtifactKey: "poem"},
					},
				},
				Artifacts:  map[string]workflow.StringExpr{"summary": workflow.Literal{Value: "docs/summary.md"}},
				ResultKeys: []string{"ok"},
			},
		},
	}

	if err := engine.Run(context.Background(), RunInput{Workflow: wf, BaseDir: baseDir, Workdir: workdir}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := prompts["write_poem"], "out="+filepath.Join(workdir, "explicit.md"); got != want {
		t.Fatalf("write prompt = %q, want %q", got, want)
	}
	if got, want := prompts["consume_poem"], "poem="+filepath.Join(workdir, "docs", "poem.md"); got != want {
		t.Fatalf("consume prompt = %q, want %q", got, want)
	}
}

func TestEngineRunsWhenStepsWhenConditionTrue(t *testing.T) {
	baseDir := t.TempDir()
	calls := []string{}
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				calls = append(calls, req.TaskID)
				switch req.TaskID {
				case "triage":
					writeFile(t, filepath.Join(baseDir, "docs", "triage.md"), "triage\n")
					return TaskResponse{Stdout: `{"outcome":"code_issues"}`, ExitCode: 0}, nil
				case "repair_code":
					writeFile(t, filepath.Join(baseDir, "docs", "repair.md"), "repair\n")
					return TaskResponse{Stdout: `{"outcome":"repaired"}`, ExitCode: 0}, nil
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %q", req.TaskID)
				}
			}),
		},
	}
	wf := &workflow.Workflow{
		ID:              "conditional",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "triage",
				Prompt:     workflow.Prompt{Inline: "triage"},
				Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/triage.md"}},
				ResultKeys: []string{"outcome"},
			},
			&workflow.When{
				ID: "address_code_issues",
				Condition: workflow.EqPredicate{
					Left:  workflow.JSONRef{StepID: "triage", Field: "outcome"},
					Right: workflow.Literal{Value: "code_issues"},
				},
				Steps: []workflow.Node{
					&workflow.Task{
						ID:         "repair_code",
						Prompt:     workflow.Prompt{Inline: "repair"},
						Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/repair.md"}},
						ResultKeys: []string{"outcome"},
					},
				},
				ElseSteps: []workflow.Node{
					&workflow.Task{
						ID:         "record_no_repair_needed",
						Prompt:     workflow.Prompt{Inline: "record"},
						Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/no-repair.md"}},
						ResultKeys: []string{"outcome"},
					},
				},
			},
		},
	}

	if err := engine.Run(context.Background(), RunInput{Workflow: wf, BaseDir: baseDir, Workdir: baseDir}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := strings.Join(calls, ","), "triage,repair_code"; got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "docs", "repair.md")); err != nil {
		t.Fatalf("expected repair artifact: %v", err)
	}
}

func TestEngineRunsWhenElseStepsWhenConditionFalse(t *testing.T) {
	baseDir := t.TempDir()
	calls := []string{}
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				calls = append(calls, req.TaskID)
				switch req.TaskID {
				case "triage":
					writeFile(t, filepath.Join(baseDir, "docs", "triage.md"), "triage\n")
					return TaskResponse{Stdout: `{"outcome":"clean"}`, ExitCode: 0}, nil
				case "record_no_repair_needed":
					writeFile(t, filepath.Join(baseDir, "docs", "no-repair.md"), "clean\n")
					return TaskResponse{Stdout: `{"outcome":"clean"}`, ExitCode: 0}, nil
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %q", req.TaskID)
				}
			}),
		},
	}
	wf := &workflow.Workflow{
		ID:              "conditional",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "triage",
				Prompt:     workflow.Prompt{Inline: "triage"},
				Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/triage.md"}},
				ResultKeys: []string{"outcome"},
			},
			&workflow.When{
				ID: "address_code_issues",
				Condition: workflow.EqPredicate{
					Left:  workflow.JSONRef{StepID: "triage", Field: "outcome"},
					Right: workflow.Literal{Value: "code_issues"},
				},
				Steps: []workflow.Node{
					&workflow.Task{
						ID:         "repair_code",
						Prompt:     workflow.Prompt{Inline: "repair"},
						Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/repair.md"}},
						ResultKeys: []string{"outcome"},
					},
				},
				ElseSteps: []workflow.Node{
					&workflow.Task{
						ID:         "record_no_repair_needed",
						Prompt:     workflow.Prompt{Inline: "record"},
						Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/no-repair.md"}},
						ResultKeys: []string{"outcome"},
					},
				},
			},
		},
	}

	if err := engine.Run(context.Background(), RunInput{Workflow: wf, BaseDir: baseDir, Workdir: baseDir}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := strings.Join(calls, ","), "triage,record_no_repair_needed"; got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "docs", "no-repair.md")); err != nil {
		t.Fatalf("expected else artifact: %v", err)
	}
}

func TestEngineSkipsWhenWithoutElseStepsWhenConditionFalse(t *testing.T) {
	baseDir := t.TempDir()
	calls := []string{}
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				calls = append(calls, req.TaskID)
				switch req.TaskID {
				case "triage":
					writeFile(t, filepath.Join(baseDir, "docs", "triage.md"), "triage\n")
					return TaskResponse{Stdout: `{"outcome":"clean"}`, ExitCode: 0}, nil
				case "after":
					writeFile(t, filepath.Join(baseDir, "docs", "after.md"), "after\n")
					return TaskResponse{Stdout: `{"ok":true}`, ExitCode: 0}, nil
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %q", req.TaskID)
				}
			}),
		},
	}
	wf := &workflow.Workflow{
		ID:              "conditional",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "triage",
				Prompt:     workflow.Prompt{Inline: "triage"},
				Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/triage.md"}},
				ResultKeys: []string{"outcome"},
			},
			&workflow.When{
				ID: "address_code_issues",
				Condition: workflow.EqPredicate{
					Left:  workflow.JSONRef{StepID: "triage", Field: "outcome"},
					Right: workflow.Literal{Value: "code_issues"},
				},
				Steps: []workflow.Node{
					&workflow.Task{
						ID:         "repair_code",
						Prompt:     workflow.Prompt{Inline: "repair"},
						Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/repair.md"}},
						ResultKeys: []string{"outcome"},
					},
				},
			},
			&workflow.Task{
				ID:         "after",
				Prompt:     workflow.Prompt{Inline: "after"},
				Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/after.md"}},
				ResultKeys: []string{"ok"},
			},
		},
	}

	if err := engine.Run(context.Background(), RunInput{Workflow: wf, BaseDir: baseDir, Workdir: baseDir}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := strings.Join(calls, ","), "triage,after"; got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
}

func TestEngineEvaluatesWhenOnEachRepeatUntilIteration(t *testing.T) {
	baseDir := t.TempDir()
	calls := []string{}
	checkRuns := 0
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				calls = append(calls, req.TaskID)
				switch req.TaskID {
				case "check":
					checkRuns++
					outcome := "code_issues"
					if checkRuns == 2 {
						outcome = "ready"
					}
					writeFile(t, filepath.Join(baseDir, "docs", "check.md"), outcome+"\n")
					return TaskResponse{Stdout: fmt.Sprintf(`{"outcome":"%s"}`, outcome), ExitCode: 0}, nil
				case "repair_code":
					writeFile(t, filepath.Join(baseDir, "docs", "repair.md"), "repair\n")
					return TaskResponse{Stdout: `{"outcome":"repaired"}`, ExitCode: 0}, nil
				case "record_clean":
					writeFile(t, filepath.Join(baseDir, "docs", "clean.md"), "clean\n")
					return TaskResponse{Stdout: `{"outcome":"clean"}`, ExitCode: 0}, nil
				default:
					return TaskResponse{}, fmt.Errorf("unexpected task %q", req.TaskID)
				}
			}),
		},
	}
	wf := &workflow.Workflow{
		ID:              "conditional-loop",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.RepeatUntil{
				ID:       "review_loop",
				MaxIters: 3,
				Steps: []workflow.Node{
					&workflow.Task{
						ID:         "check",
						Prompt:     workflow.Prompt{Inline: "check"},
						Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/check.md"}},
						ResultKeys: []string{"outcome"},
					},
					&workflow.When{
						ID: "repair_if_needed",
						Condition: workflow.EqPredicate{
							Left:  workflow.JSONRef{StepID: "check", Field: "outcome"},
							Right: workflow.Literal{Value: "code_issues"},
						},
						Steps: []workflow.Node{
							&workflow.Task{
								ID:         "repair_code",
								Prompt:     workflow.Prompt{Inline: "repair"},
								Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/repair.md"}},
								ResultKeys: []string{"outcome"},
							},
						},
						ElseSteps: []workflow.Node{
							&workflow.Task{
								ID:         "record_clean",
								Prompt:     workflow.Prompt{Inline: "record"},
								Artifacts:  map[string]workflow.StringExpr{"status": workflow.Literal{Value: "docs/clean.md"}},
								ResultKeys: []string{"outcome"},
							},
						},
					},
				},
				Until: workflow.EqPredicate{
					Left:  workflow.JSONRef{StepID: "check", Field: "outcome"},
					Right: workflow.Literal{Value: "ready"},
				},
			},
		},
	}

	if err := engine.Run(context.Background(), RunInput{Workflow: wf, BaseDir: baseDir, Workdir: baseDir}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got, want := strings.Join(calls, ","), "check,repair_code,check,record_clean"; got != want {
		t.Fatalf("calls = %s, want %s", got, want)
	}
}

func TestEngineAttributesWhenConditionEvaluationFailureToWhenID(t *testing.T) {
	wf := &workflow.Workflow{
		ID: "bad-conditional",
		Steps: []workflow.Node{
			&workflow.When{
				ID: "gate",
				Condition: workflow.EqPredicate{
					Left:  workflow.JSONRef{StepID: "missing", Field: "outcome"},
					Right: workflow.Literal{Value: "ready"},
				},
			},
		},
	}

	err := (Engine{}).Run(context.Background(), RunInput{Workflow: wf})
	if err == nil || !strings.Contains(err.Error(), `step gate: missing result for step "missing"`) {
		t.Fatalf("Run() error = %v, want when step error", err)
	}
	stepID, ok := errStepID(err)
	if !ok || stepID != "gate" {
		t.Fatalf("errStepID() = %q, %v; want gate, true", stepID, ok)
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
	for _, want := range []string{
		"literal=literal-value",
		"source=" + filepath.Join(baseDir, "docs", "source.md"),
		"status=ready",
	} {
		if !strings.Contains(childPrompt, want) {
			t.Fatalf("child prompt = %q, want %q", childPrompt, want)
		}
	}
	parentPrompt := prompts["parent.consume"]
	for _, want := range []string{
		"child=" + filepath.Join(baseDir, "docs", "child.md"),
		"source=" + filepath.Join(baseDir, "docs", "source.md"),
	} {
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

func TestEngineLogsSubworkflowStartAndDone(t *testing.T) {
	baseDir := t.TempDir()
	var logOutput bytes.Buffer
	logger := logging.New(&logOutput)
	logger.Now = func() time.Time {
		return time.Date(2026, 4, 9, 12, 0, 1, 0, time.UTC)
	}

	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				if req.WorkflowID != "child" || req.TaskID != "write" {
					return TaskResponse{}, fmt.Errorf("unexpected task %s.%s", req.WorkflowID, req.TaskID)
				}
				writeFile(t, filepath.Join(baseDir, "docs", "child.md"), "child\n")
				return TaskResponse{Stdout: `{"ok":true}`, ExitCode: 0}, nil
			}),
		},
		Logger: logger,
	}
	child := &workflow.Workflow{
		ID:              "child",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "write",
				Prompt:     workflow.Prompt{Inline: "child"},
				Artifacts:  map[string]workflow.StringExpr{"child": workflow.Literal{Value: "docs/child.md"}},
				ResultKeys: []string{"ok"},
			},
		},
		OutputArtifacts: map[string]workflow.StringExpr{
			"child": workflow.PathRef{StepID: "write", ArtifactKey: "child"},
		},
		OutputResults: map[string]workflow.ValueExpr{
			"ok": workflow.JSONRef{StepID: "write", Field: "ok"},
		},
	}
	parent := &workflow.Workflow{
		ID: "parent",
		Steps: []workflow.Node{
			&workflow.Subworkflow{
				ID:           "child",
				WorkflowPath: "workflows/child.star",
				Workflow:     child,
				Inputs:       map[string]workflow.ValueExpr{},
			},
		},
	}

	if err := engine.Run(context.Background(), RunInput{Workflow: parent, BaseDir: baseDir, Workdir: baseDir}); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	output := logOutput.String()
	for _, fragment := range []string{
		"subworkflow start id=child workflow=child",
		"subworkflow done id=child artifacts=child results=ok",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("log output missing %q:\n%s", fragment, output)
		}
	}
}

func TestEngineQualifiesSingleLevelSubworkflowFailure(t *testing.T) {
	baseDir := t.TempDir()
	var logOutput bytes.Buffer
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				return TaskResponse{Stderr: "boom", ExitCode: 1}, nil
			}),
		},
		Logger: logging.New(&logOutput),
	}
	child := &workflow.Workflow{
		ID:              "spec_refinement",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "review_spec",
				Prompt:     workflow.Prompt{Inline: "review"},
				Artifacts:  map[string]workflow.StringExpr{"review": workflow.Literal{Value: "docs/review.md"}},
				ResultKeys: []string{"ok"},
			},
		},
	}
	parent := &workflow.Workflow{
		ID: "parent",
		Steps: []workflow.Node{
			&workflow.Subworkflow{
				ID:           "spec",
				WorkflowPath: "workflows/spec.star",
				Workflow:     child,
				Inputs:       map[string]workflow.ValueExpr{},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{Workflow: parent, BaseDir: baseDir, Workdir: baseDir})
	if err == nil || !strings.Contains(err.Error(), `step spec.review_spec`) {
		t.Fatalf("Run() error = %v, want qualified child step", err)
	}
	if !strings.Contains(logOutput.String(), `subworkflow failed id=spec step=spec.review_spec`) {
		t.Fatalf("log output missing qualified failure:\n%s", logOutput.String())
	}
}

func TestEngineQualifiesNestedSubworkflowFailure(t *testing.T) {
	baseDir := t.TempDir()
	engine := Engine{
		Executors: map[string]Executor{
			"codex": fakeExecutorFunc(func(_ context.Context, req TaskRequest) (TaskResponse, error) {
				return TaskResponse{Stderr: "boom", ExitCode: 1}, nil
			}),
		},
	}
	inner := &workflow.Workflow{
		ID:              "inner",
		DefaultExecutor: &workflow.ExecutorConfig{CLI: "codex", Model: "gpt-5.4"},
		Steps: []workflow.Node{
			&workflow.Task{
				ID:         "inner_task",
				Prompt:     workflow.Prompt{Inline: "inner"},
				Artifacts:  map[string]workflow.StringExpr{"out": workflow.Literal{Value: "docs/out.md"}},
				ResultKeys: []string{"ok"},
			},
		},
	}
	refine := &workflow.Workflow{
		ID: "refine",
		Steps: []workflow.Node{
			&workflow.Subworkflow{
				ID:           "refine",
				WorkflowPath: "workflows/inner.star",
				Workflow:     inner,
				Inputs:       map[string]workflow.ValueExpr{},
			},
		},
	}
	parent := &workflow.Workflow{
		ID: "parent",
		Steps: []workflow.Node{
			&workflow.Subworkflow{
				ID:           "spec",
				WorkflowPath: "workflows/refine.star",
				Workflow:     refine,
				Inputs:       map[string]workflow.ValueExpr{},
			},
		},
	}

	err := engine.Run(context.Background(), RunInput{Workflow: parent, BaseDir: baseDir, Workdir: baseDir})
	if err == nil || !strings.Contains(err.Error(), `step spec.refine.inner_task`) {
		t.Fatalf("Run() error = %v, want recursively qualified child step", err)
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
