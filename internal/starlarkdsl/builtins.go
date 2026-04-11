package starlarkdsl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"daiag/internal/workflow"

	"go.starlark.net/starlark"
)

func (l Loader) builtinWorkflow(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var id string
	var inputsValue starlark.Value = starlark.None
	var stepsValue starlark.Value
	var defaultExecutorValue starlark.Value = starlark.None
	var outputArtifactsValue starlark.Value = starlark.None
	var outputResultsValue starlark.Value = starlark.None

	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"id", &id,
		"inputs?", &inputsValue,
		"steps", &stepsValue,
		"default_executor?", &defaultExecutorValue,
		"output_artifacts?", &outputArtifactsValue,
		"output_results?", &outputResultsValue,
	); err != nil {
		return nil, err
	}

	inputs, err := unpackOptionalStringList(inputsValue, "inputs")
	if err != nil {
		return nil, err
	}

	steps, err := unpackSteps(stepsValue)
	if err != nil {
		return nil, err
	}

	defaultExecutor, err := unpackOptionalExecutor(defaultExecutorValue)
	if err != nil {
		return nil, err
	}

	outputArtifacts, err := unpackOptionalStringExprMap(outputArtifactsValue, "output_artifacts")
	if err != nil {
		return nil, err
	}
	outputResults, err := unpackOptionalValueExprMap(outputResultsValue, "output_results")
	if err != nil {
		return nil, err
	}

	return &workflowValue{
		workflow: &workflow.Workflow{
			ID:              id,
			Inputs:          inputs,
			DefaultExecutor: defaultExecutor,
			Steps:           steps,
			OutputArtifacts: outputArtifacts,
			OutputResults:   outputResults,
		},
	}, nil
}

func (l Loader) builtinTask(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var id string
	var promptValue starlark.Value
	var artifactsValue starlark.Value
	var resultKeysValue starlark.Value
	var executorValue starlark.Value = starlark.None

	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"id", &id,
		"prompt", &promptValue,
		"artifacts", &artifactsValue,
		"result_keys", &resultKeysValue,
		"executor?", &executorValue,
	); err != nil {
		return nil, err
	}

	prompt, err := unpackPrompt(promptValue)
	if err != nil {
		return nil, err
	}
	artifacts, err := unpackArtifacts(artifactsValue)
	if err != nil {
		return nil, err
	}
	resultKeys, err := unpackStringList(resultKeysValue, "result_keys")
	if err != nil {
		return nil, err
	}
	executor, err := unpackOptionalExecutor(executorValue)
	if err != nil {
		return nil, err
	}

	return &taskValue{
		task: &workflow.Task{
			ID:         id,
			Prompt:     prompt,
			Executor:   executor,
			Artifacts:  artifacts,
			ResultKeys: resultKeys,
		},
	}, nil
}

func (l Loader) builtinRepeatUntil(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var id string
	var stepsValue starlark.Value
	var untilValue starlark.Value
	var maxIters int

	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"id", &id,
		"steps", &stepsValue,
		"until", &untilValue,
		"max_iters", &maxIters,
	); err != nil {
		return nil, err
	}

	steps, err := unpackSteps(stepsValue)
	if err != nil {
		return nil, err
	}
	predicate, err := unpackPredicate(untilValue)
	if err != nil {
		return nil, err
	}

	return &repeatUntilValue{
		loop: &workflow.RepeatUntil{
			ID:       id,
			Steps:    steps,
			Until:    predicate,
			MaxIters: maxIters,
		},
	}, nil
}

func (l Loader) builtinSubworkflow(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var id string
	var workflowID string
	var inputsValue starlark.Value = starlark.None

	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"id", &id,
		"workflow", &workflowID,
		"inputs?", &inputsValue,
	); err != nil {
		return nil, err
	}

	inputs, err := unpackOptionalValueExprMap(inputsValue, "inputs")
	if err != nil {
		return nil, err
	}
	if inputs == nil {
		inputs = map[string]workflow.ValueExpr{}
	}

	return &subworkflowValue{
		subworkflow: &workflow.Subworkflow{
			ID:           id,
			WorkflowPath: workflowID,
			Inputs:       inputs,
		},
	}, nil
}

func (l Loader) builtinArtifact(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var raw starlark.Value
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs, "path", &raw); err != nil {
		return nil, err
	}
	expr, err := unpackStringExpr(raw)
	if err != nil {
		return nil, err
	}
	return &artifactValue{expr: expr}, nil
}

func (l Loader) builtinPathRef(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var stepID string
	var artifactKey string
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"step_id", &stepID,
		"artifact_key", &artifactKey,
	); err != nil {
		return nil, err
	}
	return &pathRefValue{ref: workflow.PathRef{StepID: stepID, ArtifactKey: artifactKey}}, nil
}

func (l Loader) builtinJSONRef(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var stepID string
	var field string
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"step_id", &stepID,
		"field", &field,
	); err != nil {
		return nil, err
	}
	return &jsonRefValue{ref: workflow.JSONRef{StepID: stepID, Field: field}}, nil
}

func (l Loader) builtinLoopIter(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var loopID string
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs, "loop_id", &loopID); err != nil {
		return nil, err
	}
	return &loopIterValue{ref: workflow.LoopIter{LoopID: loopID}}, nil
}

func (l Loader) builtinInput(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	return &inputValue{ref: workflow.InputRef{Name: name}}, nil
}

func (l Loader) builtinWorkdir(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs); err != nil {
		return nil, err
	}
	return &workdirValue{}, nil
}

func (l Loader) builtinProjectdir(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs); err != nil {
		return nil, err
	}
	modulePath, err := currentCallerModulePath(thread)
	if err != nil {
		return nil, err
	}
	projectDir, err := findProjectDir(modulePath)
	if err != nil {
		return nil, err
	}
	return starlark.String(projectDir), nil
}

func (l Loader) builtinTemplateFile(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var path string
	var varsValue starlark.Value = starlark.None

	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"path", &path,
		"vars?", &varsValue,
	); err != nil {
		return nil, err
	}

	vars, err := unpackPromptVars(varsValue)
	if err != nil {
		return nil, err
	}

	modulePath, err := currentCallerModulePath(thread)
	if err != nil {
		return nil, err
	}

	return &promptTemplateValue{
		prompt: workflow.Prompt{
			TemplatePath: path,
			TemplateDir:  filepath.Dir(modulePath),
			Vars:         vars,
		},
	}, nil
}

func (l Loader) builtinParam(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
	}
	if l.paramDisabled {
		return nil, fmt.Errorf("param(%q) is not allowed in subworkflows; declare input(%q)", name, name)
	}
	value, ok := l.Params[name]
	if !ok {
		return nil, fmt.Errorf("missing workflow param %q", name)
	}
	return starlark.String(value), nil
}

func (l Loader) builtinFormat(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("format: expected template string")
	}
	template, ok := starlark.AsString(args[0])
	if !ok {
		return nil, fmt.Errorf("format: template must be a string")
	}

	replacements := make(map[string]workflow.ValueExpr, len(kwargs))
	for _, item := range kwargs {
		if len(item) != 2 {
			return nil, fmt.Errorf("format: invalid keyword argument")
		}
		name, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("format: keyword names must be strings")
		}
		value, err := unpackValueExpr(item[1])
		if err != nil {
			return nil, fmt.Errorf("format: keyword %q: %w", name, err)
		}
		replacements[name] = value
	}

	return &formatValue{
		expr: workflow.FormatExpr{
			Template: template,
			Args:     replacements,
		},
	}, nil
}

func (l Loader) builtinEq(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var leftValue starlark.Value
	var rightValue starlark.Value

	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"left", &leftValue,
		"right", &rightValue,
	); err != nil {
		return nil, err
	}

	left, err := unpackValueExpr(leftValue)
	if err != nil {
		return nil, err
	}
	right, err := unpackValueExpr(rightValue)
	if err != nil {
		return nil, err
	}

	return &predicateValue{
		predicate: workflow.EqPredicate{Left: left, Right: right},
	}, nil
}

func unpackSteps(value starlark.Value) ([]workflow.Node, error) {
	list, ok := value.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("steps must be a list")
	}

	steps := make([]workflow.Node, 0, list.Len())
	for i := 0; i < list.Len(); i++ {
		item := list.Index(i)
		switch value := item.(type) {
		case *taskValue:
			steps = append(steps, value.task)
		case *repeatUntilValue:
			steps = append(steps, value.loop)
		case *subworkflowValue:
			steps = append(steps, value.subworkflow)
		default:
			return nil, fmt.Errorf("steps[%d] must be a task, repeat_until, or subworkflow, got %s", i, item.Type())
		}
	}
	return steps, nil
}

func unpackPrompt(value starlark.Value) (workflow.Prompt, error) {
	switch v := value.(type) {
	case starlark.String:
		return workflow.Prompt{Inline: v.GoString()}, nil
	case *promptTemplateValue:
		return v.prompt, nil
	default:
		return workflow.Prompt{}, fmt.Errorf("prompt must be a string or template_file, got %s", value.Type())
	}
}

func unpackArtifacts(value starlark.Value) (map[string]workflow.StringExpr, error) {
	dict, ok := value.(*starlark.Dict)
	if !ok {
		return nil, fmt.Errorf("artifacts must be a dict")
	}

	artifacts := make(map[string]workflow.StringExpr, dict.Len())
	for _, item := range dict.Items() {
		key, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("artifact keys must be strings")
		}
		artifact, ok := item[1].(*artifactValue)
		if !ok {
			return nil, fmt.Errorf("artifact %q must be declared with artifact(...)", key)
		}
		artifacts[key] = artifact.expr
	}

	return artifacts, nil
}

func unpackPromptVars(value starlark.Value) (map[string]workflow.StringExpr, error) {
	if value == starlark.None {
		return map[string]workflow.StringExpr{}, nil
	}

	dict, ok := value.(*starlark.Dict)
	if !ok {
		return nil, fmt.Errorf("template vars must be a dict")
	}

	vars := make(map[string]workflow.StringExpr, dict.Len())
	for _, item := range dict.Items() {
		key, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("template var keys must be strings")
		}
		expr, err := unpackStringExpr(item[1])
		if err != nil {
			return nil, fmt.Errorf("template var %q: %w", key, err)
		}
		vars[key] = expr
	}
	return vars, nil
}

func unpackStringList(value starlark.Value, field string) ([]string, error) {
	list, ok := value.(*starlark.List)
	if !ok {
		return nil, fmt.Errorf("%s must be a list", field)
	}

	values := make([]string, 0, list.Len())
	for i := 0; i < list.Len(); i++ {
		item, ok := starlark.AsString(list.Index(i))
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be a string", field, i)
		}
		values = append(values, item)
	}
	return values, nil
}

func unpackOptionalStringList(value starlark.Value, field string) ([]string, error) {
	if value == starlark.None {
		return nil, nil
	}
	return unpackStringList(value, field)
}

func unpackOptionalStringExprMap(value starlark.Value, field string) (map[string]workflow.StringExpr, error) {
	if value == starlark.None {
		return nil, nil
	}

	dict, ok := value.(*starlark.Dict)
	if !ok {
		return nil, fmt.Errorf("%s must be a dict", field)
	}

	exprs := make(map[string]workflow.StringExpr, dict.Len())
	for _, item := range dict.Items() {
		key, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("%s keys must be strings", field)
		}
		expr, err := unpackStringExpr(item[1])
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", field, key, err)
		}
		exprs[key] = expr
	}
	return exprs, nil
}

func unpackOptionalValueExprMap(value starlark.Value, field string) (map[string]workflow.ValueExpr, error) {
	if value == starlark.None {
		return nil, nil
	}

	dict, ok := value.(*starlark.Dict)
	if !ok {
		return nil, fmt.Errorf("%s must be a dict", field)
	}

	exprs := make(map[string]workflow.ValueExpr, dict.Len())
	for _, item := range dict.Items() {
		key, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("%s keys must be strings", field)
		}
		expr, err := unpackValueExpr(item[1])
		if err != nil {
			return nil, fmt.Errorf("%s %q: %w", field, key, err)
		}
		exprs[key] = expr
	}
	return exprs, nil
}

func unpackOptionalExecutor(value starlark.Value) (*workflow.ExecutorConfig, error) {
	if value == starlark.None {
		return nil, nil
	}

	dict, ok := value.(*starlark.Dict)
	if !ok {
		return nil, fmt.Errorf("executor must be a dict")
	}

	executor := &workflow.ExecutorConfig{}
	for _, item := range dict.Items() {
		key, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("executor keys must be strings")
		}
		val, ok := starlark.AsString(item[1])
		if !ok {
			return nil, fmt.Errorf("executor %q must be a string", key)
		}
		switch key {
		case "cli":
			executor.CLI = val
		case "model":
			executor.Model = val
		default:
			return nil, fmt.Errorf("unsupported executor key %q", key)
		}
	}

	return executor, nil
}

func unpackPredicate(value starlark.Value) (workflow.Predicate, error) {
	predicate, ok := value.(*predicateValue)
	if !ok {
		return nil, fmt.Errorf("until must be a predicate")
	}
	return predicate.predicate, nil
}

func unpackStringExpr(value starlark.Value) (workflow.StringExpr, error) {
	switch v := value.(type) {
	case starlark.String:
		return workflow.Literal{Value: v.GoString()}, nil
	case *formatValue:
		return v.expr, nil
	case *pathRefValue:
		return v.ref, nil
	case *inputValue:
		return v.ref, nil
	case *workdirValue:
		return workflow.WorkdirRef{}, nil
	default:
		return nil, fmt.Errorf("expected string, format, path_ref, input, or workdir, got %s", value.Type())
	}
}

func unpackValueExpr(value starlark.Value) (workflow.ValueExpr, error) {
	switch v := value.(type) {
	case starlark.String:
		return workflow.Literal{Value: v.GoString()}, nil
	case starlark.Int:
		intValue, ok := v.Int64()
		if !ok {
			return nil, fmt.Errorf("expected int in range, got %s", v.String())
		}
		return workflow.IntLiteral{Value: int(intValue)}, nil
	case *formatValue:
		return v.expr, nil
	case *pathRefValue:
		return v.ref, nil
	case *jsonRefValue:
		return v.ref, nil
	case *loopIterValue:
		return v.ref, nil
	case *inputValue:
		return v.ref, nil
	case *workdirValue:
		return workflow.WorkdirRef{}, nil
	default:
		return nil, fmt.Errorf("expected string, int, format, path_ref, json_ref, loop_iter, input, or workdir, got %s", value.Type())
	}
}

func findProjectDir(modulePath string) (string, error) {
	dir := filepath.Dir(modulePath)
	walked := []string{}
	for {
		walked = append(walked, dir)
		daiagDir := filepath.Join(dir, ".daiag")
		info, err := os.Stat(daiagDir)
		if err == nil {
			if info.IsDir() {
				return dir, nil
			}
			return "", fmt.Errorf("projectdir() called from %q: %q is not a directory", modulePath, daiagDir)
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("projectdir() called from %q: stat %q: %w", modulePath, daiagDir, err)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("projectdir() called from %q but no .daiag directory found; walked: %s; pass the project path as an explicit workflow input instead", modulePath, strings.Join(walked, ", "))
		}
		dir = parent
	}
}
