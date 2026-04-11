package starlarkdsl

import (
	"fmt"
	"path/filepath"

	"daiag/internal/workflow"

	"go.starlark.net/starlark"
)

type Loader struct {
	Params  map[string]string
	Inputs  map[string]string
	BaseDir string
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

	session := newLoadSession(l, baseDir)
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

	validator := workflow.Validator{
		BaseDir: baseDir,
		Inputs:  l.validationInputs(),
	}
	if err := validator.Validate(wfValue.workflow); err != nil {
		return nil, err
	}

	return wfValue.workflow, nil
}

func (l Loader) predeclared() starlark.StringDict {
	return starlark.StringDict{
		"workflow":     starlark.NewBuiltin("workflow", l.builtinWorkflow),
		"task":         starlark.NewBuiltin("task", l.builtinTask),
		"repeat_until": starlark.NewBuiltin("repeat_until", l.builtinRepeatUntil),
		"artifact":     starlark.NewBuiltin("artifact", l.builtinArtifact),
		"path_ref":     starlark.NewBuiltin("path_ref", l.builtinPathRef),
		"json_ref":     starlark.NewBuiltin("json_ref", l.builtinJSONRef),
		"loop_iter":    starlark.NewBuiltin("loop_iter", l.builtinLoopIter),
		"input":        starlark.NewBuiltin("input", l.builtinInput),
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
