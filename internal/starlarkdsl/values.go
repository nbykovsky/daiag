package starlarkdsl

import (
	"fmt"
	"sort"
	"strings"

	"daiag/internal/workflow"

	"go.starlark.net/starlark"
)

type workflowValue struct {
	workflow *workflow.Workflow
}

func (*workflowValue) Type() string          { return "workflow" }
func (*workflowValue) Freeze()               {}
func (*workflowValue) Truth() starlark.Bool  { return starlark.True }
func (*workflowValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: workflow") }

func (w *workflowValue) String() string {
	return fmt.Sprintf("workflow(id=%q)", w.workflow.ID)
}

type taskValue struct {
	task *workflow.Task
}

func (*taskValue) Type() string          { return "task" }
func (*taskValue) Freeze()               {}
func (*taskValue) Truth() starlark.Bool  { return starlark.True }
func (*taskValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: task") }

func (t *taskValue) String() string {
	return fmt.Sprintf("task(id=%q)", t.task.ID)
}

type repeatUntilValue struct {
	loop *workflow.RepeatUntil
}

func (*repeatUntilValue) Type() string          { return "repeat_until" }
func (*repeatUntilValue) Freeze()               {}
func (*repeatUntilValue) Truth() starlark.Bool  { return starlark.True }
func (*repeatUntilValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: repeat_until") }

func (r *repeatUntilValue) String() string {
	return fmt.Sprintf("repeat_until(id=%q)", r.loop.ID)
}

type artifactValue struct {
	expr workflow.StringExpr
}

func (*artifactValue) Type() string          { return "artifact" }
func (*artifactValue) Freeze()               {}
func (*artifactValue) Truth() starlark.Bool  { return starlark.True }
func (*artifactValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: artifact") }

func (a *artifactValue) String() string {
	return fmt.Sprintf("artifact(%s)", exprString(a.expr))
}

type pathRefValue struct {
	ref workflow.PathRef
}

func (*pathRefValue) Type() string          { return "path_ref" }
func (*pathRefValue) Freeze()               {}
func (*pathRefValue) Truth() starlark.Bool  { return starlark.True }
func (*pathRefValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: path_ref") }

func (p *pathRefValue) String() string {
	return fmt.Sprintf("path_ref(step=%q, artifact=%q)", p.ref.StepID, p.ref.ArtifactKey)
}

type jsonRefValue struct {
	ref workflow.JSONRef
}

func (*jsonRefValue) Type() string          { return "json_ref" }
func (*jsonRefValue) Freeze()               {}
func (*jsonRefValue) Truth() starlark.Bool  { return starlark.True }
func (*jsonRefValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: json_ref") }

func (j *jsonRefValue) String() string {
	return fmt.Sprintf("json_ref(step=%q, field=%q)", j.ref.StepID, j.ref.Field)
}

type promptTemplateValue struct {
	prompt workflow.Prompt
}

func (*promptTemplateValue) Type() string          { return "template_file" }
func (*promptTemplateValue) Freeze()               {}
func (*promptTemplateValue) Truth() starlark.Bool  { return starlark.True }
func (*promptTemplateValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: template_file") }

func (p *promptTemplateValue) String() string {
	keys := make([]string, 0, len(p.prompt.Vars))
	for key := range p.prompt.Vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return fmt.Sprintf("template_file(path=%q, vars=%s)", p.prompt.TemplatePath, strings.Join(keys, ","))
}

type predicateValue struct {
	predicate workflow.Predicate
}

func (*predicateValue) Type() string          { return "predicate" }
func (*predicateValue) Freeze()               {}
func (*predicateValue) Truth() starlark.Bool  { return starlark.True }
func (*predicateValue) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable: predicate") }

func (p *predicateValue) String() string {
	return "predicate(eq)"
}

func exprString(expr workflow.ValueExpr) string {
	switch e := expr.(type) {
	case workflow.Literal:
		return fmt.Sprintf("%q", e.Value)
	case workflow.PathRef:
		return fmt.Sprintf("path_ref(%q, %q)", e.StepID, e.ArtifactKey)
	case workflow.JSONRef:
		return fmt.Sprintf("json_ref(%q, %q)", e.StepID, e.Field)
	default:
		return fmt.Sprintf("%T", expr)
	}
}
