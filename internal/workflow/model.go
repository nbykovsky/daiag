package workflow

type Workflow struct {
	ID              string
	DefaultExecutor *ExecutorConfig
	Steps           []Node
}

type ExecutorConfig struct {
	CLI   string
	Model string
}

type Node interface {
	node()
	NodeID() string
}

type Task struct {
	ID         string
	Prompt     Prompt
	Executor   *ExecutorConfig
	Artifacts  map[string]StringExpr
	ResultKeys []string
}

func (*Task) node() {}

func (t *Task) NodeID() string {
	return t.ID
}

type RepeatUntil struct {
	ID       string
	Steps    []Node
	Until    Predicate
	MaxIters int
}

func (*RepeatUntil) node() {}

func (r *RepeatUntil) NodeID() string {
	return r.ID
}

type Prompt struct {
	Inline       string
	TemplatePath string
	Vars         map[string]StringExpr
}

func (p Prompt) IsInline() bool {
	return p.TemplatePath == ""
}

type ValueExpr interface {
	valueExpr()
}

type StringExpr interface {
	ValueExpr
	stringExpr()
}

type Literal struct {
	Value string
}

func (Literal) valueExpr() {}

func (Literal) stringExpr() {}

type IntLiteral struct {
	Value int
}

func (IntLiteral) valueExpr() {}

type PathRef struct {
	StepID      string
	ArtifactKey string
}

func (PathRef) valueExpr() {}

func (PathRef) stringExpr() {}

type JSONRef struct {
	StepID string
	Field  string
}

func (JSONRef) valueExpr() {}

type LoopIter struct {
	LoopID string
}

func (LoopIter) valueExpr() {}

type FormatExpr struct {
	Template string
	Args     map[string]ValueExpr
}

func (FormatExpr) valueExpr() {}

func (FormatExpr) stringExpr() {}

type Predicate interface {
	predicate()
}

type EqPredicate struct {
	Left  ValueExpr
	Right ValueExpr
}

func (EqPredicate) predicate() {}
