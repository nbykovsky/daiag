package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var placeholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

type Validator struct {
	BaseDir string
}

type taskInfo struct {
	artifacts  map[string]struct{}
	resultKeys map[string]struct{}
}

func (v Validator) Validate(wf *Workflow) error {
	if wf == nil {
		return fmt.Errorf("workflow is nil")
	}
	if wf.ID == "" {
		return fmt.Errorf("workflow ID is empty")
	}

	seenTasks := make(map[string]taskInfo)
	allIDs := make(map[string]struct{})

	_, err := v.validateSteps(wf.Steps, seenTasks, allIDs, wf.DefaultExecutor, map[string]struct{}{})
	if err != nil {
		return err
	}

	return nil
}

func (v Validator) validateSteps(steps []Node, seenTasks map[string]taskInfo, allIDs map[string]struct{}, defaultExecutor *ExecutorConfig, activeLoops map[string]struct{}) (map[string]taskInfo, error) {
	current := cloneTaskInfoMap(seenTasks)

	for _, node := range steps {
		if node == nil {
			return nil, fmt.Errorf("workflow contains nil step")
		}
		if node.NodeID() == "" {
			return nil, fmt.Errorf("step ID is empty")
		}
		if _, ok := allIDs[node.NodeID()]; ok {
			return nil, fmt.Errorf("duplicate step ID %q", node.NodeID())
		}
		allIDs[node.NodeID()] = struct{}{}

		switch n := node.(type) {
		case *Task:
			if err := v.validateTask(n, current, defaultExecutor, activeLoops); err != nil {
				return nil, fmt.Errorf("task %q: %w", n.ID, err)
			}
			current[n.ID] = taskInfo{
				artifacts:  artifactKeySet(n.Artifacts),
				resultKeys: stringKeySet(n.ResultKeys),
			}
		case *RepeatUntil:
			if n.MaxIters < 1 {
				return nil, fmt.Errorf("repeat_until %q: max_iters must be at least 1", n.ID)
			}
			loopScope := cloneStringSet(activeLoops)
			loopScope[n.ID] = struct{}{}
			loopSeen, err := v.validateSteps(n.Steps, current, allIDs, defaultExecutor, loopScope)
			if err != nil {
				return nil, err
			}
			if err := v.validatePredicate(n.Until, loopSeen, loopScope); err != nil {
				return nil, fmt.Errorf("repeat_until %q: %w", n.ID, err)
			}
			current = loopSeen
		default:
			return nil, fmt.Errorf("unsupported node type %T", node)
		}
	}

	return current, nil
}

func (v Validator) validateTask(task *Task, seenTasks map[string]taskInfo, defaultExecutor *ExecutorConfig, activeLoops map[string]struct{}) error {
	if task.Prompt.TemplatePath == "" && task.Prompt.Inline == "" {
		return fmt.Errorf("prompt is required")
	}
	if len(task.Artifacts) == 0 {
		return fmt.Errorf("artifacts are required")
	}
	if len(task.ResultKeys) == 0 {
		return fmt.Errorf("result_keys are required")
	}

	resolvedExecutor := task.Executor
	if resolvedExecutor == nil {
		resolvedExecutor = defaultExecutor
	}
	if resolvedExecutor == nil {
		return fmt.Errorf("executor is required")
	}
	if resolvedExecutor.CLI == "" || resolvedExecutor.Model == "" {
		return fmt.Errorf("executor must include cli and model")
	}

	seenResultKeys := make(map[string]struct{}, len(task.ResultKeys))
	for _, key := range task.ResultKeys {
		if key == "" {
			return fmt.Errorf("result_keys must not contain empty values")
		}
		if _, ok := seenResultKeys[key]; ok {
			return fmt.Errorf("result_keys must be unique")
		}
		seenResultKeys[key] = struct{}{}
	}

	if task.Prompt.TemplatePath != "" {
		if err := v.validatePromptTemplate(task.Prompt); err != nil {
			return err
		}
	}
	for name, expr := range task.Prompt.Vars {
		if name == "" {
			return fmt.Errorf("prompt vars must not contain empty keys")
		}
		if err := validateStringExpr(expr, seenTasks, activeLoops); err != nil {
			return fmt.Errorf("prompt var %q: %w", name, err)
		}
	}

	for key, expr := range task.Artifacts {
		if key == "" {
			return fmt.Errorf("artifacts must not contain empty keys")
		}
		if err := validateStringExpr(expr, seenTasks, activeLoops); err != nil {
			return fmt.Errorf("artifact %q: %w", key, err)
		}
	}

	return nil
}

func (v Validator) validatePromptTemplate(prompt Prompt) error {
	path := prompt.TemplatePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(v.BaseDir, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read prompt template %q: %w", prompt.TemplatePath, err)
	}

	matches := placeholderPattern.FindAllStringSubmatch(string(data), -1)
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		name := match[1]
		if _, done := seen[name]; done {
			continue
		}
		seen[name] = struct{}{}
		if _, ok := prompt.Vars[name]; !ok {
			return fmt.Errorf("missing prompt variable %q for template %q", name, prompt.TemplatePath)
		}
	}

	return nil
}

func (v Validator) validatePredicate(predicate Predicate, seenTasks map[string]taskInfo, activeLoops map[string]struct{}) error {
	switch p := predicate.(type) {
	case EqPredicate:
		if err := validateValueExpr(p.Left, seenTasks, activeLoops); err != nil {
			return err
		}
		if err := validateValueExpr(p.Right, seenTasks, activeLoops); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported predicate type %T", predicate)
	}
}

func validateStringExpr(expr StringExpr, seenTasks map[string]taskInfo, activeLoops map[string]struct{}) error {
	switch e := expr.(type) {
	case Literal:
		if e.Value == "" {
			return fmt.Errorf("value must not be empty")
		}
		return nil
	case FormatExpr:
		if e.Template == "" {
			return fmt.Errorf("template must not be empty")
		}
		for name, value := range e.Args {
			if name == "" {
				return fmt.Errorf("format args must not contain empty keys")
			}
			if err := validateValueExpr(value, seenTasks, activeLoops); err != nil {
				return fmt.Errorf("format arg %q: %w", name, err)
			}
		}
		return nil
	case PathRef:
		info, ok := seenTasks[e.StepID]
		if !ok {
			return fmt.Errorf("unknown step %q", e.StepID)
		}
		if _, ok := info.artifacts[e.ArtifactKey]; !ok {
			return fmt.Errorf("step %q does not declare artifact %q", e.StepID, e.ArtifactKey)
		}
		return nil
	default:
		return fmt.Errorf("unsupported string expression type %T", expr)
	}
}

func validateValueExpr(expr ValueExpr, seenTasks map[string]taskInfo, activeLoops map[string]struct{}) error {
	switch e := expr.(type) {
	case Literal:
		if e.Value == "" {
			return fmt.Errorf("value must not be empty")
		}
		return nil
	case IntLiteral:
		return nil
	case FormatExpr:
		return validateStringExpr(e, seenTasks, activeLoops)
	case PathRef:
		return validateStringExpr(e, seenTasks, activeLoops)
	case JSONRef:
		info, ok := seenTasks[e.StepID]
		if !ok {
			return fmt.Errorf("unknown step %q", e.StepID)
		}
		if e.Field == "" {
			return fmt.Errorf("json ref field must not be empty")
		}
		if _, ok := info.resultKeys[e.Field]; !ok {
			return fmt.Errorf("step %q does not declare result key %q", e.StepID, e.Field)
		}
		return nil
	case LoopIter:
		if e.LoopID == "" {
			return fmt.Errorf("loop_iter loop ID must not be empty")
		}
		if _, ok := activeLoops[e.LoopID]; !ok {
			return fmt.Errorf("loop %q is not active in this scope", e.LoopID)
		}
		return nil
	default:
		return fmt.Errorf("unsupported value expression type %T", expr)
	}
}

func artifactKeySet(artifacts map[string]StringExpr) map[string]struct{} {
	keys := make(map[string]struct{}, len(artifacts))
	for key := range artifacts {
		keys[key] = struct{}{}
	}
	return keys
}

func cloneTaskInfoMap(src map[string]taskInfo) map[string]taskInfo {
	dst := make(map[string]taskInfo, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func stringKeySet(values []string) map[string]struct{} {
	keys := make(map[string]struct{}, len(values))
	for _, value := range values {
		keys[value] = struct{}{}
	}
	return keys
}

func cloneStringSet(src map[string]struct{}) map[string]struct{} {
	dst := make(map[string]struct{}, len(src))
	for key := range src {
		dst[key] = struct{}{}
	}
	return dst
}
