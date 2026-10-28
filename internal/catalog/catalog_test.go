package catalog_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/catalog"
)

func TestValidID(t *testing.T) {
	for _, id := range []string{"a", "a-b", "0", "ab", "approach-a"} {
		if !catalog.ValidID(id) {
			t.Errorf("rejected %q", id)
		}
	}
	for _, id := range []string{"A", "", "a_b", "-a", "a.b"} {
		if catalog.ValidID(id) {
			t.Errorf("accepted %q", id)
		}
	}
}

func TestRealRepoCatalog(t *testing.T) {
	root := repoRoot(t)
	c := catalog.New(root)

	projects, err := c.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(projects, ",") != "sum-large-doubles" {
		t.Fatalf("projects = %v", projects)
	}
	runtimes, err := c.ListRuntimes()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(runtimes, ",") != "bun,node,py" {
		t.Fatalf("runtimes = %v", runtimes)
	}
	fams, err := c.ListFamilies()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(fams, ",") != "js,py" {
		t.Fatalf("families = %v", fams)
	}
	if err := c.CheckRegistry(); err != nil {
		t.Fatal(err)
	}
	if err := c.CheckProject("sum-large-doubles"); err != nil {
		t.Fatal(err)
	}
	for _, runtime := range []string{"bun", "node", "py"} {
		doc, err := c.Plan("sum-large-doubles", runtime)
		if err != nil {
			t.Fatal(err)
		}
		if doc.Project != "sum-large-doubles" || doc.Runtime.ID != runtime {
			t.Fatalf("plan identity %+v", doc)
		}
		if doc.Params.N != 100000 || doc.Params.Warmup != 20 || doc.Params.Iters != 500 || doc.Params.Runs != 10 {
			t.Fatalf("params %+v", doc.Params)
		}
		if len(doc.Approaches) != 2 || doc.Approaches[0].Name != "for-loop" || doc.Approaches[1].Name != "map-filter-reduce" {
			t.Fatalf("approaches %+v", doc.Approaches)
		}
		for _, a := range doc.Approaches {
			if !strings.Contains(a.TimedCmd, "{iters}") || strings.Contains(a.TimedCmd, "{probe}") || a.BuildCmd != nil || a.ProbeCmd != nil {
				t.Fatalf("cmds %+v", a)
			}
		}
	}
}

func TestParams(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"plain", "n = 100000\nwarmup = 20\niters = 500\nruns = 10\n"},
		{"warmup zero", "n = 1\nwarmup = 0\niters = 1\nruns = 1\n"},
		{"quoted int", "n = \"4\"\nwarmup = 0\niters = 1\nruns = 1\n"},
		{"comments and probes", "# head\nn = 1 # units\nwarmup = 0\niters = 2\nruns = 3\nprobes = \"gc alloc\"\n"},
		{"zero n", "n = 0\nwarmup = 0\niters = 1\nruns = 1\n"},
		{"missing n", "warmup = 0\niters = 1\nruns = 1\n"},
		{"unknown", "n = 1\nwarmup = 0\niters = 1\nruns = 1\nfoo = 1\nbar = \"x\"\n"},
		{"table", "[bench]\nn = 1\n"},
		{"duplicate", "n = 1\nn = 2\nwarmup = 0\niters = 1\nruns = 1\n"},
		{"reserved probe", "n = 1\nwarmup = 0\niters = 1\nruns = 1\nprobes = \"time\"\n"},
		{"bad probe", "n = 1\nwarmup = 0\niters = 1\nruns = 1\nprobes = \"gc-1\"\n"},
		{"float", "n = 1.5\nwarmup = 0\niters = 1\nruns = 1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "bench.toml")
			if err := os.WriteFile(path, []byte(tc.body), 0o644); err != nil {
				t.Fatal(err)
			}
			params, err := catalog.New(dir).Params(path)
			valid := tc.name == "plain" || tc.name == "warmup zero" || tc.name == "quoted int" || tc.name == "comments and probes"
			if (err == nil) != valid {
				t.Fatalf("valid=%v: %v", valid, err)
			}
			if valid && (params.N < 1 || params.Iters < 1 || params.Runs < 1) {
				t.Fatalf("params: %+v", params)
			}
			if tc.name == "comments and probes" && strings.Join(params.Probes, ",") != "gc,alloc" {
				t.Fatalf("probes: %v", params.Probes)
			}
		})
	}
}

func TestCatalogValidation(t *testing.T) {
	const nodeRun = "family = \"js\"\nneeds = \"node\"\nrun = \"node {runner} {input} {approach} {n} {iters} {probe}\"\n"
	cases := []struct {
		name  string
		files map[string]string
		fn    string
		args  []string
	}{
		{"registry ok", baseFiles(), "catalog_check_registry", nil},
		{"project ok", baseFiles(), "catalog_check_project", []string{"demo"}},
		{"plan ok", baseFiles(), "catalog_plan", []string{"demo", "node"}},
		{
			"agents too long",
			withBase(map[string]string{"benches/demo/AGENTS.md": strings.Repeat("a", 201) + "\n"}),
			"catalog_check_project", []string{"demo"},
		},
		{
			"agents at limit",
			withBase(map[string]string{"benches/demo/AGENTS.md": strings.Repeat("b", 200) + "\n"}),
			"catalog_check_project", []string{"demo"},
		},
		{
			"missing agents",
			withBase(nil, "benches/demo/AGENTS.md"),
			"catalog_check_project", []string{"demo"},
		},
		{
			"invalid id",
			baseFiles(),
			"catalog_check_project", []string{"Bad"},
		},
		{
			"purity",
			withBase(map[string]string{"benches/demo/js/approach-a.mjs": "export const x = JSON.stringify(1)\n"}),
			"catalog_check_project", []string{"demo"},
		},
		{
			"unknown family dir",
			withBase(map[string]string{"benches/demo/nope/keep.txt": "x\n"}),
			"catalog_check_project", []string{"demo"},
		},
		{
			"bad extension",
			withBase(map[string]string{"benches/demo/js/notes.txt": "x\n"}),
			"catalog_check_project", []string{"demo"},
		},
		{
			"approach sets",
			withBase(map[string]string{
				"families/py/runner.py":              "def run():\n    return 0\n",
				"families/py/template/_input.py":     "def make_input(n):\n    return []\n",
				"families/py/template/approach-b.py": "def run(data):\n    return 0\n",
				"benches/demo/py/_input.py":          "def make_input(n):\n    return []\n",
				"benches/demo/py/approach-b.py":      "def run(data):\n    return 0\n",
			}),
			"catalog_check_project", []string{"demo"},
		},
		{
			"bad approach name",
			withBase(map[string]string{"benches/demo/js/Bad.mjs": "export function run(){return 0}\n"}),
			"catalog_check_project", []string{"demo"},
		},
		{
			"probe plan",
			probeRepo(nodeRun),
			"catalog_plan", []string{"demo", "node"},
		},
		{
			"probe selftest",
			probeRepo(nodeRun),
			"catalog_selftest_plan", []string{"node"},
		},
		{
			"probe not offered",
			probeRepo(nodeRun),
			"catalog_check_project", []string{"demo"},
		},
		{
			"probe registry",
			probeRepo(nodeRun),
			"catalog_check_registry", nil,
		},
		{
			"missing null",
			withBase(map[string]string{"families/js/probes/gc.mjs": "export function snapshot(){return {bytes:0}}\n"}),
			"catalog_check_registry", nil,
		},
		{
			"runner mentions probes",
			withBase(map[string]string{"families/js/runner.mjs": "load('probes/gc.mjs')\n"}),
			"catalog_check_registry", nil,
		},
		{
			"invalid runtime id",
			withBase(map[string]string{"runtimes/Bad.toml": "family = \"nope\"\nneeds = \"x\"\nrun = \"x {iters}\"\n"}),
			"catalog_check_registry", nil,
		},
		{
			"unknown runtime key",
			withBase(map[string]string{"runtimes/node.toml": nodeRun + "extra = \"1\"\n"}),
			"catalog_check_registry", nil,
		},
		{
			"unknown placeholder",
			withBase(map[string]string{"runtimes/node.toml": nodeRun + "build = \"cc {bogus}\"\n"}),
			"catalog_check_registry", nil,
		},
		{
			"missing run",
			withBase(map[string]string{"runtimes/node.toml": "family = \"js\"\nneeds = \"node\"\n"}),
			"catalog_check_registry", nil,
		},
		{
			"run without iters",
			withBase(map[string]string{"runtimes/node.toml": "family = \"js\"\nneeds = \"node\"\nrun = \"node {runner}\"\n"}),
			"catalog_check_registry", nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFiles(t, root, tc.files)
			got, err := goCall(catalog.New(root), tc.fn, tc.args...)
			valid := map[string]bool{"registry ok": true, "project ok": true, "plan ok": true, "agents at limit": true, "probe plan": true, "probe selftest": true, "probe registry": true}[tc.name]
			if (err == nil) != valid {
				t.Fatalf("valid=%v: %v", valid, err)
			}
			if valid && (tc.fn == "catalog_plan" || tc.fn == "catalog_selftest_plan") {
				var doc map[string]any
				if err := json.Unmarshal([]byte(got), &doc); err != nil {
					t.Fatal(err)
				}
				if doc["runtime"] == nil || doc["approaches"] == nil {
					t.Fatalf("plan: %s", got)
				}
			}
		})
	}
}

func TestScaffoldNoClobber(t *testing.T) {
	repo := repoRoot(t)
	goRoot := t.TempDir()
	for _, sub := range []string{"families", "runtimes"} {
		if err := os.Symlink(filepath.Join(repo, sub), filepath.Join(goRoot, sub)); err != nil {
			t.Fatal(err)
		}
	}
	if err := catalog.New(goRoot).Scaffold("demo", "js", "py"); err != nil {
		t.Fatal(err)
	}
	agents := filepath.Join(goRoot, "benches", "demo", "AGENTS.md")
	approach := filepath.Join(goRoot, "benches", "demo", "js", "approach-a.mjs")
	input := filepath.Join(goRoot, "benches", "demo", "py", "_input.py")
	if err := os.WriteFile(agents, []byte("custom agents\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(approach, []byte("custom approach\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(input); err != nil {
		t.Fatal(err)
	}
	if err := catalog.New(goRoot).Scaffold("demo", "js", "py"); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(agents); string(b) != "custom agents\n" {
		t.Fatalf("agents overwritten: %q", b)
	}
	if b, _ := os.ReadFile(approach); string(b) != "custom approach\n" {
		t.Fatalf("approach overwritten: %q", b)
	}
	if _, err := os.Stat(input); err != nil {
		t.Fatal(err)
	}
}

func baseFiles() map[string]string {
	return map[string]string{
		"families/js/runner.mjs":              "export const runner = 1\n",
		"families/js/template/_input.mjs":     "export function makeInput(){return []}\n",
		"families/js/template/approach-a.mjs": "export function run(){return 0}\n",
		"families/js/forbid.re":               "JSON\\.\n",
		"runtimes/node.toml":                  "family = \"js\"\nneeds = \"node\"\nrun = \"node {runner} {input} {approach} {n} {iters} {probe}\"\n",
		"benches/demo/AGENTS.md":              "short\n",
		"benches/demo/bench.toml":             "n = 10\nwarmup = 0\niters = 2\nruns = 3\n",
		"benches/demo/js/_input.mjs":          "export function makeInput(){return []}\n",
		"benches/demo/js/approach-a.mjs":      "export function run(){return 0}\n",
	}
}

func probeRepo(nodeRun string) map[string]string {
	return withBase(map[string]string{
		"runtimes/node.toml":           nodeRun + "build = \"echo {bin}\"\nprobes = \"gc alloc\"\n",
		"families/js/probes/gc.mjs":    "export function snapshot(){return {bytes:0}}\n",
		"families/js/probes/alloc.mjs": "export function snapshot(){return {bytes:0}}\n",
		"families/js/probes/_null.mjs": "export function snapshot(){return {bytes:0}}\n",
		"benches/demo/bench.toml":      "n = 10\nwarmup = 0\niters = 2\nruns = 3\nprobes = \"gc nope\"\n",
	})
}

func withBase(extra map[string]string, drop ...string) map[string]string {
	out := maps.Clone(baseFiles())
	for k, v := range extra {
		out[k] = v
	}
	for _, k := range drop {
		delete(out, k)
	}
	return out
}

func writeFiles(t *testing.T, root string, files map[string]string) {
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

func goCall(c *catalog.Catalog, fn string, args ...string) (string, error) {
	switch fn {
	case "catalog_projects":
		ids, err := c.ListProjects()
		return strings.Join(ids, "\n"), err
	case "catalog_runtimes":
		ids, err := c.ListRuntimes()
		return strings.Join(ids, "\n"), err
	case "catalog_families":
		ids, err := c.ListFamilies()
		return strings.Join(ids, "\n"), err
	case "catalog_project_runtimes":
		ids, err := c.ProjectRuntimes(args[0])
		return strings.Join(ids, "\n"), err
	case "catalog_params":
		p, err := c.Params(args[0])
		if err != nil {
			return "", err
		}
		b, err := json.Marshal(p)
		return string(b), err
	case "catalog_plan":
		b, err := c.Plan(args[0], args[1])
		raw, marshalErr := json.Marshal(b)
		if err != nil {
			return "", err
		}
		return string(raw), marshalErr
	case "catalog_selftest_plan":
		b, err := c.SelftestPlan(args[0])
		raw, marshalErr := json.Marshal(b)
		if err != nil {
			return "", err
		}
		return string(raw), marshalErr
	case "catalog_check_registry":
		return "", c.CheckRegistry()
	case "catalog_check_project":
		return "", c.CheckProject(args[0])
	case "catalog_runtime_missing":
		miss, err := c.RuntimeMissing(args[0])
		return strings.Join(miss, " "), err
	case "catalog_project_exists":
		if c.ProjectExists(args[0]) {
			return "", nil
		}
		return "", errors.New("absent")
	case "catalog_runtime_exists":
		if c.RuntimeExists(args[0]) {
			return "", nil
		}
		return "", errors.New("absent")
	default:
		return "", fmt.Errorf("unknown fn %s", fn)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestPlanDefersExecutionPathsAndCapturesVersionCommand(t *testing.T) {
	root := t.TempDir()
	writeFiles(t, root, baseFiles())
	c := catalog.New(root)
	plan, err := c.Plan("demo", "node")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Runtime.VersionCmd != "node --version" {
		t.Fatalf("version command: %s", plan.Runtime.VersionCmd)
	}
	for _, a := range plan.Approaches {
		if a.Bin != "{build_dir}/"+a.Name {
			t.Fatalf("execution path allocated in catalog: %s", a.Bin)
		}
	}
	path := filepath.Join(root, "runtimes", "node.toml")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, []byte("version = \"engine-version --short\"\n")...), 0644); err != nil {
		t.Fatal(err)
	}
	if err := c.CheckRegistry(); err != nil {
		t.Fatal(err)
	}
	plan, err = c.Plan("demo", "node")
	if err != nil || plan.Runtime.VersionCmd != "engine-version --short" {
		t.Fatalf("custom version: %s %v", plan.Runtime.VersionCmd, err)
	}
}
