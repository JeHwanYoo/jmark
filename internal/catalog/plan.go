package catalog

import (
	"errors"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

// Params reads a bench.toml path and validates workload keys.
func (c *Catalog) Params(path string) (benchmark.Params, error) {
	f, err := parseFlat(path)
	if err != nil {
		return benchmark.Params{}, err
	}
	if unknown := f.unknown(benchKeys); len(unknown) > 0 {
		allowed := strings.Join(benchKeys, " ")
		var errs []error
		for _, k := range unknown {
			errs = append(errs, jerr("%s: unknown key %s (allowed: %s)", path, k, allowed))
		}
		return benchmark.Params{}, errors.Join(errs...)
	}
	nums := map[string]int64{}
	for _, k := range []string{"n", "warmup", "iters", "runs"} {
		v := f.get(k)
		if !reCanon.MatchString(v) {
			return benchmark.Params{}, jerr("%s: %s must be a non-negative integer", path, k)
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return benchmark.Params{}, jerr("%s: %s must be a non-negative integer", path, k)
		}
		if k != "warmup" && n < 1 {
			return benchmark.Params{}, jerr("%s: %s must be ≥ 1", path, k)
		}
		nums[k] = n
	}
	metrics := strings.Fields(f.get("metrics"))
	if _, present := f.val["metrics"]; present && len(metrics) == 0 {
		return benchmark.Params{}, jerr("metrics must not be empty")
	}
	for _, id := range metrics {
		if _, ok := benchmark.Metrics[id]; !ok {
			return benchmark.Params{}, jerr("unknown metric %s", id)
		}
	}
	rawProbes := f.get("probes")
	for _, name := range ifsFields(rawProbes) {
		if err := checkProbeName(name, path); err != nil {
			return benchmark.Params{}, err
		}
	}
	return benchmark.Params{
		N:       int(nums["n"]),
		Metrics: metrics,
		Warmup:  int(nums["warmup"]),
		Iters:   int(nums["iters"]),
		Runs:    int(nums["runs"]),
		Probes:  splitSpace(rawProbes),
	}, nil
}

// Plan builds the execution plan for one project and runtime.
func (c *Catalog) Plan(id, runtime string) (benchmark.Plan, error) {
	L, err := c.Runtime(runtime)
	if err != nil {
		return benchmark.Plan{}, err
	}
	fam := L.Family
	ext, err := c.familyExt(fam)
	if err != nil {
		return benchmark.Plan{}, err
	}
	dir := filepath.Join(c.root, "benches", id, fam)
	params, err := c.Params(filepath.Join(c.root, "benches", id, "bench.toml"))
	if err != nil {
		return benchmark.Plan{}, err
	}
	names, err := approaches(dir, ext)
	if err != nil {
		return benchmark.Plan{}, err
	}
	return c.planFor(id, runtime, dir, filepath.Join(dir, "_input."+ext), params, names)
}

// SelftestPlan builds a benchmark.Plan that runs every claimed probe against probes/_null.
func (c *Catalog) SelftestPlan(runtime string) (benchmark.Plan, error) {
	L, err := c.Runtime(runtime)
	if err != nil {
		return benchmark.Plan{}, err
	}
	ext, err := c.familyExt(L.Family)
	if err != nil {
		return benchmark.Plan{}, err
	}
	params := benchmark.Params{
		N:      1000,
		Warmup: 100,
		Iters:  10000,
		Runs:   3,
		Probes: splitSpace(L.Probes),
	}
	dir := filepath.Join(c.root, "families", L.Family, "probes")
	input := filepath.Join(c.root, "families", L.Family, "template", "_input."+ext)
	return c.planFor("_selftest", runtime, dir, input, params, []string{"_null"})
}

func (c *Catalog) planFor(id, runtime, srcdir, input string, params benchmark.Params, names []string) (benchmark.Plan, error) {
	L, err := c.Runtime(runtime)
	if err != nil {
		return benchmark.Plan{}, err
	}
	ext, err := c.familyExt(L.Family)
	if err != nil {
		return benchmark.Plan{}, err
	}
	claimed := splitSpace(L.Probes)
	claimSet := membership(claimed)
	hookdir := filepath.Join(c.root, "families", L.Family, "probes")
	probes := []benchmark.PlanProbe{}
	unavailable := []string{}
	for _, name := range params.Probes {
		if claimSet[name] {
			probes = append(probes, benchmark.PlanProbe{
				Name: name,
				Hook: filepath.Join(hookdir, name+"."+ext),
			})
		} else {
			unavailable = append(unavailable, name)
		}
	}
	bindir := "{build_dir}"
	runner := filepath.Join(c.root, "families", L.Family, "runner."+ext)
	approachesOut := make([]benchmark.Approach, 0, len(names))
	probeArg := "{probe_out}"
	if len(probes) > 0 {
		hooks := make([]string, len(probes))
		for i, p := range probes {
			hooks[i] = p.Hook
		}
		probeArg = "{probe_out} " + strings.Join(hooks, " ")
	}
	for _, name := range names {
		path := filepath.Join(srcdir, name+"."+ext)
		bin := filepath.Join(bindir, name)
		expanded := expandCmd(L.Run, runner, input, path, bin, params.N)
		a := benchmark.Approach{
			Name:     name,
			Path:     path,
			Bin:      bin,
			TimedCmd: squeeze(strings.ReplaceAll(expanded, "{probe}", "")),
		}
		if L.Build != "" {
			b := expandCmd(L.Build, runner, input, path, bin, params.N)
			a.BuildCmd = &b
		}
		if len(probes) > 0 {
			p := squeeze(strings.ReplaceAll(expanded, "{probe}", probeArg))
			a.ProbeCmd = &p
		}
		approachesOut = append(approachesOut, a)
	}
	version := L.Version
	if version == "" {
		if needs := strings.Fields(L.Needs); len(needs) > 0 {
			version = needs[0] + " --version"
		}
	}
	doc := benchmark.Plan{
		Project: id,
		Runtime: benchmark.PlanRuntime{
			ID:         runtime,
			VersionCmd: version,
			Family:     L.Family,
			Needs:      L.Needs,
			Probes:     claimed,
		},
		Family: benchmark.PlanFamily{
			ID:     L.Family,
			Ext:    ext,
			Runner: runner,
		},
		Input:             input,
		Params:            params,
		Probes:            probes,
		ProbesUnavailable: unavailable,
		Approaches:        approachesOut,
	}
	if doc.Params.Probes == nil {
		doc.Params.Probes = []string{}
	}
	return doc, nil
}

// Iteration and build-directory placeholders are bound by the execution layer.
func expandCmd(tmpl, runner, input, approach, bin string, n int) string {
	s := strings.ReplaceAll(tmpl, "{runner}", runner)
	s = strings.ReplaceAll(s, "{input}", input)
	s = strings.ReplaceAll(s, "{approach}", approach)
	s = strings.ReplaceAll(s, "{bin}", bin)
	s = strings.ReplaceAll(s, "{n}", strconv.Itoa(n))
	return s
}

var spaceRun = regexp.MustCompile(` +`)

func squeeze(s string) string {
	return strings.TrimSuffix(spaceRun.ReplaceAllString(s, " "), " ")
}

func placeholderNames(run, build string) []string {
	var found []string
	seen := map[string]bool{}
	for _, src := range []string{run, build} {
		for _, m := range phPat.FindAllString(src, -1) {
			name := strings.TrimSuffix(strings.TrimPrefix(m, "{"), "}")
			if seen[name] {
				continue
			}
			seen[name] = true
			found = append(found, name)
		}
	}
	sort.Strings(found)
	return found
}

var phPat = regexp.MustCompile(`\{[^}]*\}`)

func checkProbeName(name, where string) error {
	if !probePat.MatchString(name) {
		return jerr("%s: invalid probe name '%s'", where, name)
	}
	if reservedProbes[name] {
		return jerr("%s: probe '%s' is reserved; wall time comes only from the external clock", where, name)
	}
	return nil
}
