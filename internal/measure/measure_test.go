//go:build live

package measure

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

func TestAtReplacesItersOnly(t *testing.T) {
	got := At("run {iters} {probe} {iters}", 4)
	if got != "run 4 {probe} 4" {
		t.Fatalf("got %q", got)
	}
}

func TestExecSplitsSpacesAndClosesStdin(t *testing.T) {
	script := writeFile(t, "s.py", "import sys\nsys.stdout.write('ok %d\\n' % len(sys.stdin.read()))\n")
	out, err := Exec("python3 " + script + " ; echo pwned")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "ok 0\n" {
		t.Fatalf("got %q", out)
	}
}

func TestSinkNumberAndRejects(t *testing.T) {
	m := &Measure{}
	script := writeFile(t, "s.py", `
import sys
mode = sys.argv[1]
iters = int(sys.argv[2])
if mode == "echo":
    print(iters)
elif mode == "lines":
    print(1)
    print(2)
elif mode == "text":
    print("nope")
elif mode == "die":
    sys.exit(3)
elif mode == "half":
    print(iters * 0.5)
elif mode == "big":
    print(9007199254740993 if iters == 1 else 18014398509481984)
elif mode == "neg":
    print(-3 * iters)
`)
	cmd := "python3 " + script + " echo {iters}"
	got, err := m.Sink(cmd, 7)
	if err != nil {
		t.Fatal(err)
	}
	if got != "7" {
		t.Fatalf("sink %q", got)
	}
	if _, err := m.Sink("python3 "+script+" lines {iters}", 1); err == nil || !strings.Contains(err.Error(), "must be one number") {
		t.Fatalf("lines: %v", err)
	}
	if _, err := m.Sink("python3 "+script+" text {iters}", 1); err == nil || !strings.Contains(err.Error(), "must be one number") {
		t.Fatalf("text: %v", err)
	}
	if _, err := m.Sink("python3 "+script+" die {iters}", 1); err == nil || !strings.Contains(err.Error(), "runner failed at iters=1") {
		t.Fatalf("die: %v", err)
	}
}

func TestProportionalPaths(t *testing.T) {
	m := &Measure{}
	script := writeFile(t, "s.py", `
import sys
mode, iters = sys.argv[1], int(sys.argv[2])
if mode == "echo":
    print(iters * 10)
elif mode == "half":
    print(iters * 0.5)
elif mode == "drift":
    print(iters * 0.5 + 0.25)
elif mode == "big":
    print(9007199254740993 if iters == 1 else 18014398509481984)
elif mode == "neg":
    print(-3 * iters)
`)
	if err := m.Proportional("python3 "+script+" echo {iters}", 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.Proportional("python3 "+script+" half {iters}", 1, 2); err != nil {
		t.Fatal(err)
	}
	if err := m.Proportional("python3 "+script+" neg {iters}", 1, 2); err != nil {
		t.Fatal(err)
	}
	err := m.Proportional("python3 "+script+" big {iters}", 1, 2)
	if err == nil || !strings.Contains(err.Error(), "not proportional") {
		t.Fatalf("big: %v", err)
	}
	err = m.Proportional("python3 "+script+" drift {iters}", 1, 2)
	if err == nil || !strings.Contains(err.Error(), "sink(1)=0.75") {
		t.Fatalf("drift: %v", err)
	}
}

func TestBuildClearsBinFamily(t *testing.T) {
	m := &Measure{}
	dir := t.TempDir()
	bin := filepath.Join(dir, "app")
	if err := os.WriteFile(bin, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin+".o", []byte("obj"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app2"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := m.Build(benchmark.Approach{Bin: bin}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin + ".o"); err != nil {
		t.Fatal("nil build_cmd should leave files")
	}
	empty := ""
	if err := m.Build(benchmark.Approach{Bin: bin, BuildCmd: &empty}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(bin + ".o"); err != nil {
		t.Fatal("empty build_cmd should leave files")
	}
	cmd := "printf new > " + strconv.Quote(bin)
	if err := m.Build(benchmark.Approach{Bin: bin, BuildCmd: &cmd}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "new" {
		t.Fatalf("bin %q", body)
	}
	if _, err := os.Stat(bin + ".o"); !os.IsNotExist(err) {
		t.Fatalf("bin.o still present: %v", err)
	}
	kept, err := os.ReadFile(filepath.Join(dir, "app2"))
	if err != nil || string(kept) != "keep" {
		t.Fatalf("sibling: %q %v", kept, err)
	}
	bad := "exit 1"
	err = m.Build(benchmark.Approach{Bin: bin, BuildCmd: &bad})
	if err == nil || !strings.Contains(err.Error(), "build failed: exit 1") {
		t.Fatalf("build err %v", err)
	}
}

func TestPlanFastHyperfine(t *testing.T) {
	root := t.TempDir()
	if strings.Contains(root, " ") {
		t.Fatalf("temp dir %q contains a space", root)
	}
	logPath := filepath.Join(root, "iters.log")
	script := writeFile(t, "s.py", `
import sys
iters = int(sys.argv[1])
with open(sys.argv[2], "a") as fh:
    fh.write(str(iters) + "\n")
print(5 if iters == 0 else iters)
`)
	binDir := filepath.Join(root, ".jmark", "build", "p", "py")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "stale"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	bin := "{build_dir}/b"
	build := "printf built > " + strconv.Quote(bin+".ok")
	timed := "python3 " + script + " {iters} " + logPath
	plan := map[string]any{
		"project": "p",
		"runtime": map[string]any{"id": "py"},
		"params":  map[string]any{"n": 1, "warmup": 0, "iters": 2, "runs": 1, "probes": []any{}},
		"probes":  []any{},
		"approaches": []any{
			map[string]any{"name": "a", "bin": "{build_dir}/a", "timed_cmd": timed},
			map[string]any{"name": "b", "bin": bin, "build_cmd": build, "timed_cmd": timed},
		},
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	m := &Measure{Root: root, Stderr: &stderr}
	observations, err := m.Run(testPlan(t, raw))
	if err != nil {
		t.Fatalf("plan: %v\nstderr: %s", err, stderr.String())
	}
	lines := sampleLines(observations.Samples)
	if len(lines) != 2 {
		t.Fatalf("lines %#v", lines)
	}
	var sinks []string
	for i, name := range []string{"a", "b"} {
		f := strings.Split(lines[i], "\t")
		if len(f) != 4 || f[0] != name || f[1] != "1" {
			t.Fatalf("line %q", lines[i])
		}
		for _, col := range f[2:] {
			v, err := strconv.ParseFloat(col, 64)
			if err != nil || v <= 0 {
				t.Fatalf("time %q", col)
			}
		}
		sinks = append(sinks, f[1])
	}
	if sinks[0] != sinks[1] {
		t.Fatalf("sinks %v", sinks)
	}
	progress := stderr.String()
	if !strings.Contains(progress, "→ p py a\n") || !strings.Contains(progress, "→ p py b\n") {
		t.Fatalf("progress %q", progress)
	}
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLine(string(logged), "0") {
		t.Fatalf("warmup 0 was not passed to hyperfine: %s", logged)
	}
	if _, err := os.Stat(filepath.Join(binDir, "stale")); err != nil {
		t.Fatal("another execution's build files were removed")
	}
	entries, err := os.ReadDir(filepath.Join(root, ".jmark", "build"))
	if err != nil || len(entries) != 1 || entries[0].Name() != "p" {
		t.Fatalf("workspace leak: %v %v", entries, err)
	}

}

func TestPlanProportionalFailure(t *testing.T) {
	script := writeFile(t, "s.py", "import sys\nprint(1)\n")
	timed := "python3 " + script + " {iters}"
	plan := map[string]any{
		"project": "p",
		"runtime": map[string]any{"id": "py"},
		"params":  map[string]any{"warmup": 1, "iters": 1, "runs": 1},
		"approaches": []any{
			map[string]any{"name": "a", "timed_cmd": timed},
		},
	}
	raw, _ := json.Marshal(plan)
	var stderr bytes.Buffer
	m := &Measure{Root: t.TempDir(), Stderr: &stderr}
	_, err := m.timings(testPlan(t, raw))
	if err == nil {
		t.Fatal("expected proportional failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "not proportional") || !strings.Contains(msg, "py a: failed sink proportionality") {
		t.Fatalf("err %v", err)
	}
}

func TestHyperfineFailure(t *testing.T) {
	var stderr bytes.Buffer
	m := &Measure{Stderr: &stderr}
	_, _, err := m.Time("python3 /no/such/jmark-missing.py {iters}", 1, 2, 1)
	if err == nil || !strings.Contains(err.Error(), "hyperfine failed:") {
		t.Fatalf("err %v", err)
	}
	if stderr.Len() == 0 {
		t.Fatal("hyperfine stderr was dropped")
	}
}

func TestProbesMedianOrderAndSelftest(t *testing.T) {
	script := writeFile(t, "s.py", `
import sys
iters = int(sys.argv[1])
out = sys.argv[2]
state = sys.argv[3]
try:
    n = int(open(state).read() or "0")
except FileNotFoundError:
    n = 0
open(state, "w").write(str(n + 1))
deltas = [1, 5, 10, 14]
d = deltas[n]
with open(out, "w") as fh:
    fh.write("z\t%d\n" % d)
    fh.write("a\t%d\n" % (d * 2))
    fh.write("skip this\n")
`)
	state := filepath.Join(t.TempDir(), "n")
	cmd := "python3 " + script + " {iters} {probe_out} " + state
	plan := map[string]any{
		"params": map[string]any{"warmup": 1, "iters": 1, "runs": 2},
		"probes": []any{map[string]any{"name": "alloc", "hook": "/h"}},
		"approaches": []any{
			map[string]any{"name": "nm", "probe_cmd": cmd},
		},
	}
	raw, _ := json.Marshal(plan)
	var stderr bytes.Buffer
	m := &Measure{Stderr: &stderr}
	got, err := m.probes(testPlan(t, raw))
	if err != nil {
		t.Fatal(err)
	}
	want := "nm\ta\t6\t24\nnm\tz\t3\t12\n"
	if probeLines(got) != want {
		t.Fatalf("got %v want %q", got, want)
	}
	if !strings.Contains(stderr.String(), "→ probe nm\n") {
		t.Fatalf("progress %q", stderr.String())
	}

	emptyPlan, _ := json.Marshal(map[string]any{
		"params": map[string]any{"warmup": 1, "iters": 1, "runs": 1},
		"probes": []any{},
	})
	text, err := m.probes(testPlan(t, emptyPlan))
	if err != nil || len(text) != 0 {
		t.Fatalf("empty probes: %v %v", text, err)
	}
	if err := m.ProbeSelftest(testPlan(t, emptyPlan)); err != nil {
		t.Fatal(err)
	}
}

func TestProbeSelftestBias(t *testing.T) {
	m := &Measure{Root: t.TempDir()}
	if err := m.ProbeSelftest(testPlan(t, probePlan(t, 0, 0, 1000))); err != nil {
		t.Fatal(err)
	}
	// 1/10000 = 0.0001, under the 0.001 allowance.
	if err := m.ProbeSelftest(testPlan(t, probePlan(t, 0, 1, 10000))); err != nil {
		t.Fatal(err)
	}
	bad := probePlan(t, 0, 50, 100)
	err := m.ProbeSelftest(testPlan(t, bad))
	if err == nil || err.Error() != "biased probe on _null: alloc=0.5/op" {
		t.Fatalf("err %v", err)
	}
	failCmd := "python3 /no/such/jmark-probe.py {iters} {probe_out}"
	raw, _ := json.Marshal(map[string]any{
		"params": map[string]any{"warmup": 1, "iters": 1, "runs": 1},
		"probes": []any{map[string]any{"name": "alloc"}},
		"approaches": []any{
			map[string]any{"name": "nm", "probe_cmd": failCmd},
		},
	})
	_, err = m.probes(testPlan(t, raw))
	if err == nil || !strings.Contains(err.Error(), "probe run failed: nm iters=1") {
		t.Fatalf("probe fail: %v", err)
	}
}

func TestHostSnapshot(t *testing.T) {
	raw, err := HostSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	var snap struct {
		OS        string `json:"os"`
		Arch      string `json:"arch"`
		Kernel    string `json:"kernel"`
		Hostname  string `json:"hostname"`
		CPU       string `json:"cpu"`
		Hyperfine string `json:"hyperfine"`
		NCPU      *int   `json:"ncpu"`
	}
	if err := json.Unmarshal(raw, &snap); err != nil {
		t.Fatal(err)
	}
	osName := mustOut(t, "uname", "-s")
	if snap.OS != osName || snap.Arch == "" || snap.Kernel == "" || snap.Hostname != "" {
		t.Fatalf("snap %+v raw %s", snap, raw)
	}
	if !strings.Contains(snap.Hyperfine, "hyperfine") {
		t.Fatalf("hyperfine %q", snap.Hyperfine)
	}
	if osName == "Darwin" {
		if snap.CPU == "" || snap.NCPU == nil || *snap.NCPU < 1 {
			t.Fatalf("darwin cpu %+v", snap)
		}
	}
}

func probePlan(t *testing.T, loDelta, hiDelta, iters int) []byte {
	t.Helper()
	script := writeFile(t, "p.py", `
import sys
iters = int(sys.argv[1])
out = sys.argv[2]
lo = int(sys.argv[3])
hi = int(sys.argv[4])
delta = lo if iters == int(sys.argv[5]) else hi
with open(out, "w") as fh:
    fh.write("alloc\t%d\n" % delta)
`)
	// argv: iters, probe_out, loDelta, hiDelta, warmup
	cmd := "python3 " + script + " {iters} {probe_out} " + strconv.Itoa(loDelta) + " " + strconv.Itoa(hiDelta) + " 0"
	raw, err := json.Marshal(map[string]any{
		"params": map[string]any{"warmup": 0, "iters": iters, "runs": 1},
		"probes": []any{map[string]any{"name": "alloc", "hook": "/h"}},
		"approaches": []any{
			map[string]any{"name": "_null", "probe_cmd": cmd},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func writeFile(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	if strings.Contains(dir, " ") {
		t.Fatalf("temp dir %q contains a space", dir)
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func nonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func hasLine(s, want string) bool {
	for _, line := range strings.Split(s, "\n") {
		if line == want {
			return true
		}
	}
	return false
}

func mustOut(t *testing.T, name string, args ...string) string {
	t.Helper()
	b, err := exec.Command(name, args...).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimRight(string(b), "\n")
}

func mustCombined(t *testing.T, name string, args ...string) string {
	t.Helper()
	b, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func testPlan(t *testing.T, raw []byte) benchmark.Plan {
	t.Helper()
	var plan benchmark.Plan
	if err := json.Unmarshal(raw, &plan); err != nil {
		t.Fatal(err)
	}
	return plan
}
func sampleLines(samples []benchmark.Sample) []string {
	var lines []string
	for _, s := range samples {
		lines = append(lines, s.Name+"\t"+string(s.Sink)+"\t"+formatFloat(s.Lo)+"\t"+formatFloat(s.Hi))
	}
	return lines
}
func probeLines(samples []benchmark.ProbeSample) string {
	var b strings.Builder
	for _, s := range samples {
		b.WriteString(s.Name + "\t" + s.Counter + "\t" + formatFloat(s.Lo) + "\t" + formatFloat(s.Hi) + "\n")
	}
	return b.String()
}
