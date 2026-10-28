package source

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

func TestCaptureVerifyAndMarkdown(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"benches/demo/bench.toml":      "n = 1\n",
		"runtimes/engine.toml":         "family = \"core\"\n",
		"benches/demo/core/_input.src": "input",
		"benches/demo/core/loop.src":   "code\n```\n<script>example</script>\n",
		"families/core/runner.src":     "runner",
	}
	for path, body := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}
	plan := benchmark.Plan{Project: "demo", Runtime: benchmark.PlanRuntime{ID: "engine"}, Input: filepath.Join(root, "benches/demo/core/_input.src"), Family: benchmark.PlanFamily{Runner: filepath.Join(root, "families/core/runner.src")}, Approaches: []benchmark.Approach{{Name: "loop", Path: filepath.Join(root, "benches/demo/core/loop.src")}}}
	snapshot, err := Capture(root, plan)
	if err != nil {
		t.Fatal(err)
	}
	markdown := snapshot.Markdown()
	if !strings.Contains(markdown, "````src\ncode\n```\n<script>example</script>\n````") {
		t.Fatal("code escaped its fence", markdown)
	}
	if !strings.Contains(markdown, "```toml") {
		t.Fatal("missing syntax tag")
	}
	if err := snapshot.Verify(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan.Approaches[0].Path, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Verify(root); err == nil {
		t.Fatal("source change undetected")
	}
	if snapshot.Markdown() != markdown {
		t.Fatal("snapshot followed mutable source")
	}
	newer, err := Capture(root, plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := snapshot.Merge(newer); err == nil {
		t.Fatal("mixed runtime sources accepted")
	}
}
