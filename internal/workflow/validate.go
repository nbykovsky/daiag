package workflow

import (
	"fmt"
	"os"
	"regexp"
)

var placeholderPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)

type Validator struct {
	BaseDir string
	Inputs  map[string]string
}

type nodeInfo struct {
	artifacts  map[string]struct{}
	resultKeys map[string]struct{}
}

func (v Validator) Validate(wf *Workflow) error {
	return v.validateWorkflow(wf, v.BaseDir)
}

func (v Validator) validateWorkflow(wf *Workflow, templateBaseDir string) error {
	if wf == nil {
		return fmt.Errorf("workflow is nil")
	}
	if wf.ID == "" {
		return fmt.Errorf("workflow ID is empty")
	}

	declaredInputs, err := v.validateInputs(wf.Inputs)
	if err != nil {
		return err
	}

	seenNodes := make(map[string]nodeInfo)
	allIDs := make(map[string]struct{})

	finalScope, err := v.validateSteps(wf.Steps, seenNodes, allIDs, wf.DefaultExecutor, map[string]struct{}{}, declaredInputs, templateBaseDir)
	if err != nil {
		return err
	}
	if err := v.validateWorkflowOutputs(wf, finalScope, declaredInputs); err != nil {
		return err
	}

	return nil
}

func (v Validator) validateInputs(inputs []string) (map[string]struct{}, error) {
	declared := make(map[string]struct{}, len(inputs))
	for _, name := range inputs {
		if name == "" {
			return nil, fmt.Errorf("workflow inputs must not contain empty values")
		}
		if _, ok := declared[name]; ok {
			return nil, fmt.Errorf("workflow inputs must be unique")
		}
		declared[name] = struct{}{}
	}
	return declared, nil
}

func (v Validator) validateSteps(steps []Node, seenNodes map[string]nodeInfo, allIDs map[string]struct{}, defaultExecutor *ExecutorConfig, activeLoops map[string]struct{}, declaredInputs map[string]struct{}, templateBaseDir string) (map[string]nodeInfo, error) {
	current := cloneNodeInfoMap(seenNodes)

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
			if err := v.validateTask(n, current, defaultExecutor, activeLoops, declaredInputs, templateBaseDir); err != nil {
				return nil, fmt.Errorf("task %q: %w", n.ID, err)
			}
			current[n.ID] = nodeInfo{
				artifacts:  artifactKeySet(n.Artifacts),
				resultKeys: stringKeySet(n.ResultKeys),
			}
		case *RepeatUntil:
			if n.MaxIters < 1 {
				return nil, fmt.Errorf("repeat_until %q: max_iters must be at least 1", n.ID)
			}
			loopScope := cloneStringSet(activeLoops)
			loopScope[n.ID] = struct{}{}
			loopSeen, err := v.validateSteps(n.Steps, current, allIDs, defaultExecutor, loopScope, declaredInputs, templateBaseDir)
			if err != nil {
				return nil, err
			}
			if err := v.validatePredicate(n.Until, loopSeen, loopScope, declaredInputs); err != nil {
				return nil, fmt.Errorf("repeat_until %q: %w", n.ID, err)
			}
			current = loopSeen
		case *When:
			if err := v.validatePredicate(n.Condition, current, activeLoops, declaredInputs); err != nil {
				return nil, fmt.Errorf("when %q: %w", n.ID, err)
			}
			if _, err := v.validateSteps(n.Steps, current, allIDs, defaultExecutor, activeLoops, declaredInputs, templateBaseDir); err != nil {
				return nil, err
			}
			if _, err := v.validateSteps(n.ElseSteps, current, allIDs, defaultExecutor, activeLoops, declaredInputs, templateBaseDir); err != nil {
				return nil, err
			}
		case *Subworkflow:
			if err := v.validateSubworkflow(n, current, activeLoops, declaredInputs); err != nil {
				return nil, fmt.Errorf("subworkflow %q: %w", n.ID, err)
			}
			current[n.ID] = nodeInfo{
				artifacts:  artifactKeySet(n.Workflow.OutputArtifacts),
				resultKeys: valueExprKeySet(n.Workflow.OutputResults),
			}
		default:
			return nil, fmt.Errorf("unsupported node type %T", node)
		}
	}

	return current, nil
}

func (v Validator) validateTask(task *Task, seenNodes map[string]nodeInfo, defaultExecutor *ExecutorConfig, activeLoops map[string]struct{}, declaredInputs map[string]struct{}, templateBaseDir string) error {
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
		if err := v.validatePromptTemplate(task.Prompt, templateBaseDir); err != nil {
			return err
		}
	}
	for name, expr := range task.Prompt.Vars {
		if name == "" {
			return fmt.Errorf("prompt vars must not contain empty keys")
		}
		if err := validateStringExpr(expr, seenNodes, activeLoops, declaredInputs); err != nil {
			return fmt.Errorf("prompt var %q: %w", name, err)
		}
	}

	for key, expr := range task.Artifacts {
		if key == "" {
			return fmt.Errorf("artifacts must not contain empty keys")
		}
		if err := validateStringExpr(expr, seenNodes, activeLoops, declaredInputs); err != nil {
			return fmt.Errorf("artifact %q: %w", key, err)
		}
	}

	return nil
}

func (v Validator) validateSubworkflow(sub *Subworkflow, parentNodes map[string]nodeInfo, parentActiveLoops map[string]struct{}, parentInputs map[string]struct{}) error {
	if sub.WorkflowPath == "" {
		return fmt.Errorf("workflow path is empty")
	}
	if sub.Workflow == nil {
		return fmt.Errorf("workflow is not loaded")
	}

	childInputs := stringKeySet(sub.Workflow.Inputs)
	if err := v.validateWorkflow(sub.Workflow, v.BaseDir); err != nil {
		return err
	}

	for key, expr := range sub.Inputs {
		if _, ok := childInputs[key]; !ok {
			return fmt.Errorf("unknown input binding %q", key)
		}
		if err := validateValueExpr(expr, parentNodes, parentActiveLoops, parentInputs); err != nil {
			return fmt.Errorf("input %q: %w", key, err)
		}
	}
	for key := range childInputs {
		if _, ok := sub.Inputs[key]; !ok {
			return fmt.Errorf("missing input binding %q", key)
		}
	}

	return nil
}

func (v Validator) validateWorkflowOutputs(wf *Workflow, seenNodes map[string]nodeInfo, declaredInputs map[string]struct{}) error {
	for key, expr := range wf.OutputArtifacts {
		if key == "" {
			return fmt.Errorf("output_artifacts must not contain empty keys")
		}
		if err := validateStringExpr(expr, seenNodes, map[string]struct{}{}, declaredInputs); err != nil {
			return fmt.Errorf("output_artifacts %q: %w", key, err)
		}
	}
	for key, expr := range wf.OutputResults {
		if key == "" {
			return fmt.Errorf("output_results must not contain empty keys")
		}
		if err := validateValueExpr(expr, seenNodes, map[string]struct{}{}, declaredInputs); err != nil {
			return fmt.Errorf("output_results %q: %w", key, err)
		}
	}
	return nil
}

func (v Validator) validatePromptTemplate(prompt Prompt, templateBaseDir string) error {
	path := prompt.ResolvedTemplatePath(templateBaseDir)
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

func (v Validator) validatePredicate(predicate Predicate, seenNodes map[string]nodeInfo, activeLoops map[string]struct{}, declaredInputs map[string]struct{}) error {
	switch p := predicate.(type) {
	case EqPredicate:
		if err := validateValueExpr(p.Left, seenNodes, activeLoops, declaredInputs); err != nil {
			return err
		}
		if err := validateValueExpr(p.Right, seenNodes, activeLoops, declaredInputs); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported predicate type %T", predicate)
	}
}

func validateStringExpr(expr StringExpr, seenNodes map[string]nodeInfo, activeLoops map[string]struct{}, declaredInputs map[string]struct{}) error {
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
			if err := validateValueExpr(value, seenNodes, activeLoops, declaredInputs); err != nil {
				return fmt.Errorf("format arg %q: %w", name, err)
			}
		}
		return nil
	case PathRef:
		info, ok := seenNodes[e.StepID]
		if !ok {
			return fmt.Errorf("unknown step %q", e.StepID)
		}
		if _, ok := info.artifacts[e.ArtifactKey]; !ok {
			return fmt.Errorf("step %q does not declare artifact %q", e.StepID, e.ArtifactKey)
		}
		return nil
	case InputRef:
		if e.Name == "" {
			return fmt.Errorf("input name must not be empty")
		}
		if _, ok := declaredInputs[e.Name]; !ok {
			return fmt.Errorf("unknown workflow input %q", e.Name)
		}
		return nil
	case RunDirRef:
		return nil
	case ProjectDirRef:
		return nil
	default:
		return fmt.Errorf("unsupported string expression type %T", expr)
	}
}

func validateValueExpr(expr ValueExpr, seenNodes map[string]nodeInfo, activeLoops map[string]struct{}, declaredInputs map[string]struct{}) error {
	switch e := expr.(type) {
	case Literal:
		if e.Value == "" {
			return fmt.Errorf("value must not be empty")
		}
		return nil
	case IntLiteral:
		return nil
	case FormatExpr:
		return validateStringExpr(e, seenNodes, activeLoops, declaredInputs)
	case PathRef:
		return validateStringExpr(e, seenNodes, activeLoops, declaredInputs)
	case JSONRef:
		info, ok := seenNodes[e.StepID]
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
	case InputRef:
		return validateStringExpr(e, seenNodes, activeLoops, declaredInputs)
	case RunDirRef:
		return nil
	case ProjectDirRef:
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

func valueExprKeySet(values map[string]ValueExpr) map[string]struct{} {
	keys := make(map[string]struct{}, len(values))
	for key := range values {
		keys[key] = struct{}{}
	}
	return keys
}

func cloneNodeInfoMap(src map[string]nodeInfo) map[string]nodeInfo {
	dst := make(map[string]nodeInfo, len(src))
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
