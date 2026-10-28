package measure

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

func TestRunWorkspaceLifetime(t *testing.T) {
	root := t.TempDir()
	tools := t.TempDir()
	// The timer fixture reports fixed medians; runner and probe processes still execute.
	timer := `#!/bin/sh
while [ "$#" -gt 0 ]; do
 if [ "$1" = "--export-json" ]; then shift; out=$1; fi
 shift
done
printf '{"results":[{"median":1,"parameters":{"iters":"0"}},{"median":2,"parameters":{"iters":"2"}}]}' > "$out"
`
	if err := os.WriteFile(filepath.Join(tools, "hyperfine"), []byte(timer), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	runner := filepath.Join(root, "runner")
	source := `#!/bin/sh
if [ -n "$2" ]; then printf 'alloc\t%s\n' "$(( $1 * 2 ))" > "$2"; fi
printf '%s\n' "$1"
`
	if err := os.WriteFile(runner, []byte(source), 0755); err != nil {
		t.Fatal(err)
	}
	build := "cp " + runner + " {build_dir}/loop"
	probe := "{build_dir}/loop {iters} {probe_out}"
	plan := benchmark.Plan{
		Project: "demo", Runtime: benchmark.PlanRuntime{ID: "engine"},
		Params:     benchmark.Params{N: 1, Iters: 2, Runs: 1},
		Probes:     []benchmark.PlanProbe{{Name: "alloc"}},
		Approaches: []benchmark.Approach{{Name: "loop", Bin: "{build_dir}/loop", BuildCmd: &build, TimedCmd: "{build_dir}/loop {iters}", ProbeCmd: &probe}},
	}
	m := &Measure{Root: root, Stderr: io.Discard}
	// Hold two prepared plans simultaneously: neither build may overwrite the other.
	first, closeFirst, err := m.prepare(plan)
	if err != nil {
		t.Fatal(err)
	}
	defer closeFirst()
	second, closeSecond, err := m.prepare(plan)
	if err != nil {
		t.Fatal(err)
	}
	defer closeSecond()
	if first.Approaches[0].Bin == second.Approaches[0].Bin {
		t.Fatal("shared build path")
	}
	closeFirst()
	if _, err := os.Stat(second.Approaches[0].Bin); err != nil {
		t.Fatal("cleanup removed another build", err)
	}
	closeSecond()
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			result, err := m.Run(plan)
			if err != nil {
				t.Error(err)
				return
			}
			if len(result.Samples) != 1 || result.Samples[0].Sink != "1" || len(result.Probes) != 1 || result.Probes[0].Hi != 4 {
				t.Errorf("observations: %+v", result)
			}
		})
	}
	wg.Wait()
	if *plan.Approaches[0].BuildCmd != build || plan.Approaches[0].Bin != "{build_dir}/loop" {
		t.Fatal("caller plan mutated")
	}
	assertNoWorkspaces(t, root)
	// Both build errors and later probe errors release the execution workspace.
	for _, failure := range []string{"build", "probe"} {
		t.Run(failure, func(t *testing.T) {
			broken := plan
			broken.Approaches = append([]benchmark.Approach(nil), plan.Approaches...)
			if failure == "build" {
				cmd := "exit 7"
				broken.Approaches[0].BuildCmd = &cmd
			} else {
				cmd := "missing-jmark-command {iters}"
				broken.Approaches[0].ProbeCmd = &cmd
			}
			if _, err := m.Run(broken); err == nil {
				t.Fatal("expected failure")
			}
			assertNoWorkspaces(t, root)
		})
	}
	// Probe self-tests build their executable before using it and release it afterward.
	plan.Params.Iters = 2
	if err := m.ProbeSelftest(plan); err == nil || !strings.Contains(err.Error(), "biased probe") {
		t.Fatalf("self-test: %v", err)
	}
	assertNoWorkspaces(t, root)
}

func assertNoWorkspaces(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, ".jmark", "build"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("workspace leak: %v %v", entries, err)
	}
}

func TestRunRejectsInvalidCounts(t *testing.T) {
	for _, params := range []benchmark.Params{{Iters: 0, Runs: 1}, {Iters: 1, Runs: 0}, {Iters: 1, Runs: 1, Warmup: -1}, {Iters: 1, Runs: 1, Warmup: int(^uint(0) >> 1)}} {
		t.Run(fmt.Sprint(params), func(t *testing.T) {
			m := &Measure{Root: t.TempDir()}
			if _, err := m.Run(benchmark.Plan{Params: params}); err == nil {
				t.Fatal("invalid counts accepted")
			}
		})
	}
}

func TestRuntimeVersionCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "version")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf ' engine 1.2  \\nsecond\\n' >&2\n"), 0755); err != nil {
		t.Fatal(err)
	}
	got, err := RuntimeVersion(path + " --version")
	if err != nil || got != "engine 1.2" {
		t.Fatalf("version: %q %v", got, err)
	}
	if got, err := RuntimeVersion(""); err != nil || got != "" {
		t.Fatalf("empty: %q %v", got, err)
	}
	if _, err := RuntimeVersion("missing-jmark-tool --version"); err == nil {
		t.Fatal("version failure swallowed")
	}
}
