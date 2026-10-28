package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

// CheckRegistry reports layout errors for families/ and runtimes/.
// A nil error means the registry matches the catalog rules.
func (c *Catalog) CheckRegistry() error {
	var errs []error
	fams, err := c.ListFamilies()
	if err != nil {
		return err
	}
	famSet := membership(fams)
	for _, fam := range fams {
		if !ValidID(fam) {
			errs = append(errs, jerr("families/%s: invalid family id", fam))
			continue
		}
		ext, err := c.familyExt(fam)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tdir := filepath.Join(c.root, "families", fam, "template")
		if !isFile(filepath.Join(tdir, "_input."+ext)) {
			errs = append(errs, jerr("families/%s: missing template/_input.%s", fam, ext))
		}
		names, err := approaches(tdir, ext)
		if err != nil {
			errs = append(errs, err)
		} else if len(names) == 0 {
			errs = append(errs, jerr("families/%s: template has no approach", fam))
		}
		runner := filepath.Join(c.root, "families", fam, "runner."+ext)
		body, err := os.ReadFile(runner)
		if err != nil {
			errs = append(errs, jerr("%s", err))
		} else if strings.Contains(string(body), "probes/") {
			errs = append(errs, jerr("families/%s/runner.%s: must not reference probes/ (hooks arrive only as probe-pass argv)", fam, ext))
		}
		offered, err := c.familyProbes(fam, ext)
		if err != nil {
			errs = append(errs, err)
		} else if len(offered) > 0 && !isFile(filepath.Join(c.root, "families", fam, "probes", "_null."+ext)) {
			errs = append(errs, jerr("families/%s/probes: missing _null.%s for the bias self-test", fam, ext))
		}
		var purityFiles []string
		if isFile(runner) {
			purityFiles = append(purityFiles, runner)
		}
		tmpl, err := extFiles(tdir, ext)
		if err != nil {
			errs = append(errs, err)
		} else {
			purityFiles = append(purityFiles, tmpl...)
		}
		if err := c.purity(fam, purityFiles); err != nil {
			errs = append(errs, err)
		}
	}

	runtimes, err := c.ListRuntimes()
	if err != nil {
		return err
	}
	allowed := strings.Join(runtimeKeys, " ")
	for _, runtime := range runtimes {
		rel := "runtimes/" + runtime + ".toml"
		if !ValidID(runtime) {
			errs = append(errs, jerr("%s: invalid runtime id", rel))
			continue
		}
		f, err := parseFlat(c.runtimePath(runtime))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if unknown := f.unknown(runtimeKeys); len(unknown) > 0 {
			for _, k := range unknown {
				errs = append(errs, jerr("%s: unknown key %s (allowed: %s)", f.path, k, allowed))
			}
			continue
		}
		fam := f.get("family")
		if fam == "" || !famSet[fam] {
			errs = append(errs, jerr("%s: family '%s' not found under families/", rel, fam))
			continue
		}
		run := f.get("run")
		if run == "" {
			errs = append(errs, jerr("%s: missing run", rel))
		}
		if f.get("needs") == "" {
			errs = append(errs, jerr("%s: missing needs", rel))
		}
		for _, ph := range placeholderNames(run, f.get("build")) {
			if !placeholders[ph] {
				errs = append(errs, jerr("%s: unknown placeholder {%s}", rel, ph))
			}
		}
		if !strings.Contains(run, "{iters}") {
			errs = append(errs, jerr("%s: run must contain {iters}", rel))
		}
		ext, err := c.familyExt(fam)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		offered, err := c.familyProbes(fam, ext)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		have := membership(offered)
		for _, probe := range ifsFields(f.get("probes")) {
			if err := checkProbeName(probe, rel); err != nil {
				errs = append(errs, err)
				continue
			}
			if !have[probe] {
				errs = append(errs, jerr("%s: claims probe '%s' but families/%s/probes has no hook", rel, probe, fam))
			}
		}
	}
	return errors.Join(errs...)
}

// CheckProject reports layout errors for benches/<id>.
// A nil error means the project matches the catalog rules.
func (c *Catalog) CheckProject(id string) error {
	if !ValidID(id) {
		return jerr("invalid project id '%s'", id)
	}
	dir := filepath.Join(c.root, "benches", id)
	agents := filepath.Join(dir, "AGENTS.md")
	if !isFile(agents) {
		return jerr("benches/%s: missing AGENTS.md", id)
	}
	var errs []error
	n, err := agentRunes(agents)
	if err != nil {
		return err
	}
	if n > agentsMax {
		errs = append(errs, jerr("benches/%s/AGENTS.md: %d chars (max 200)", id, n))
	}
	bench := filepath.Join(dir, "bench.toml")
	if !isFile(bench) {
		errs = append(errs, jerr("benches/%s: missing bench.toml", id))
	} else if params, err := c.Params(bench); err != nil {
		errs = append(errs, err)
	} else {
		claimed, err := c.claimedProbes()
		if err != nil {
			errs = append(errs, err)
		} else {
			for _, probe := range params.Probes {
				if !claimed[probe] {
					errs = append(errs, jerr("benches/%s/bench.toml: no registered runtime offers probe '%s'", id, probe))
				}
			}
		}
	}

	entries, err := readDir(dir)
	if err != nil {
		return err
	}
	known, err := c.ListFamilies()
	if err != nil {
		return err
	}
	knownSet := membership(known)
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !isDir(filepath.Join(dir, name)) {
			continue
		}
		if !knownSet[name] {
			errs = append(errs, jerr("benches/%s/%s/: not a known family (%s)", id, name, bashWords(known)))
		}
	}

	fams, err := c.ProjectFamilies(id)
	if err != nil {
		return err
	}
	if len(fams) == 0 {
		errs = append(errs, jerr("benches/%s: no family source dirs", id))
		return errors.Join(errs...)
	}

	var first []string
	var firstFam string
	for _, fam := range fams {
		ext, err := c.familyExt(fam)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		famDir := filepath.Join(dir, fam)
		matches, err := filepath.Glob(filepath.Join(famDir, "*"))
		if err != nil {
			errs = append(errs, err)
			continue
		}
		for _, m := range matches {
			if !isFile(m) {
				continue
			}
			base := filepath.Base(m)
			if !strings.HasSuffix(base, "."+ext) {
				errs = append(errs, jerr("benches/%s/%s/%s: expected .%s", id, fam, base, ext))
			}
		}
		if !isFile(filepath.Join(famDir, "_input."+ext)) {
			errs = append(errs, jerr("benches/%s/%s: missing _input.%s", id, fam, ext))
		}
		names, err := approaches(famDir, ext)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(names) == 0 {
			errs = append(errs, jerr("benches/%s/%s: no approach files", id, fam))
			continue
		}
		for _, name := range names {
			if !ValidID(name) {
				errs = append(errs, jerr("benches/%s/%s/%s.%s: approach name must match %s", id, fam, name, ext, idPattern))
			}
		}
		if firstFam == "" {
			first = names
			firstFam = fam
		} else if !slices.Equal(first, names) {
			errs = append(errs, jerr("benches/%s: approach sets differ: %s=[ %s] %s=[ %s]", id, firstFam, bashWords(first), fam, bashWords(names)))
		}
		srcs, err := extFiles(famDir, ext)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := c.purity(fam, srcs); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// agentRunes counts Unicode scalars after stripping LF, matching `tr -d '\n' | wc -m`.
func agentRunes(path string) (int, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	s := strings.ReplaceAll(string(b), "\n", "")
	return utf8.RuneCountInString(s), nil
}

func (c *Catalog) purity(fam string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	rePath := filepath.Join(c.root, "families", fam, "forbid.re")
	b, err := os.ReadFile(rePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return jerr("%s", err)
	}
	var patterns []*regexp.Regexp
	for _, line := range splitLines(string(b)) {
		if line == "" {
			continue
		}
		re, err := regexp.Compile(line)
		if err != nil {
			return jerr("%s: %v", rePath, err)
		}
		patterns = append(patterns, re)
	}
	if len(patterns) == 0 {
		return nil
	}
	var hits []string
	rootPrefix := c.root + string(os.PathSeparator)
	for _, path := range files {
		body, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for i, line := range splitLines(string(body)) {
			if !anyMatch(patterns, line) {
				continue
			}
			hit := fmt.Sprintf("%s:%d:%s", path, i+1, line)
			if strings.HasPrefix(hit, rootPrefix) {
				hit = "  " + strings.TrimPrefix(hit, rootPrefix)
			}
			hits = append(hits, hit)
		}
	}
	if len(hits) == 0 {
		return nil
	}
	var bld strings.Builder
	bld.WriteString("jmark: clock/JSON use is forbidden in pure-logic and runner files:")
	for _, h := range hits {
		bld.WriteByte('\n')
		bld.WriteString(h)
	}
	return errors.New(bld.String())
}

func anyMatch(patterns []*regexp.Regexp, line string) bool {
	for _, re := range patterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
