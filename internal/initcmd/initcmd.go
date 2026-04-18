package initcmd

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

//go:embed templates
var templates embed.FS

// Config controls what daiag init creates.
type Config struct {
	ProjectDir string   // target directory; empty means CWD
	Workflows  []string // empty means all bundled workflows
	Force      bool     // overwrite existing .daiag
}

// Init creates a .daiag directory in the target project and populates it with
// bundled workflow templates, skills, and agents.
func Init(cfg Config, stdout io.Writer) error {
	target, err := resolveTargetDir(cfg.ProjectDir)
	if err != nil {
		return err
	}
	daiagDir := filepath.Join(target, ".daiag")

	if _, err := os.Stat(daiagDir); err == nil && !cfg.Force {
		return fmt.Errorf(".daiag already exists at %q; use --force to overwrite", daiagDir)
	}

	wfSet, err := resolveWorkflows(cfg.Workflows)
	if err != nil {
		return err
	}

	for _, dir := range []string{
		filepath.Join(daiagDir, "workflows"),
		filepath.Join(daiagDir, "skills"),
		filepath.Join(daiagDir, "agents"),
		filepath.Join(daiagDir, "runs"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
	}

	if err := copyDir("templates/skills", filepath.Join(daiagDir, "skills")); err != nil {
		return err
	}
	if err := copyDir("templates/agents", filepath.Join(daiagDir, "agents")); err != nil {
		return err
	}
	if err := copyFile("templates/workflows/WORKFLOWS.md", filepath.Join(daiagDir, "workflows", "WORKFLOWS.md")); err != nil {
		return err
	}

	for _, wf := range wfSet {
		src := "templates/workflows/" + wf
		dst := filepath.Join(daiagDir, "workflows", wf)
		if err := copyDir(src, dst); err != nil {
			return err
		}
	}

	fmt.Fprintf(stdout, "initialized .daiag in %s\n", target)
	fmt.Fprintf(stdout, "workflows: %v\n", wfSet)
	return nil
}

// AvailableWorkflows returns the IDs of all bundled workflow templates.
func AvailableWorkflows() []string {
	entries, err := templates.ReadDir("templates/workflows")
	if err != nil {
		return nil
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids
}

func resolveTargetDir(path string) (string, error) {
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get current directory: %w", err)
		}
		return cwd, nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve --projectdir: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("--projectdir %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("--projectdir %q is not a directory", abs)
	}
	return abs, nil
}

func resolveWorkflows(requested []string) ([]string, error) {
	all := AvailableWorkflows()
	if len(requested) == 0 {
		return all, nil
	}
	available := make(map[string]bool, len(all))
	for _, id := range all {
		available[id] = true
	}
	for _, id := range requested {
		if !available[id] {
			return nil, fmt.Errorf("%q is not a bundled workflow; available: %v", id, all)
		}
	}
	return requested, nil
}

func copyDir(srcFS string, dst string) error {
	return fs.WalkDir(templates, srcFS, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcFS, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(srcFS string, dst string) error {
	data, err := templates.ReadFile(srcFS)
	if err != nil {
		return fmt.Errorf("read template %q: %w", srcFS, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent for %q: %w", dst, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %q: %w", dst, err)
	}
	return nil
}
