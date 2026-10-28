// Package catalog owns layout, naming, and plan assembly for a jmark repo.
package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	idPattern    = `^[a-z0-9][a-z0-9-]*$`
	probePattern = `^[a-z0-9][a-z0-9_]*$`
	agentsMax    = 200
)

var (
	idPat    = regexp.MustCompile(idPattern)
	probePat = regexp.MustCompile(probePattern)

	runtimeKeys = []string{"family", "needs", "build", "run", "version", "probes"}
	benchKeys   = []string{"n", "warmup", "iters", "runs", "probes", "metrics"}

	placeholders = map[string]bool{
		"runner": true, "input": true, "approach": true, "bin": true,
		"n": true, "iters": true, "probe": true,
	}
	reservedProbes = map[string]bool{
		"time": true, "wall": true, "wall_time": true, "walltime": true,
		"clock": true, "cpu_time": true, "duration": true, "elapsed": true,
		"latency": true, "us_per_op": true, "ns_per_op": true, "ms_per_op": true,
		"s_per_op": true,
	}
)

// Catalog reads and checks one repo tree. root is the absolute repo root.
type Catalog struct {
	root string
}

// New returns a catalog rooted at root. The path is cleaned.
func New(root string) *Catalog {
	return &Catalog{root: filepath.Clean(root)}
}

// ValidID reports whether s matches the project, runtime, family, and approach id rule.
func ValidID(s string) bool {
	return idPat.MatchString(s)
}

// Runtime is one runtimes/<id>.toml record. Its command and list fields are raw TOML strings.
type Runtime struct {
	ID      string
	Family  string
	Needs   string
	Build   string
	Run     string
	Version string
	Probes  string
}

func (c *Catalog) runtimePath(id string) string {
	return filepath.Join(c.root, "runtimes", id+".toml")
}

// ListProjects returns bench ids that have AGENTS.md, sorted.
func (c *Catalog) ListProjects() ([]string, error) {
	entries, err := readDir(filepath.Join(c.root, "benches"))
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		id := e.Name()
		if strings.HasPrefix(id, ".") {
			continue
		}
		dir := filepath.Join(c.root, "benches", id)
		if !isDir(dir) || !isFile(filepath.Join(dir, "AGENTS.md")) {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}

// ListRuntimes returns runtime ids from runtimes/*.toml, sorted.
func (c *Catalog) ListRuntimes() ([]string, error) {
	entries, err := readDir(filepath.Join(c.root, "runtimes"))
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") || !strings.HasSuffix(name, ".toml") {
			continue
		}
		if !isFile(filepath.Join(c.root, "runtimes", name)) {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".toml"))
	}
	return ids, nil
}

// ListFamilies returns family ids from families/*, sorted.
func (c *Catalog) ListFamilies() ([]string, error) {
	entries, err := readDir(filepath.Join(c.root, "families"))
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		id := e.Name()
		if strings.HasPrefix(id, ".") {
			continue
		}
		if isDir(filepath.Join(c.root, "families", id)) {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ProjectExists reports whether benches/<id>/AGENTS.md is a file.
func (c *Catalog) ProjectExists(id string) bool {
	return isFile(filepath.Join(c.root, "benches", id, "AGENTS.md"))
}

// RuntimeExists reports whether id is a valid id and runtimes/<id>.toml is a file.
func (c *Catalog) RuntimeExists(id string) bool {
	return ValidID(id) && isFile(c.runtimePath(id))
}

// Runtime reads runtimes/<id>.toml. It does not check that the family exists.
func (c *Catalog) Runtime(id string) (Runtime, error) {
	f, err := parseFlat(c.runtimePath(id))
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{
		ID:      id,
		Family:  f.get("family"),
		Needs:   f.get("needs"),
		Build:   f.get("build"),
		Run:     f.get("run"),
		Version: f.get("version"),
		Probes:  f.get("probes"),
	}, nil
}

// RuntimeMissing returns needs commands that are not on PATH, in needs order.
func (c *Catalog) RuntimeMissing(id string) ([]string, error) {
	runtime, err := c.Runtime(id)
	if err != nil {
		return nil, err
	}
	var missing []string
	for _, cmd := range ifsFields(runtime.Needs) {
		if _, err := exec.LookPath(cmd); err != nil {
			missing = append(missing, cmd)
		}
	}
	return missing, nil
}

// ProjectFamilies returns family source dirs of a project that exist under families/, sorted.
func (c *Catalog) ProjectFamilies(id string) ([]string, error) {
	entries, err := readDir(filepath.Join(c.root, "benches", id))
	if err != nil {
		return nil, err
	}
	known, err := c.ListFamilies()
	if err != nil {
		return nil, err
	}
	set := membership(known)
	var fams []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		if !isDir(filepath.Join(c.root, "benches", id, name)) {
			continue
		}
		if set[name] {
			fams = append(fams, name)
		}
	}
	return fams, nil
}

// ProjectRuntimes returns runtimes whose family is one of the project's families, in runtime-id order.
func (c *Catalog) ProjectRuntimes(id string) ([]string, error) {
	fams, err := c.ProjectFamilies(id)
	if err != nil {
		return nil, err
	}
	set := membership(fams)
	runtimes, err := c.ListRuntimes()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, runtimeID := range runtimes {
		runtime, err := c.Runtime(runtimeID)
		if err != nil {
			return nil, err
		}
		if set[runtime.Family] {
			out = append(out, runtimeID)
		}
	}
	return out, nil
}

func (c *Catalog) familyExt(fam string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(c.root, "families", fam, "runner.*"))
	if err != nil {
		return "", err
	}
	var files []string
	for _, m := range matches {
		if isFile(m) {
			files = append(files, m)
		}
	}
	if len(files) != 1 {
		return "", jerr("families/%s: expected exactly one runner.*, found %d", fam, len(files))
	}
	return strings.TrimPrefix(filepath.Ext(files[0]), "."), nil
}

func (c *Catalog) familyProbes(fam, ext string) ([]string, error) {
	dir := filepath.Join(c.root, "families", fam, "probes")
	if !isDir(dir) {
		return nil, nil
	}
	return approaches(dir, ext)
}

func (c *Catalog) claimedProbes() (map[string]bool, error) {
	runtimes, err := c.ListRuntimes()
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, id := range runtimes {
		runtime, err := c.Runtime(id)
		if err != nil {
			return nil, err
		}
		for _, p := range splitSpace(runtime.Probes) {
			set[p] = true
		}
	}
	return set, nil
}

func readDir(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return entries, err
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.IsDir()
}

func isFile(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Mode().IsRegular()
}

func membership(ids []string) map[string]bool {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func jerr(format string, args ...any) error {
	return fmt.Errorf("jmark: "+format, args...)
}

// bashWords matches `tr '\n' ' '` on a newline-terminated list: a trailing space when non-empty.
func bashWords(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return strings.Join(ids, " ") + " "
}

func ifsFields(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r'
	})
}

func splitSpace(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, " ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func approaches(dir, ext string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*."+ext))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, m := range matches {
		if !isFile(m) {
			continue
		}
		stem := strings.TrimSuffix(filepath.Base(m), "."+ext)
		if stem == "" || strings.HasPrefix(stem, "_") {
			continue
		}
		names = append(names, stem)
	}
	sort.Strings(names)
	return names, nil
}

func extFiles(dir, ext string) ([]string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*."+ext))
	if err != nil {
		return nil, err
	}
	var files []string
	for _, m := range matches {
		if isFile(m) {
			files = append(files, m)
		}
	}
	sort.Strings(files)
	return files, nil
}
