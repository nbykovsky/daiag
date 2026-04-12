package logging

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

type Logger struct {
	Out io.Writer
	Now func() time.Time
}

func New(out io.Writer) *Logger {
	return &Logger{
		Out: out,
		Now: time.Now,
	}
}

func (l *Logger) WorkflowStart(id, file string) {
	l.printf("workflow start id=%s file=%s", id, file)
}

func (l *Logger) StepStart(id, cli, model string) {
	l.printf("step start id=%s cli=%s model=%s", id, cli, model)
}

func (l *Logger) StepDone(id string, artifacts []string, result map[string]any) {
	fields := []string{
		fmt.Sprintf("step done id=%s", id),
		fmt.Sprintf("artifacts=%s", strings.Join(artifacts, ",")),
	}
	if outcome, ok := result["outcome"]; ok {
		fields = append(fields, fmt.Sprintf("outcome=%v", outcome))
	}
	l.printf("%s", strings.Join(fields, " "))
}

func (l *Logger) SubworkflowStart(id, workflowID string) {
	l.printf("subworkflow start id=%s workflow=%s", id, workflowID)
}

func (l *Logger) SubworkflowDone(id string, artifacts []string, results []string) {
	l.printf("subworkflow done id=%s artifacts=%s results=%s", id, strings.Join(artifacts, ","), strings.Join(results, ","))
}

func (l *Logger) SubworkflowFailed(id, stepID string, err error) {
	if stepID != "" {
		l.printf("subworkflow failed id=%s step=%s error=%s", id, stepID, err)
		return
	}
	l.printf("subworkflow failed id=%s error=%s", id, err)
}

func (l *Logger) LoopIter(id string, n int) {
	l.printf("loop iter id=%s n=%d", id, n)
}

func (l *Logger) LoopCheck(id, result string) {
	l.printf("loop check id=%s result=%s", id, result)
}

func (l *Logger) WhenCheck(id, result string) {
	l.printf("when check id=%s result=%s", id, result)
}

func (l *Logger) WorkflowDone(id string) {
	l.printf("workflow done id=%s status=success", id)
}

func (l *Logger) WorkflowFailed(id, stepID string, err error) {
	if stepID != "" {
		l.printf("workflow failed id=%s step=%s error=%s", id, stepID, err)
		return
	}
	l.printf("workflow failed id=%s error=%s", id, err)
}

func SortKeys(values []string) []string {
	dup := append([]string(nil), values...)
	sort.Strings(dup)
	return dup
}

func (l *Logger) printf(format string, args ...any) {
	timestamp := "00:00:00"
	if l.Now != nil {
		timestamp = l.Now().Format("15:04:05")
	}
	fmt.Fprintf(l.Out, "[%s] %s\n", timestamp, fmt.Sprintf(format, args...))
}
