package logging

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestLoggerWhenCheck(t *testing.T) {
	var output bytes.Buffer
	logger := New(&output)
	logger.Now = func() time.Time {
		return time.Date(2026, 4, 9, 12, 0, 1, 0, time.UTC)
	}

	logger.WhenCheck("address_code_issues", "steps")
	logger.WhenCheck("address_code_issues", "else_steps")
	logger.WhenCheck("address_code_issues", "skip")

	for _, want := range []string{
		"[12:00:01] when check id=address_code_issues result=steps",
		"[12:00:01] when check id=address_code_issues result=else_steps",
		"[12:00:01] when check id=address_code_issues result=skip",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("log output missing %q:\n%s", want, output.String())
		}
	}
}
