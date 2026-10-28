package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindRootFromPackageDir(t *testing.T) {
	root, err := findRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"justfile", "cmd/jmark/main.go"} {
		if _, err := os.Stat(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDoctorAndCatalogWithMissingRuntime(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root, map[string]string{
		"families/core/runner.src":          "runner",
		"families/core/template/_input.src": "input",
		"families/core/template/loop.src":   "approach",
		"runtimes/available.toml":           "family = \"core\"\nneeds = \"engine\"\nrun = \"engine {runner} {iters}\"\n",
		"runtimes/unavailable.toml":         "family = \"core\"\nneeds = \"absent-engine\"\nrun = \"absent-engine {runner} {iters}\"\n",
		"benches/demo/AGENTS.md":            "Compare implementations with identical inputs and iteration counts.",
		"benches/demo/bench.toml":           "n = 1\nwarmup = 0\niters = 1\nruns = 1\n",
		"benches/demo/core/_input.src":      "input",
		"benches/demo/core/loop.src":        "approach",
	})
	bin := t.TempDir()
	for _, name := range []string{"just", "bash", "hyperfine", "mise", "engine"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nprintf 'hyperfine 2.0.0\\n'\n"), 0755); err != nil {
			t.Fatal(err)
		}
	}
	// The readings check exports one measurement that reports every collector reading.
	fakeHyperfine := "#!/bin/sh\nwhile [ $# -gt 0 ]; do\n  if [ \"$1\" = --export-json ]; then\n    printf '%s' '" + allReadingsExport + "' > \"$2\"\n    exit 0\n  fi\n  shift\ndone\nprintf 'hyperfine 2.0.0\\n'\n"
	if err := os.WriteFile(filepath.Join(bin, "hyperfine"), []byte(fakeHyperfine), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	out, errOut, code := runOut(t, root, "doctor")
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d: %s %s", code, out, errOut)
	}
	for _, line := range []string{"doctor: metrics available: branch_misses cache_miss_rate cache_misses cache_references cpu_cycles cpu_time instructions ipc peak_rss system_time throughput user_time wall_time", "doctor: registry ok", "doctor: project ok demo (runtimes: available unavailable)", "doctor: runtime available (core) found", "doctor: runtime unavailable (core) missing: absent-engine", "doctor: OK"} {
		if !strings.Contains(out, line+"\n") {
			t.Errorf("missing %q in %s", line, out)
		}
	}
	out, errOut, code = runOut(t, root, "list")
	if code != 0 || errOut != "" || out != "demo\n" {
		t.Fatalf("list: %d %q %q", code, out, errOut)
	}
	out, errOut, code = runOut(t, root, "runtimes")
	if code != 0 || errOut != "" || out != "available\tcore\tfound\nunavailable\tcore\tmissing: absent-engine\n" {
		t.Fatalf("runtimes: %d %q %q", code, out, errOut)
	}
	if err := os.Remove(filepath.Join(bin, "hyperfine")); err != nil {
		t.Fatal(err)
	}
	_, errOut, code = runOut(t, root, "doctor")
	if code == 0 || !strings.Contains(errOut, "missing host command: hyperfine") {
		t.Fatalf("missing host: %d %q", code, errOut)
	}
}

const allReadingsExport = `{"results":[{"measurements":[{"time_wall_clock":{"value":1},"time_cpu":{"value":1},"time_user":{"value":1},"time_system":{"value":0},"memory_peak_resident":{"value":1},"cpu_cycles":{"value":1},"instructions":{"value":1},"cache_references":{"value":1},"cache_misses":{"value":0},"branch_misses":{"value":0},"exit_code":0}]}]}`

func TestCreateScaffold(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root, map[string]string{
		"runtimes/node.toml":                  "family = \"js\"\nneeds = \"node\"\nrun = \"node {runner} {input} {approach} {n} {iters} {probe}\"\n",
		"runtimes/py.toml":                    "family = \"py\"\nneeds = \"python3\"\nrun = \"python3 {runner} {input} {approach} {n} {iters} {probe}\"\n",
		"families/js/template/_input.mjs":     "input\n",
		"families/js/template/approach-a.mjs": "approach\n",
		"families/py/template/_input.py":      "py\n",
		"families/py/template/approach-a.py":  "py-a\n",
	})
	out, errOut, code := runOut(t, root, "create", "verify-x", "py", "node")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, errOut)
	}
	if out != "created benches/verify-x (families: py js)\n" {
		t.Fatalf("stdout %q", out)
	}
	path := filepath.Join(root, "benches", "verify-x", "js", "approach-a.mjs")
	if err := os.WriteFile(path, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out, errOut, code = runOut(t, root, "create", "verify-x", "node", "py")
	if code != 0 {
		t.Fatalf("second exit %d stderr %s", code, errOut)
	}
	if out != "created benches/verify-x (families: js py)\n" {
		t.Fatalf("second stdout %q", out)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "edited\n" {
		t.Fatalf("scaffold overwrote %q", body)
	}
	_, errOut, code = runOut(t, root, "create", "verify-z", "rust")
	if code != 1 || !strings.Contains(errOut, "unknown runtime 'rust' (have: node py )") {
		t.Fatalf("exit %d stderr %q", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, "benches", "verify-z")); !os.IsNotExist(err) {
		t.Fatalf("unknown runtime wrote a project: %v", err)
	}
	_, errOut, code = runOut(t, root, "create", "Foo", "node")
	if code != 1 || !strings.Contains(errOut, "invalid project id 'Foo' (must match ^[a-z0-9][a-z0-9-]*$)") {
		t.Fatalf("exit %d stderr %q", code, errOut)
	}
	_, errOut, code = runOut(t, root, "create")
	if code != 1 || !strings.Contains(errOut, "usage: create <id> <runtime>...") {
		t.Fatalf("exit %d stderr %q", code, errOut)
	}
}

func TestBenchFlagsAndStubs(t *testing.T) {
	root := mustRoot(t)
	cases := []struct {
		args   []string
		code   int
		stderr string
		stdout string
	}{
		{[]string{"bench", "--help"}, 0, "", "usage: bench"},
		{[]string{"bench", "--format", "nope"}, 1, "bench: invalid --format 'nope'", ""},
		{[]string{"bench", "--format="}, 1, "bench: --format needs a value", ""},
		{[]string{"bench", "--runtime"}, 1, "bench: --runtime needs a value", ""},
		{[]string{"bench", "--runtime="}, 1, "bench: --runtime needs a value", ""},
		{[]string{"bench", "--runtime", "a", "--runtime", "b"}, 1, "bench: --runtime given more than once", ""},
		{[]string{"bench", "--weird"}, 1, "bench: unknown option '--weird'", ""},
		{[]string{"bench", "a", "b"}, 1, "bench: unexpected argument 'b'", ""},
		{[]string{"bench"}, 1, "bench: id is required", ""},
		{[]string{"bench", "Bad"}, 1, "invalid project id 'Bad'", ""},
		{[]string{"bench", "missing-project"}, 1, "unknown project 'missing-project' (see: just list)", ""},
		{[]string{"bench", "sum-large-doubles", "--runtime", "rust"}, 1, "unknown runtime 'rust'", ""},
		{[]string{"bench", "sum-large-doubles", "--runtime=rust"}, 1, "unknown runtime 'rust'", ""},
		{[]string{"open", "--help"}, 0, "", "usage: open"},
		{[]string{"open", "--help", "x"}, 0, "", "usage: open"},
		{[]string{"open", "a", "b"}, 1, "usage: open [<id>]", ""},
		{[]string{"delete", "-h"}, 0, "", "usage: delete"},
		{[]string{"delete", "a", "b"}, 1, "usage: delete [<id>]", ""},
		{[]string{"nope"}, 1, "usage: doctor", ""},
	}
	for _, tc := range cases {
		out, errOut, code := runOut(t, root, tc.args...)
		if code != tc.code {
			t.Errorf("%v exit %d want %d\nstderr %s\nstdout %s", tc.args, code, tc.code, errOut, out)
			continue
		}
		if tc.stderr != "" && !strings.Contains(errOut, tc.stderr) {
			t.Errorf("%v stderr %q missing %q", tc.args, errOut, tc.stderr)
		}
		if len(tc.args) == 1 && tc.args[0] == "bench" && !strings.Contains(errOut, "usage: bench") {
			t.Errorf("bench without an id did not print usage on stderr:\n%s", errOut)
		}
		if tc.stdout != "" && !strings.Contains(out, tc.stdout) {
			t.Errorf("%v stdout %q missing %q", tc.args, out, tc.stdout)
		}
		if tc.code != 0 && out != "" && tc.stdout == "" {
			t.Errorf("%v wrote stdout %q", tc.args, out)
		}
	}
}

func TestWhitespaceRoot(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "has space")
	if err := os.MkdirAll(filepath.Join(dir, "runtimes"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "families"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "justfile"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	_, errOut, code := runOut(t, dir, "list")
	if code != 1 || !strings.Contains(errOut, "repo path must not contain whitespace") {
		t.Fatalf("exit %d stderr %q", code, errOut)
	}
}

func TestWriteArtifactsHTML(t *testing.T) {
	root := t.TempDir()
	a := &app{root: root, stdout: os.Stdout, stderr: os.Stderr}
	var obj map[string]any
	_ = json.Unmarshal([]byte(`{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"us_per_op":{"label":"Wall time","unit":"\u00b5s/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"a","sink":3,"metrics":{"us_per_op":{"status":"ok","value":1.5}},"observations":{},"statistics":{}}],"meta":{}}`), &obj)
	obj["meta"].(map[string]any)["host"] = map[string]any{"os": "Darwin", "arch": "arm64"}
	obj["meta"].(map[string]any)["runtime"] = "v1"
	obj["meta"].(map[string]any)["captured_at"] = "2026-10-27T00:00:00Z"
	line, err := marshalCompact(obj)
	if err != nil {
		t.Fatal(err)
	}
	rel, err := a.writeArtifacts("demo", "html", [][]byte{line})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, "artifacts/private/demo/") || !strings.HasSuffix(rel, "/report.html") {
		t.Fatalf("path %s", rel)
	}
	html, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	text := string(html)
	if strings.Contains(text, "report pending") {
		t.Fatalf("stub report still present")
	}
	for _, needle := range []string{`id="jmark-data"`, `id="app"`, `"name":"a"`} {
		if !strings.Contains(text, needle) {
			t.Errorf("missing %s", needle)
		}
	}
	dir := filepath.Dir(filepath.Join(root, rel))
	nd, err := os.ReadFile(filepath.Join(dir, "result.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	if string(nd) != string(line)+"\n" {
		t.Fatalf("ndjson %s", nd)
	}
	envRaw, err := os.ReadFile(filepath.Join(dir, "env.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err := json.Unmarshal(envRaw, &env); err != nil {
		t.Fatal(err)
	}
	if env["captured_at"] != "2026-10-27T00:00:00Z" {
		t.Fatalf("captured_at %#v", env["captured_at"])
	}
	runtimes, _ := env["runtimes"].(map[string]any)
	if runtimes["node"] != "v1" {
		t.Fatalf("runtimes %#v", env["runtimes"])
	}
	host, _ := env["host"].(map[string]any)
	if host["os"] != "Darwin" {
		t.Fatalf("host %#v", env["host"])
	}
	ndRel, err := a.writeArtifacts("demo", "ndjson", [][]byte{line})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(ndRel, "/result.ndjson") {
		t.Fatalf("ndjson path %s", ndRel)
	}
}

func TestSinkDiff(t *testing.T) {
	same := []map[string]any{
		{"runtime": "node", "results": []any{map[string]any{"sink": float64(1950000)}}},
		{"runtime": "py", "results": []any{map[string]any{"sink": float64(1950000)}}},
	}
	if sinkDiff(same) != "" {
		t.Fatalf("same %q", sinkDiff(same))
	}
	diff := []map[string]any{
		{"runtime": "node", "results": []any{map[string]any{"sink": float64(1)}}},
		{"runtime": "py", "results": []any{map[string]any{"sink": float64(2)}}},
	}
	if sinkDiff(diff) != "node=1 py=2" {
		t.Fatalf("diff %q", sinkDiff(diff))
	}
}

func TestRuntimeMissingLine(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root, map[string]string{
		"runtimes/zz.toml": "family = \"js\"\nneeds = \"definitely-not-a-command-zz\"\nrun = \"zz {iters}\"\n",
	})
	out, errOut, code := runOut(t, root, "runtimes")
	if code != 0 {
		t.Fatalf("exit %d stderr %s", code, errOut)
	}
	if out != "zz\tjs\tmissing: definitely-not-a-command-zz\n" {
		t.Fatalf("stdout %q", out)
	}
}

func runOut(t *testing.T, root string, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut strings.Builder
	code := run(args, nil, &out, &errOut, root)
	return out.String(), errOut.String(), code
}

func mustRoot(t *testing.T) string {
	t.Helper()
	root, err := findRoot()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func writeRepo(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMetricSelection(t *testing.T) {
	for _, args := range [][]string{{"demo", "--metrics", "peak_rss"}, {"demo", "--metrics=wall_time,throughput"}} {
		o, e := parseBench(args)
		if e != nil || len(o.metrics) == 0 {
			t.Fatalf("%v: %v", args, e)
		}
	}
	for _, args := range [][]string{{"demo", "--metrics"}, {"demo", "--metrics="}, {"demo", "--metrics", "unknown"}, {"demo", "--metrics=peak_rss", "--metrics=wall_time"}} {
		if _, e := parseBench(args); e == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
