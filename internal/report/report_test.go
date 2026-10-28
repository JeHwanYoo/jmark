package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRenderReportPayload(t *testing.T) {
	page, err := RenderReport([]byte(parityNDJSON))
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(scriptData(page)), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 4 || rows[0]["runtime_version"] != "</script>" || rows[0]["value"] != 1.25 {
		t.Fatalf("payload: %v", rows)
	}
	if strings.Contains(scriptData(page), "<") {
		t.Fatal("unsafe script payload")
	}
	for _, marker := range []string{"@@PROJECT@@", "@@DATA_ID@@", "@@INDEX_HREF@@", "@@SOURCES@@", "@@DATA@@", "@@JS@@", "@@CSS@@"} {
		if strings.Contains(page, marker) {
			t.Fatalf("unresolved template marker: %s", marker)
		}
	}
}

func TestReportEscapesHTML(t *testing.T) {
	input := strings.ReplaceAll(parityNDJSON, "demo", "</title><img src=x onerror=alert(1)>")
	page, err := RenderReport([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(page, "<img src=x") {
		t.Fatal("HTML injection")
	}
	input = strings.ReplaceAll(parityNDJSON, "demo", "<!--<script>")
	page, err = RenderReport([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(scriptData(page), "<") {
		t.Fatal("script escaped-state injection")
	}
}

func TestRenderCompareShape(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.ndjson")
	b := filepath.Join(dir, "b.ndjson")
	if err := os.WriteFile(a, []byte(parityNDJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte(parityNDJSON2), 0o644); err != nil {
		t.Fatal(err)
	}
	goHTML, err := RenderCompare([]CompareRun{
		{Label: "demo/one", Path: a},
		{Label: "demo/two", Path: b},
		{Label: "demo/missing", Path: filepath.Join(dir, "nope.ndjson")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`id="jmark-data"`, "demo/one", "demo/two", "for-loop"} {
		if !strings.Contains(goHTML, needle) {
			t.Fatalf("missing %q in compare html", needle)
		}
	}
}

func TestRenderCompareEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.ndjson")
	if err := os.WriteFile(path, []byte(`{"project":"demo","results":[]}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RenderCompare([]CompareRun{{Label: "demo/t", Path: path}}); err == nil {
		t.Fatal("expected no results")
	}
	if _, err := RenderCompare(nil); err == nil {
		t.Fatal("expected no results for no runs")
	}
}

func TestEnvJSONMatchesGolden(t *testing.T) {
	want, err := os.ReadFile(filepath.Join(repoRoot(t), "internal", "report", "testdata", "env.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Environment([]byte(envNDJSON))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("env.json\n got %s\nwant %s", got, want)
	}
}

func TestOpenerDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		return
	}
	bin, err := findOpener()
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" && bin != "/usr/bin/open" {
		t.Fatalf("opener %s", bin)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "justfile")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root")
		}
		dir = parent
	}
}

func scriptData(html string) string {
	const start = `<script type="application/json" id="jmark-data">`
	i := strings.Index(html, start)
	if i < 0 {
		return ""
	}
	rest := html[i+len(start):]
	j := strings.Index(rest, "</script>")
	if j < 0 {
		return rest
	}
	return rest[:j]
}

const parityNDJSON = `{"project":"demo","runtime":"node","metric":"us_per_op","results":[{"name":"for-loop","value":1.25,"iters":2,"total_ms":3.5,"sink":9},{"name":"map","value":4,"iters":2,"total_ms":8,"sink":9}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T00:00:00Z","runtime":"</script>","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}
{"project":"demo","runtime":"py","metric":"us_per_op","results":[{"name":"for-loop","value":5,"iters":2,"total_ms":10,"sink":9},{"name":"map","value":6.5,"iters":2,"total_ms":13,"sink":9}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T00:00:01Z","runtime":"3.12","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}
`

const parityNDJSON2 = `{"project":"demo","runtime":"node","metric":"us_per_op","results":[{"name":"for-loop","value":2,"iters":2,"total_ms":4,"sink":9}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T01:00:00Z","runtime":"v1","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}
`

const envNDJSON = `{"project":"demo","runtime":"node","metric":"us_per_op","results":[{"name":"a","value":1,"iters":1,"total_ms":1,"sink":1}],"meta":{"captured_at":"","runtime":"v1","host":{}}}
{"project":"demo","runtime":"node","metric":"us_per_op","results":[{"name":"a","value":1,"iters":1,"total_ms":1,"sink":1}],"meta":{"captured_at":"2026-10-27T00:00:00Z","runtime":"</script>","host":{"os":"Darwin","ncpu":2}}}
{"project":"demo","runtime":"py","metric":"us_per_op","results":[{"name":"a","value":1,"iters":1,"total_ms":1,"sink":1}],"meta":{"runtime":null,"host":{}}}
{"project":"demo","runtime":"node","metric":"us_per_op","results":[{"name":"a","value":1,"iters":1,"total_ms":1,"sink":1}],"meta":{"captured_at":"2026-10-27T00:00:01Z","runtime":"v2","host":{"os":"Linux"}}}
`

func TestMalformedNDJSONRejectedBeforeWriting(t *testing.T) {
	for _, input := range []string{parityNDJSON + "]", parityNDJSON + "}", parityNDJSON + "garbage"} {
		if _, err := RenderReport([]byte(input)); err == nil {
			t.Fatal("accepted malformed NDJSON")
		}

	}
}

func TestLegacyRuntimeReports(t *testing.T) {
	legacy := strings.NewReplacer(`"runtime":"node"`, `"lang":"node"`, `"runtime":"py"`, `"lang":"py"`).Replace(parityNDJSON)
	oldPage, err := RenderReport([]byte(legacy))
	if err != nil {
		t.Fatal(err)
	}
	newPage, err := RenderReport([]byte(parityNDJSON))
	if err != nil {
		t.Fatal(err)
	}
	if oldPage != newPage {
		t.Fatal("legacy runtime IDs or versions changed during normalization")
	}
	oldEnv, err := Environment([]byte(legacy))
	if err != nil {
		t.Fatal(err)
	}
	newEnv, err := Environment([]byte(parityNDJSON))
	if err != nil {
		t.Fatal(err)
	}
	if string(oldEnv) != string(newEnv) {
		t.Fatal("legacy environment changed")
	}
	dir := t.TempDir()
	oldPath, newPath := filepath.Join(dir, "old.ndjson"), filepath.Join(dir, "new.ndjson")
	if err := os.WriteFile(oldPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte(parityNDJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	page, err := RenderCompare([]CompareRun{{Label: "old", Path: oldPath}, {Label: "new", Path: newPath}})
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(scriptData(page)), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 8 {
		t.Fatalf("rows = %d", len(rows))
	}
	for _, row := range rows {
		if _, old := row["lang"]; old || row["runtime"] == nil || row["runtime_version"] == nil {
			t.Fatalf("bad normalized row: %v", row)
		}
	}
}

func TestLegacyHostnameIsNotExported(t *testing.T) {
	input := []byte(`{"project":"demo","runtime":"node","metric":"time","unit":"us/op","n":1,"warmup":1,"iters":1,"runs":1,"meta":{"host":{"os":"Darwin","hostname":"private-machine"}},"results":[{"name":"loop","value":1,"total_ms":1,"iters":1}]}`)
	html, err := RenderReport(input)
	if err != nil {
		t.Fatal(err)
	}
	env, err := Environment(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{html, string(env)} {
		if strings.Contains(output, "private-machine") || strings.Contains(output, `"hostname"`) {
			t.Fatal("hostname leaked")
		}
		if !strings.Contains(output, "Darwin") {
			t.Fatal("OS metadata lost")
		}
	}
}
