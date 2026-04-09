package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"daiag/internal/logging"
	"daiag/internal/workflow"
)

var placeholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

type Executor interface {
	Run(context.Context, TaskRequest) (TaskResponse, error)
}

type TaskRequest struct {
	WorkflowID string
	TaskID     string
	Prompt     string
	Model      string
	Workdir    string
}

type TaskResponse struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type RunInput struct {
	Workflow     *workflow.Workflow
	WorkflowPath string
	BaseDir      string
	Workdir      string
}

type Engine struct {
	Executors map[string]Executor
	Logger    *logging.Logger
}

type state struct {
	artifacts map[string]map[string]string
	results   map[string]map[string]any
	loops     map[string]int
}

func (e Engine) Run(ctx context.Context, input RunInput) error {
	if input.Workflow == nil {
		return fmt.Errorf("workflow is nil")
	}

	st := &state{
		artifacts: make(map[string]map[string]string),
		results:   make(map[string]map[string]any),
		loops:     make(map[string]int),
	}

	if e.Logger != nil {
		e.Logger.WorkflowStart(input.Workflow.ID, input.WorkflowPath)
	}

	if err := e.runNodes(ctx, input, st, input.Workflow.Steps); err != nil {
		if e.Logger != nil {
			stepID, _ := errStepID(err)
			e.Logger.WorkflowFailed(input.Workflow.ID, stepID, err)
		}
		return err
	}

	if e.Logger != nil {
		e.Logger.WorkflowDone(input.Workflow.ID)
	}

	return nil
}

func (e Engine) runNodes(ctx context.Context, input RunInput, st *state, nodes []workflow.Node) error {
	for _, node := range nodes {
		switch n := node.(type) {
		case *workflow.Task:
			if err := e.runTask(ctx, input, st, n); err != nil {
				return err
			}
		case *workflow.RepeatUntil:
			if err := e.runRepeatUntil(ctx, input, st, n); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported node type %T", node)
		}
	}
	return nil
}

func (e Engine) runTask(ctx context.Context, input RunInput, st *state, task *workflow.Task) error {
	executorConfig, err := resolveExecutor(input.Workflow, task)
	if err != nil {
		return stepError{StepID: task.ID, Err: err}
	}

	prompt, err := renderPrompt(task.Prompt, input.BaseDir, st)
	if err != nil {
		return stepError{StepID: task.ID, Err: err}
	}

	executor, ok := e.Executors[executorConfig.CLI]
	if !ok {
		return stepError{StepID: task.ID, Err: fmt.Errorf("unknown executor %q", executorConfig.CLI)}
	}

	if e.Logger != nil {
		e.Logger.StepStart(task.ID, executorConfig.CLI, executorConfig.Model)
	}

	response, err := executor.Run(ctx, TaskRequest{
		WorkflowID: input.Workflow.ID,
		TaskID:     task.ID,
		Prompt:     prompt,
		Model:      executorConfig.Model,
		Workdir:    input.Workdir,
	})
	if err != nil {
		return stepError{StepID: task.ID, Err: err}
	}
	if response.ExitCode != 0 {
		return stepError{StepID: task.ID, Err: fmt.Errorf("executor exited with code %d: %s", response.ExitCode, response.Stderr)}
	}

	result, err := parseResult(response.Stdout, task.ResultKeys)
	if err != nil {
		return stepError{StepID: task.ID, Err: err}
	}

	artifacts := make(map[string]string, len(task.Artifacts))
	artifactKeys := make([]string, 0, len(task.Artifacts))
	for key, expr := range task.Artifacts {
		path, err := resolveStringExpr(expr, st)
		if err != nil {
			return stepError{StepID: task.ID, Err: fmt.Errorf("resolve artifact %q: %w", key, err)}
		}
		if err := ensureFileExists(input.Workdir, path); err != nil {
			return stepError{StepID: task.ID, Err: fmt.Errorf("artifact %q: %w", key, err)}
		}
		artifacts[key] = path
		artifactKeys = append(artifactKeys, key)
	}

	st.artifacts[task.ID] = artifacts
	st.results[task.ID] = result

	if e.Logger != nil {
		e.Logger.StepDone(task.ID, logging.SortKeys(artifactKeys), result)
	}

	return nil
}

func (e Engine) runRepeatUntil(ctx context.Context, input RunInput, st *state, loop *workflow.RepeatUntil) error {
	defer delete(st.loops, loop.ID)

	for i := 1; i <= loop.MaxIters; i++ {
		st.loops[loop.ID] = i

		if e.Logger != nil {
			e.Logger.LoopIter(loop.ID, i)
		}

		if err := e.runNodes(ctx, input, st, loop.Steps); err != nil {
			return err
		}

		ok, err := evalPredicate(loop.Until, st)
		if err != nil {
			return stepError{StepID: loop.ID, Err: err}
		}

		if ok {
			if e.Logger != nil {
				e.Logger.LoopCheck(loop.ID, "stop")
			}
			return nil
		}

		if e.Logger != nil {
			e.Logger.LoopCheck(loop.ID, "continue")
		}
	}

	return stepError{
		StepID: loop.ID,
		Err:    fmt.Errorf("loop reached max_iters=%d without satisfying condition", loop.MaxIters),
	}
}

func resolveExecutor(wf *workflow.Workflow, task *workflow.Task) (*workflow.ExecutorConfig, error) {
	if task.Executor != nil {
		return task.Executor, nil
	}
	if wf.DefaultExecutor != nil {
		return wf.DefaultExecutor, nil
	}
	return nil, fmt.Errorf("executor is required")
}

func renderPrompt(prompt workflow.Prompt, baseDir string, st *state) (string, error) {
	if prompt.IsInline() {
		return prompt.Inline, nil
	}

	path := prompt.TemplatePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt template %q: %w", prompt.TemplatePath, err)
	}

	content := placeholderPattern.ReplaceAllStringFunc(string(data), func(match string) string {
		name := placeholderPattern.FindStringSubmatch(match)[1]
		expr, ok := prompt.Vars[name]
		if !ok {
			return match
		}
		value, err := resolveStringExpr(expr, st)
		if err != nil {
			return fmt.Sprintf("<<error:%s>>", err)
		}
		return value
	})

	if unresolved := placeholderPattern.FindStringSubmatch(content); unresolved != nil {
		return "", fmt.Errorf("unresolved prompt variable %q", unresolved[1])
	}
	if match := regexp.MustCompile(`<<error:(.+)>>`).FindStringSubmatch(content); match != nil {
		return "", fmt.Errorf("resolve prompt variable: %s", match[1])
	}

	return content, nil
}

func parseResult(stdout string, requiredKeys []string) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		return nil, fmt.Errorf("parse task result: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("task result must be a JSON object")
	}
	for _, key := range requiredKeys {
		if _, ok := result[key]; !ok {
			return nil, fmt.Errorf("result missing key %q", key)
		}
	}
	return result, nil
}

func ensureFileExists(workdir, path string) error {
	checkPath := path
	if !filepath.IsAbs(checkPath) {
		checkPath = filepath.Join(workdir, checkPath)
	}
	info, err := os.Stat(checkPath)
	if err != nil {
		return fmt.Errorf("expected file %q: %w", path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("expected file %q, got directory", path)
	}
	return nil
}

func resolveStringExpr(expr workflow.StringExpr, st *state) (string, error) {
	switch e := expr.(type) {
	case workflow.Literal:
		return e.Value, nil
	case workflow.FormatExpr:
		return renderFormatExpr(e, st)
	case workflow.PathRef:
		artifacts, ok := st.artifacts[e.StepID]
		if !ok {
			return "", fmt.Errorf("missing artifacts for step %q", e.StepID)
		}
		path, ok := artifacts[e.ArtifactKey]
		if !ok {
			return "", fmt.Errorf("missing artifact %q on step %q", e.ArtifactKey, e.StepID)
		}
		return path, nil
	default:
		return "", fmt.Errorf("unsupported string expression type %T", expr)
	}
}

func resolveValueExpr(expr workflow.ValueExpr, st *state) (any, error) {
	switch e := expr.(type) {
	case workflow.Literal:
		return e.Value, nil
	case workflow.IntLiteral:
		return e.Value, nil
	case workflow.FormatExpr:
		return renderFormatExpr(e, st)
	case workflow.PathRef:
		return resolveStringExpr(e, st)
	case workflow.JSONRef:
		result, ok := st.results[e.StepID]
		if !ok {
			return nil, fmt.Errorf("missing result for step %q", e.StepID)
		}
		value, ok := result[e.Field]
		if !ok {
			return nil, fmt.Errorf("missing result field %q on step %q", e.Field, e.StepID)
		}
		return value, nil
	case workflow.LoopIter:
		iter, ok := st.loops[e.LoopID]
		if !ok {
			return nil, fmt.Errorf("loop %q is not active", e.LoopID)
		}
		return iter, nil
	default:
		return nil, fmt.Errorf("unsupported value expression type %T", expr)
	}
}

func renderFormatExpr(expr workflow.FormatExpr, st *state) (string, error) {
	values := make(map[string]string, len(expr.Args))
	for key, valueExpr := range expr.Args {
		value, err := resolveValueExpr(valueExpr, st)
		if err != nil {
			return "", fmt.Errorf("resolve format arg %q: %w", key, err)
		}
		values[key] = fmt.Sprint(value)
	}

	var rendered strings.Builder
	for i := 0; i < len(expr.Template); {
		if expr.Template[i] != '{' {
			rendered.WriteByte(expr.Template[i])
			i++
			continue
		}
		end := strings.IndexByte(expr.Template[i:], '}')
		if end <= 1 {
			return "", fmt.Errorf("malformed format template %q", expr.Template)
		}
		end += i
		key := expr.Template[i+1 : end]
		value, ok := values[key]
		if !ok {
			return "", fmt.Errorf("missing format value %q", key)
		}
		rendered.WriteString(value)
		i = end + 1
	}

	return rendered.String(), nil
}

func evalPredicate(predicate workflow.Predicate, st *state) (bool, error) {
	switch p := predicate.(type) {
	case workflow.EqPredicate:
		left, err := resolveValueExpr(p.Left, st)
		if err != nil {
			return false, err
		}
		right, err := resolveValueExpr(p.Right, st)
		if err != nil {
			return false, err
		}
		return fmt.Sprint(left) == fmt.Sprint(right), nil
	default:
		return false, fmt.Errorf("unsupported predicate type %T", predicate)
	}
}

type stepError struct {
	StepID string
	Err    error
}

func (e stepError) Error() string {
	return fmt.Sprintf("step %s: %v", e.StepID, e.Err)
}

func (e stepError) Unwrap() error {
	return e.Err
}

func errStepID(err error) (string, bool) {
	var target stepError
	if ok := asStepError(err, &target); ok {
		return target.StepID, true
	}
	return "", false
}

func asStepError(err error, target *stepError) bool {
	current, ok := err.(stepError)
	if ok {
		*target = current
		return true
	}
	type unwrapper interface{ Unwrap() error }
	w, ok := err.(unwrapper)
	if !ok {
		return false
	}
	next := w.Unwrap()
	if next == nil {
		return false
	}
	return asStepError(next, target)
}
