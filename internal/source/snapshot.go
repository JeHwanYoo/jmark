// Package source captures the files used by a benchmark execution.
package source

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

type Snapshot map[string][]byte

func Capture(root string, plan benchmark.Plan) (Snapshot, error) {
	paths := []string{plan.Input, plan.Family.Runner, filepath.Join(root, "benches", plan.Project, "bench.toml"), filepath.Join(root, "runtimes", plan.Runtime.ID+".toml")}
	for _, a := range plan.Approaches {
		paths = append(paths, a.Path)
	}
	for _, p := range plan.Probes {
		paths = append(paths, p.Hook)
	}
	snapshot := Snapshot{}
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		if err != nil || !filepath.IsLocal(rel) {
			return nil, fmt.Errorf("source outside repository: %s", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		snapshot[filepath.ToSlash(rel)] = body
	}
	return snapshot, nil
}

func (s Snapshot) Merge(other Snapshot) error {
	for path, body := range other {
		if previous, ok := s[path]; ok && !bytes.Equal(previous, body) {
			return fmt.Errorf("source changed between runtimes: %s", path)
		}
		s[path] = body
	}
	return nil
}

func (s Snapshot) Verify(root string) error {
	for path, body := range s {
		current, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		if !bytes.Equal(body, current) {
			return fmt.Errorf("source changed during measurement: %s", path)
		}
	}
	return nil
}

func (s Snapshot) Markdown() string {
	var paths []string
	for path := range s {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	var out strings.Builder
	for _, path := range paths {
		body := string(s[path])
		fence := "```"
		for strings.Contains(body, fence) {
			fence += "`"
		}
		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		language := map[string]string{"mjs": "javascript", "cjs": "javascript", "js": "javascript", "py": "python", "rs": "rust", "sh": "bash", "ts": "typescript", "h": "c", "cc": "cpp", "hpp": "cpp", "toml": "toml"}[ext]
		if language == "" {
			language = ext
		}
		title := strings.NewReplacer("\\", "\\\\", "`", "\\`", "[", "\\[", "]", "\\]", "<", "\\<", ">", "\\>", "_", "\\_").Replace(path)
		fmt.Fprintf(&out, "## %s\n\n%s%s\n%s", title, fence, language, body)
		if !strings.HasSuffix(body, "\n") {
			out.WriteByte('\n')
		}
		fmt.Fprintf(&out, "%s\n\n", fence)
	}
	return out.String()
}
