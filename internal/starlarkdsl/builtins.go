package starlarkdsl

import (
	"fmt"
	"strings"

	"daiag/internal/workflow"

	"go.starlark.net/starlark"
)

func (l Loader) builtinWorkflow(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var id string
	var stepsValue starlark.Value
	var defaultExecutorValue starlark.Value = starlark.None

	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs,
		"id", &id,
		"steps", &stepsValue,
		"default_executor?", &defaultExecutorValue,
	); err != nil {
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

	return &workflowValue{
		workflow: &workflow.Workflow{
			ID:              id,
			DefaultExecutor: defaultExecutor,
			Steps:           steps,
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

func (l Loader) builtinTemplateFile(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
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

	return &promptTemplateValue{
		prompt: workflow.Prompt{
			TemplatePath: path,
			Vars:         vars,
		},
	}, nil
}

func (l Loader) builtinParam(_ *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	var name string
	if err := starlark.UnpackArgs(builtin.Name(), args, kwargs, "name", &name); err != nil {
		return nil, err
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

	replacements := make(map[string]string, len(kwargs))
	for _, item := range kwargs {
		if len(item) != 2 {
			return nil, fmt.Errorf("format: invalid keyword argument")
		}
		name, ok := starlark.AsString(item[0])
		if !ok {
			return nil, fmt.Errorf("format: keyword names must be strings")
		}
		value, ok := starlark.AsString(item[1])
		if !ok {
			return nil, fmt.Errorf("format: keyword %q must be a string", name)
		}
		replacements[name] = value
	}

	var builder strings.Builder
	for i := 0; i < len(template); {
		if template[i] != '{' {
			builder.WriteByte(template[i])
			i++
			continue
		}
		end := strings.IndexByte(template[i:], '}')
		if end <= 1 {
			return nil, fmt.Errorf("format: malformed placeholder in %q", template)
		}
		end += i
		name := template[i+1 : end]
		value, ok := replacements[name]
		if !ok {
			return nil, fmt.Errorf("format: missing value for %q", name)
		}
		builder.WriteString(value)
		i = end + 1
	}

	return starlark.String(builder.String()), nil
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
		default:
			return nil, fmt.Errorf("steps[%d] must be a task or repeat_until, got %s", i, item.Type())
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
	case *pathRefValue:
		return v.ref, nil
	default:
		return nil, fmt.Errorf("expected string or path_ref, got %s", value.Type())
	}
}

func unpackValueExpr(value starlark.Value) (workflow.ValueExpr, error) {
	switch v := value.(type) {
	case starlark.String:
		return workflow.Literal{Value: v.GoString()}, nil
	case *pathRefValue:
		return v.ref, nil
	case *jsonRefValue:
		return v.ref, nil
	default:
		return nil, fmt.Errorf("expected string, path_ref, or json_ref, got %s", value.Type())
	}
}
