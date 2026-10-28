package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	agentsStub = "TODO: subject, metric, input shape, fairness. Max 200 chars. See ../../AGENTS.md.\n"
	benchStub  = "n = 100000\nwarmup = 10\niters = 200\nruns = 5\nmetrics = \"wall_time cpu_time peak_rss throughput\"\n"
)

// Scaffold copies family templates into benches/<id>. Existing files are left in place.
func (c *Catalog) Scaffold(id string, families ...string) error {
	dir := filepath.Join(c.root, "benches", id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := writeNew(filepath.Join(dir, "AGENTS.md"), agentsStub); err != nil {
		return err
	}
	if err := writeNew(filepath.Join(dir, "bench.toml"), benchStub); err != nil {
		return err
	}
	for _, fam := range families {
		destDir := filepath.Join(dir, fam)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return err
		}
		tmpl := filepath.Join(c.root, "families", fam, "template")
		entries, err := os.ReadDir(tmpl)
		if err != nil {
			return jerr("families/%s/template: %s", fam, err.Error())
		}
		var names []string
		for _, e := range entries {
			// template/* does not match dotfiles.
			if strings.HasPrefix(e.Name(), ".") {
				continue
			}
			names = append(names, e.Name())
		}
		if len(names) == 0 {
			return jerr("families/%s/template: no files", fam)
		}
		for _, name := range names {
			src := filepath.Join(tmpl, name)
			if !isFile(src) {
				return jerr("families/%s/template/%s: not a file", fam, name)
			}
			body, err := os.ReadFile(src)
			if err != nil {
				return err
			}
			if err := writeNewBytes(filepath.Join(destDir, name), body); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeNew(path, body string) error {
	return writeNewBytes(path, []byte(body))
}

func writeNewBytes(path string, body []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(body)
	return err
}
