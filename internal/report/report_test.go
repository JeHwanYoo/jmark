package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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
	for _, marker := range []string{"@@PROJECT@@", "@@DESCRIPTION@@", "@@CANONICAL@@", "@@DATA_ID@@", "@@INDEX_HREF@@", "@@SOURCES@@", "@@DATA@@", "@@JS@@", "@@CSS@@"} {
		if strings.Contains(page, marker) {
			t.Fatalf("unresolved template marker: %s", marker)
		}
	}
}

func TestReportDescribesResults(t *testing.T) {
	page, err := RenderReport([]byte(parityNDJSON))
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`<meta name="description" content="([^"]*)">`).FindStringSubmatch(page)
	if match == nil || !strings.HasPrefix(match[1], "demo benchmark comparing ") {
		t.Fatalf("description: %q", match)
	}
	if strings.Contains(page, `rel="canonical"`) {
		t.Fatal("local report claims a canonical URL")
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
	if err := os.WriteFile(path, []byte(`{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"wall_time":{"label":"wall_time","unit":"wall_time","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[],"meta":{}}`+"\n"), 0o644); err != nil {
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

const parityNDJSON = `{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":10},"warmup":1,"iters":2,"runs":3},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"for-loop","sink":9,"metrics":{"us_per_op":{"status":"ok","value":1.25}},"observations":{},"statistics":{}},{"name":"map","sink":9,"metrics":{"us_per_op":{"status":"ok","value":4}},"observations":{},"statistics":{}}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T00:00:00Z","runtime":"</script>","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}
{"schema_version":2,"project":"demo","runtime":"py","workload":{"parameters":{"n":10},"warmup":1,"iters":2,"runs":3},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"for-loop","sink":9,"metrics":{"us_per_op":{"status":"ok","value":5}},"observations":{},"statistics":{}},{"name":"map","sink":9,"metrics":{"us_per_op":{"status":"ok","value":6.5}},"observations":{},"statistics":{}}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T00:00:01Z","runtime":"3.12","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}`

const parityNDJSON2 = `{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":10},"warmup":1,"iters":2,"runs":3},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"for-loop","sink":9,"metrics":{"us_per_op":{"status":"ok","value":2}},"observations":{},"statistics":{}}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T01:00:00Z","runtime":"v1","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}`

const envNDJSON = `{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"a","sink":1,"metrics":{"us_per_op":{"status":"ok","value":1}},"observations":{},"statistics":{}}],"meta":{"captured_at":"","runtime":"v1","host":{}}}
{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"a","sink":1,"metrics":{"us_per_op":{"status":"ok","value":1}},"observations":{},"statistics":{}}],"meta":{"captured_at":"2026-10-27T00:00:00Z","runtime":"</script>","host":{"os":"Darwin","ncpu":2}}}
{"schema_version":2,"project":"demo","runtime":"py","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"a","sink":1,"metrics":{"us_per_op":{"status":"ok","value":1}},"observations":{},"statistics":{}}],"meta":{"runtime":null,"host":{}}}
{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"a","sink":1,"metrics":{"us_per_op":{"status":"ok","value":1}},"observations":{},"statistics":{}}],"meta":{"captured_at":"2026-10-27T00:00:01Z","runtime":"v2","host":{"os":"Linux"}}}`

func TestMalformedNDJSONRejectedBeforeWriting(t *testing.T) {
	for _, input := range []string{parityNDJSON + "]", parityNDJSON + "}", parityNDJSON + "garbage"} {
		if _, err := RenderReport([]byte(input)); err == nil {
			t.Fatal("accepted malformed NDJSON")
		}

	}
}

func TestOldResultsRejected(t *testing.T) {
	if _, err := RenderReport([]byte(`{"project":"old","runtime":"node","metric":"us_per_op","results":[]}`)); err == nil {
		t.Fatal("legacy result accepted")
	}
}

func TestLegacyHostnameIsNotExported(t *testing.T) {
	input := []byte(`{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"time":{"label":"time","unit":"us/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"loop","sink":0,"metrics":{"time":{"status":"ok","value":1}},"observations":{},"statistics":{}}],"meta":{"host":{"os":"Darwin","hostname":"private-machine"}}}`)
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

func TestWorkloadMetadataProjection(t *testing.T) {
	raw := []byte(`{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"size":64,"distribution":"uniform"},"warmup":0,"iters":1,"runs":1},"metrics":{"throughput":{"label":"throughput","unit":"items/s","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"loop","sink":0,"metrics":{"throughput":{"status":"ok","value":20}},"observations":{},"statistics":{}}],"meta":{"parameters":{"size":64,"distribution":"uniform"},"unit":"items/s"}}`)
	rows, _, err := flatten(raw, "")
	if err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(rows[0], &row); err != nil {
		t.Fatal(err)
	}
	if row["unit"] != "items/s" || row["parameters"].(map[string]any)["size"] != float64(64) {
		t.Fatalf("metadata lost: %s", rows[0])
	}
}
