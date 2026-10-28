package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/catalog"
)

const sampleResult = `{"schema_version":2,"project":"demo","runtime":"node","workload":{"parameters":{"n":1},"warmup":0,"iters":1,"runs":1},"metrics":{"us_per_op":{"label":"Wall time","unit":"µs/op","direction":"lower","method":"difference_per_op","scope":"operation"}},"results":[{"name":"loop","sink":1,"metrics":{"us_per_op":{"status":"ok","value":1.5}},"observations":{},"statistics":{}}],"meta":{"n":1,"warmup":0,"iters":1,"runs":1,"captured_at":"2026-10-27T00:00:00Z","runtime":"v1","host":{"os":"Darwin"}}}` + "\n"

func TestSplitExpect(t *testing.T) {
	key, lines := splitExpect("\nfoo\n")
	if key != "" || strings.Join(lines, ",") != "foo" {
		t.Fatalf("enter %q %v", key, lines)
	}
	key, lines = splitExpect("q\n")
	if key != "q" || len(lines) != 0 {
		t.Fatalf("quit %q %v", key, lines)
	}
	key, lines = splitExpect(openDeleteKey + "\na/1\nb/2\n")
	if key != openDeleteKey || strings.Join(lines, ",") != "a/1,b/2" {
		t.Fatalf("multi %q %v", key, lines)
	}
	key, lines = splitExpect("")
	if key != "" || lines != nil {
		t.Fatalf("empty %q %v", key, lines)
	}
}

func TestFzfStatus(t *testing.T) {
	if err := fzfStatus(exec.Command("bash", "-c", "exit 1").Run()); !errors.Is(err, errSoftQuit) {
		t.Fatalf("exit 1: %v", err)
	}
	if err := fzfStatus(exec.Command("bash", "-c", "exit 130").Run()); !errors.Is(err, errSoftQuit) {
		t.Fatalf("exit 130: %v", err)
	}
	err := fzfStatus(exec.Command("bash", "-c", "exit 2").Run())
	if errors.Is(err, errSoftQuit) || !strings.Contains(err.Error(), "fzf failed (exit 2)") {
		t.Fatalf("exit 2: %v", err)
	}
	if fzfStatus(nil) != nil {
		t.Fatal("nil status")
	}
}

func TestOpenHelpAndMissing(t *testing.T) {
	root := t.TempDir()
	out, errOut, code := runOut(t, root, "open", "--help")
	if code != 0 || out != openUsage || errOut != "" {
		t.Fatalf("help exit %d\n%s\n%s", code, out, errOut)
	}
	_, errOut, code = runOut(t, root, "open")
	if code != 1 || !strings.Contains(errOut, "no artifacts/ yet") {
		t.Fatalf("exit %d %s", code, errOut)
	}
	if err := os.MkdirAll(filepath.Join(root, "artifacts"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, errOut, code = runOut(t, root, "open", "Bad")
	if code != 1 || !strings.Contains(errOut, "invalid project id 'Bad'") {
		t.Fatalf("exit %d %s", code, errOut)
	}
	_, errOut, code = runOut(t, root, "delete", "demo")
	if code != 1 || !strings.Contains(errOut, "no artifacts for demo") {
		t.Fatalf("exit %d %s", code, errOut)
	}
}

func TestBenchPickerSoftQuit(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root, map[string]string{
		"benches/demo/AGENTS.md": "demo\n",
	})
	yes := true
	var got string
	var args []string
	a := &app{
		root: root,
		tty:  &yes,
		fzf: func(input string, fzfArgs []string) (string, []string, error) {
			got = input
			args = fzfArgs
			return "q", nil, nil
		},
	}
	out, errOut, code := finishApp(t, a, "bench")
	if code != 0 || out != "" || errOut != "jmark: quit\n" {
		t.Fatalf("exit %d stdout %q stderr %q", code, out, errOut)
	}
	if got != "demo\n" {
		t.Fatalf("picker input %q", got)
	}
	joined := strings.Join(args, "\n")
	if !strings.Contains(joined, "--expect=q") || !strings.Contains(joined, "AGENTS.md") || !strings.Contains(joined, "--prompt=jmark bench> ") {
		t.Fatalf("args\n%s", joined)
	}

	a.fzf = func(string, []string) (string, []string, error) {
		return "", []string{"nope"}, nil
	}
	_, errOut, code = finishApp(t, a, "bench", "--format", "html")
	if code != 1 || !strings.Contains(errOut, "unknown project 'nope'") {
		t.Fatalf("exit %d %s", code, errOut)
	}

	called := false
	a.fzf = func(string, []string) (string, []string, error) {
		called = true
		return "", nil, errSoftQuit
	}
	empty := &app{root: t.TempDir(), tty: &yes, fzf: a.fzf}
	_, errOut, code = finishApp(t, empty, "bench")
	if called || code != 1 || !strings.Contains(errOut, "no projects yet") {
		t.Fatalf("called %v exit %d %s", called, code, errOut)
	}
}

func TestOpenDeleteFlows(t *testing.T) {
	root := t.TempDir()
	newer := makeRun(t, root, "demo", "20260102-000000")
	older := makeRun(t, root, "demo", "20260101-000000")
	other := makeRun(t, root, "beta", "20260103-000000")
	yes := true

	var listed string
	a := &app{
		root: root,
		tty:  &yes,
		fzf: func(input string, args []string) (string, []string, error) {
			listed = input
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "--expect=ctrl-x,q") || !strings.Contains(joined, "-m") {
				t.Fatalf("args %v", args)
			}
			return "", nil, errSoftQuit
		},
	}
	_, errOut, code := finishApp(t, a, "open")
	if code != 0 || errOut != "jmark: quit\n" {
		t.Fatalf("esc exit %d %q", code, errOut)
	}
	wantOrder := newer + "\n" + older + "\n" + other + "\n"
	if listed != wantOrder {
		t.Fatalf("list\n got %q\nwant %q", listed, wantOrder)
	}

	a.fzf = func(string, []string) (string, []string, error) {
		return "q", []string{newer}, nil
	}
	_, errOut, code = finishApp(t, a, "open", "demo")
	if code != 0 || errOut != "jmark: quit\n" {
		t.Fatalf("q exit %d %q", code, errOut)
	}

	a.fzf = func(input string, _ []string) (string, []string, error) {
		if strings.Contains(input, "beta/") {
			t.Fatalf("filter leaked %q", input)
		}
		return openDeleteKey, []string{older}, nil
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("n\n")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	a.in = r
	_, errOut, code = finishApp(t, a, "open", "demo")
	if code != 1 || !strings.Contains(errOut, "delete cancelled") {
		t.Fatalf("cancel exit %d %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", older)); err != nil {
		t.Fatal(err)
	}

	// Multi-select delete from open must remove every picked run.
	r, w, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("y\n")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	a.in = r
	a.fzf = func(string, []string) (string, []string, error) {
		return openDeleteKey, []string{newer, older}, nil
	}
	_, errOut, code = finishApp(t, a, "open", "demo")
	if code != 0 {
		t.Fatalf("multi-delete exit %d %s", code, errOut)
	}
	if !strings.Contains(errOut, "jmark: delete 2 run(s):") {
		t.Fatalf("want multi confirm, got %s", errOut)
	}
	for _, rel := range []string{newer, older} {
		if _, err := os.Stat(filepath.Join(root, "artifacts", rel)); !os.IsNotExist(err) {
			t.Fatalf("still present %s: %v", rel, err)
		}
		if !strings.Contains(errOut, "jmark: deleted "+rel) {
			t.Fatalf("missing deleted line for %s in %s", rel, errOut)
		}
	}
	// Recreate runs for the rest of the test.
	newer = makeRun(t, root, "demo", "20260102-000000")
	older = makeRun(t, root, "demo", "20260101-000000")

	r, w, err = os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("y\n")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	a.in = r
	a.fzf = func(_ string, args []string) (string, []string, error) {
		if !strings.Contains(strings.Join(args, " "), "--expect=q") {
			t.Fatalf("delete args %v", args)
		}
		return "", []string{older}, nil
	}
	_, errOut, code = finishApp(t, a, "delete", "demo")
	if code != 0 || !strings.Contains(errOut, "jmark: deleted "+older) {
		t.Fatalf("delete exit %d %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", older)); !os.IsNotExist(err) {
		t.Fatalf("old run still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", "private", "demo")); err != nil {
		t.Fatal("sibling project dir removed", err)
	}

	var opened string
	a.browse = func(path string) error {
		opened = path
		return nil
	}
	a.fzf = func(string, []string) (string, []string, error) {
		return "", []string{newer, other}, nil
	}
	_, errOut, code = finishApp(t, a, "open")
	if code != 0 || !strings.Contains(errOut, "jmark: opened compare of 2 run(s)") {
		t.Fatalf("compare exit %d %s", code, errOut)
	}
	html, err := os.ReadFile(opened)
	if err != nil {
		t.Fatal(err)
	}
	os.Remove(opened)
	text := string(html)
	for _, needle := range []string{`"run":"private/demo/20260102-000000"`, `"run":"private/beta/20260103-000000"`, `id="jmark-data"`, `id="app"`} {
		if !strings.Contains(text, needle) {
			t.Errorf("compare missing %s", needle)
		}
	}

	a.fzf = func(string, []string) (string, []string, error) {
		return "", []string{"../etc"}, nil
	}
	_, errOut, code = finishApp(t, a, "open")
	if code != 1 || !strings.Contains(errOut, "invalid pick: ../etc") {
		t.Fatalf("pick exit %d %s", code, errOut)
	}

	a.browse = func(string) error {
		return errors.New("no browser opener (open/xdg-open); file is /tmp/x")
	}
	a.fzf = func(string, []string) (string, []string, error) {
		return "", []string{newer}, nil
	}
	_, errOut, code = finishApp(t, a, "open")
	if code != 1 || !strings.Contains(errOut, "no browser opener") || !strings.Contains(errOut, "could not open compare view") {
		t.Fatalf("opener exit %d %s", code, errOut)
	}
}

func TestDeletePrunesEmptyProject(t *testing.T) {
	root := t.TempDir()
	only := makeRun(t, root, "solo", "20260101-010101")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("y\n")); err != nil {
		t.Fatal(err)
	}
	w.Close()
	a := &app{
		root: root,
		in:   r,
		fzf: func(string, []string) (string, []string, error) {
			return "", []string{only}, nil
		},
	}
	_, errOut, code := finishApp(t, a, "delete")
	if code != 0 || !strings.Contains(errOut, "jmark: deleted "+only) {
		t.Fatalf("exit %d %s", code, errOut)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", "private", "solo")); !os.IsNotExist(err) {
		t.Fatalf("project dir remains: %v", err)
	}
}

func finishApp(t *testing.T, a *app, args ...string) (string, string, int) {
	t.Helper()
	var out, errOut strings.Builder
	a.stdout = &out
	a.stderr = &errOut
	if a.cat == nil {
		a.cat = catalog.New(a.root)
	}
	code := finish(a, args)
	return out.String(), errOut.String(), code
}

func makeRun(t *testing.T, root, id, stamp string) string {
	t.Helper()
	rel := "private/" + id + "/" + stamp
	dir := filepath.Join(root, "artifacts", "private", id, stamp)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "result.ndjson"), []byte(sampleResult), 0o644); err != nil {
		t.Fatal(err)
	}
	return rel
}

func TestBenchPickOptions(t *testing.T) {
	root := pickerRepo(t)
	for _, tc := range []struct {
		name        string
		opts        benchOpts
		picks       []string
		wantRuntime string
		wantFormat  string
	}{
		{"defaults", benchOpts{format: "stream"}, []string{"all", "stream"}, "", "stream"},
		{"runtime-html", benchOpts{format: "stream"}, []string{"available", "html"}, "available", "html"},
		{"runtime-ndjson", benchOpts{format: "stream"}, []string{"available", "ndjson"}, "available", "ndjson"},
		{"explicit-runtime", benchOpts{runtime: "available", runtimeSet: true, format: "stream"}, []string{"html"}, "available", "html"},
		{"explicit-format", benchOpts{format: "stream", formatSet: true}, []string{"available"}, "available", "stream"},
		{"both-explicit", benchOpts{runtime: "available", runtimeSet: true, format: "ndjson", formatSet: true}, nil, "available", "ndjson"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var errOut strings.Builder
			a := &app{root: root, cat: catalog.New(root), stderr: &errOut}
			calls := 0
			a.fzf = func(input string, args []string) (string, []string, error) {
				if calls >= len(tc.picks) {
					t.Fatal("unexpected picker")
				}
				pick := tc.picks[calls]
				calls++
				for _, line := range strings.Split(strings.TrimSuffix(input, "\n"), "\n") {
					if strings.SplitN(line, "\t", 2)[0] == pick {
						return "", []string{line}, nil
					}
				}
				t.Fatalf("missing choice %q in %q", pick, input)
				return "", nil, nil
			}
			opts := tc.opts
			opts.id = "demo"
			got, err := a.benchPickOptions(opts)
			if err != nil || got.runtime != tc.wantRuntime || got.runtimeSet != (tc.wantRuntime != "") || got.format != tc.wantFormat || calls != len(tc.picks) {
				t.Fatalf("got %+v, calls %d, error %v", got, calls, err)
			}
		})
	}
}

func TestBenchOptionsCancelBeforeMeasurement(t *testing.T) {
	for _, cancelAt := range []int{1, 2} {
		yes := true
		calls := 0
		a := &app{root: pickerRepo(t), tty: &yes}
		a.fzf = func(input string, args []string) (string, []string, error) {
			defer func() { calls++ }()
			if calls == cancelAt {
				return "q", nil, nil
			}
			return "", []string{strings.SplitN(input, "\n", 2)[0]}, nil
		}
		out, errOut, code := finishApp(t, a, "bench")
		if code != 0 || out != "" || !strings.HasSuffix(errOut, "jmark: quit\n") || calls != cancelAt+1 {
			t.Fatalf("cancel at %d: calls %d exit %d stdout %q stderr %q", cancelAt, calls, code, out, errOut)
		}
	}
}

func TestListRunsIgnoresStaging(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root, map[string]string{
		"artifacts/private/demo/complete/result.ndjson":            sampleResult,
		"artifacts/private/demo/.pending-incomplete/result.ndjson": sampleResult,
	})
	for _, filter := range []string{"", "demo"} {
		runs, err := listRuns(filepath.Join(root, "artifacts"), filter)
		if err != nil || strings.Join(runs, ",") != "private/demo/complete" {
			t.Fatalf("runs: %v %v", runs, err)
		}
	}
}

func pickerRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeRepo(t, root, map[string]string{
		"runtimes/available.toml":           "family = \"core\"\nneeds = \"sh\"\nrun = \"sh {runner} {iters}\"\n",
		"families/core/runner.src":          "runner",
		"families/core/template/_input.src": "input",
		"families/core/template/loop.src":   "approach",
		"benches/demo/AGENTS.md":            "Compare approaches with identical inputs and counts.",
		"benches/demo/bench.toml":           "n = 1\nwarmup = 0\niters = 1\nruns = 1\n",
		"benches/demo/core/_input.src":      "input",
		"benches/demo/core/loop.src":        "approach",
	})
	return root
}
