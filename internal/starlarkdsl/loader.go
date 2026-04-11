package starlarkdsl

import (
	"fmt"
	"path/filepath"
	"strings"

	"daiag/internal/workflow"

	"go.starlark.net/starlark"
)

type Loader struct {
	Params  map[string]string
	Inputs  map[string]string
	BaseDir string

	paramDisabled bool
}

func (l Loader) Load(path string) (*workflow.Workflow, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve workflow path: %w", err)
	}

	baseDir := l.BaseDir
	if baseDir == "" {
		baseDir = filepath.Dir(absPath)
	}
	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve workflow base dir: %w", err)
	}

	ctx := &workflowLoadContext{}
	wf, err := l.loadWorkflow(absPath, baseDir, ctx, false)
	if err != nil {
		return nil, err
	}

	validator := workflow.Validator{
		BaseDir: baseDir,
		Inputs:  l.validationInputs(),
	}
	if err := validator.Validate(wf); err != nil {
		return nil, err
	}

	return wf, nil
}

func (l Loader) loadWorkflow(path string, baseDir string, ctx *workflowLoadContext, paramDisabled bool) (*workflow.Workflow, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve workflow path: %w", err)
	}
	if cycle := ctx.find(absPath); cycle >= 0 {
		chain := append(append([]string{}, ctx.loading[cycle:]...), absPath)
		return nil, fmt.Errorf("subworkflow cycle detected:\n  %s", strings.Join(chain, "\n  "))
	}

	ctx.loading = append(ctx.loading, absPath)
	defer func() {
		ctx.loading = ctx.loading[:len(ctx.loading)-1]
	}()

	workflowLoader := l
	workflowLoader.BaseDir = baseDir
	workflowLoader.paramDisabled = paramDisabled

	session := newLoadSession(workflowLoader, baseDir)
	thread := &starlark.Thread{Name: "workflow"}
	thread.Load = session.loadModule

	globals, err := session.execModule(thread, absPath, true)
	if err != nil {
		return nil, fmt.Errorf("load workflow: %w", err)
	}

	raw, ok := globals["wf"]
	if !ok {
		return nil, fmt.Errorf("workflow file %q does not define top-level wf", path)
	}

	wfValue, ok := raw.(*workflowValue)
	if !ok {
		return nil, fmt.Errorf("top-level wf must be a workflow, got %s", raw.Type())
	}

	if err := workflowLoader.loadSubworkflows(wfValue.workflow, baseDir, ctx); err != nil {
		return nil, err
	}

	return wfValue.workflow, nil
}

func (l Loader) loadSubworkflows(wf *workflow.Workflow, baseDir string, ctx *workflowLoadContext) error {
	for _, node := range wf.Steps {
		if err := l.loadSubworkflowsInNode(node, baseDir, ctx); err != nil {
			return err
		}
	}
	return nil
}

func (l Loader) loadSubworkflowsInNode(node workflow.Node, baseDir string, ctx *workflowLoadContext) error {
	switch n := node.(type) {
	case *workflow.Task:
		return nil
	case *workflow.RepeatUntil:
		for _, child := range n.Steps {
			if err := l.loadSubworkflowsInNode(child, baseDir, ctx); err != nil {
				return err
			}
		}
		return nil
	case *workflow.Subworkflow:
		childPath, err := resolveSubworkflowPath(n.ModuleDir, n.WorkflowPath, baseDir)
		if err != nil {
			return fmt.Errorf("load subworkflow %q: %w", n.ID, err)
		}
		child, err := l.loadWorkflow(childPath, baseDir, ctx, true)
		if err != nil {
			return fmt.Errorf("load subworkflow %q: %w", n.ID, err)
		}
		n.WorkflowPath = childPath
		n.Workflow = child
		return nil
	default:
		return fmt.Errorf("unsupported node type %T", node)
	}
}

func (l Loader) predeclared() starlark.StringDict {
	return starlark.StringDict{
		"workflow":     starlark.NewBuiltin("workflow", l.builtinWorkflow),
		"task":         starlark.NewBuiltin("task", l.builtinTask),
		"repeat_until": starlark.NewBuiltin("repeat_until", l.builtinRepeatUntil),
		"subworkflow":  starlark.NewBuiltin("subworkflow", l.builtinSubworkflow),
		"artifact":     starlark.NewBuiltin("artifact", l.builtinArtifact),
		"path_ref":     starlark.NewBuiltin("path_ref", l.builtinPathRef),
		"json_ref":     starlark.NewBuiltin("json_ref", l.builtinJSONRef),
		"loop_iter":    starlark.NewBuiltin("loop_iter", l.builtinLoopIter),
		"input":        starlark.NewBuiltin("input", l.builtinInput),
		"workdir":      starlark.NewBuiltin("workdir", l.builtinWorkdir),
		"template_file": starlark.NewBuiltin(
			"template_file",
			l.builtinTemplateFile,
		),
		"param":  starlark.NewBuiltin("param", l.builtinParam),
		"format": starlark.NewBuiltin("format", l.builtinFormat),
		"eq":     starlark.NewBuiltin("eq", l.builtinEq),
	}
}

func (l Loader) validationInputs() map[string]string {
	if l.Inputs != nil {
		return l.Inputs
	}
	return l.Params
}

type workflowLoadContext struct {
	loading []string
}

func (c *workflowLoadContext) find(path string) int {
	for i, candidate := range c.loading {
		if candidate == path {
			return i
		}
	}
	return -1
}
