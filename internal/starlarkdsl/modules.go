package starlarkdsl

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type loadSession struct {
	loader      Loader
	baseDir     string
	moduleCache map[string]starlark.StringDict
	loading     []string
}

func newLoadSession(loader Loader, baseDir string) *loadSession {
	return &loadSession{
		loader:      loader,
		baseDir:     baseDir,
		moduleCache: make(map[string]starlark.StringDict),
	}
}

func (s *loadSession) loadModule(thread *starlark.Thread, module string) (starlark.StringDict, error) {
	importerPath, err := currentModulePath(thread)
	if err != nil {
		return nil, err
	}

	modulePath, err := s.resolveModulePath(importerPath, module)
	if err != nil {
		return nil, err
	}

	return s.execModule(thread, modulePath, false)
}

func (s *loadSession) execModule(thread *starlark.Thread, path string, entry bool) (starlark.StringDict, error) {
	if globals, ok := s.moduleCache[path]; ok {
		return globals, nil
	}
	if cycle := s.findLoading(path); cycle >= 0 {
		chain := append(append([]string{}, s.loading[cycle:]...), path)
		return nil, fmt.Errorf("load cycle detected:\n  %s", strings.Join(chain, "\n  "))
	}

	s.loading = append(s.loading, path)
	defer func() {
		s.loading = s.loading[:len(s.loading)-1]
	}()

	globals, err := starlark.ExecFileOptions(&syntax.FileOptions{}, thread, path, nil, s.loader.predeclared())
	if err != nil {
		return nil, err
	}
	if !entry {
		if _, ok := globals["wf"]; ok {
			return nil, fmt.Errorf("loaded module %q must not define top-level wf", path)
		}
	}

	s.moduleCache[path] = globals
	return globals, nil
}

func (s *loadSession) resolveModulePath(importerPath string, module string) (string, error) {
	if module == "" {
		return "", fmt.Errorf("load path must not be empty")
	}
	if !strings.HasSuffix(module, ".star") {
		return "", fmt.Errorf("load path %q must end with .star", module)
	}
	if strings.Contains(module, "://") {
		return "", fmt.Errorf("unsupported load path %q", module)
	}

	path := module
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(importerPath), path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve load path %q: %w", module, err)
	}
	if err := ensureWithinBase(absPath, s.baseDir); err != nil {
		return "", fmt.Errorf("resolve load path %q: %w", module, err)
	}
	if _, err := os.Stat(absPath); err != nil {
		return "", fmt.Errorf("stat %q: %w", absPath, err)
	}

	return absPath, nil
}

func currentModulePath(thread *starlark.Thread) (string, error) {
	return modulePathAtDepth(thread, 0)
}

func currentCallerModulePath(thread *starlark.Thread) (string, error) {
	return modulePathAtDepth(thread, 1)
}

func modulePathAtDepth(thread *starlark.Thread, depth int) (string, error) {
	if thread.CallStackDepth() <= depth {
		return "", fmt.Errorf("cannot resolve load path without an importing module")
	}
	return thread.CallFrame(depth).Pos.Filename(), nil
}

func ensureWithinBase(path string, baseDir string) error {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return fmt.Errorf("resolve relative path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("path %q escapes base directory %q", path, baseDir)
	}
	return nil
}

func (s *loadSession) findLoading(path string) int {
	for i, candidate := range s.loading {
		if candidate == path {
			return i
		}
	}
	return -1
}
