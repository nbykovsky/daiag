package starlarkdsl

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

var workflowIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func ResolveWorkflowID(workflowsLib string, id string) (string, error) {
	if !workflowIDPattern.MatchString(id) {
		return "", fmt.Errorf("workflow reference %q must be a workflow ID matching [A-Za-z0-9_-]+", id)
	}

	absLib, err := filepath.Abs(workflowsLib)
	if err != nil {
		return "", fmt.Errorf("resolve workflows library: %w", err)
	}

	path := filepath.Join(absLib, id, "workflow.star")
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("workflow %q not found: expected %s", id, path)
		}
		return "", fmt.Errorf("stat workflow %q at %s: %w", id, path, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("workflow %q not found: expected file %s", id, path)
	}

	return path, nil
}
