// Package artifact publishes complete benchmark runs to the local artifact store.
package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/JeHwanYoo/jmark/internal/catalog"
	"github.com/JeHwanYoo/jmark/internal/report"
)

// Write renders all outputs before staging them and returns the published entry path.
func Write(root, project string, ndjson []byte, format string, sources ...string) (string, error) {
	if !catalog.ValidID(project) {
		return "", fmt.Errorf("invalid artifact project %q", project)
	}
	if format != "ndjson" && format != "html" {
		return "", fmt.Errorf("internal: bad artifact mode %q", format)
	}
	env, err := report.Environment(ndjson)
	if err != nil {
		return "", err
	}
	sourceMarkdown := strings.Join(sources, "\n")
	files := map[string][]byte{"result.ndjson": ndjson, "env.json": env}
	if sourceMarkdown != "" {
		files["sources.md"] = []byte(sourceMarkdown)
	}
	entry := "result.ndjson"
	if format == "html" {
		page, err := report.RenderReportWithOptions(ndjson, report.Options{Sources: sourceMarkdown})
		if err != nil {
			return "", err
		}
		entry = "report.html"
		files[entry] = []byte(page)
	}
	parent := filepath.Join(root, "artifacts", "private", project)
	dir, err := publish(parent, func(stage string) error {
		for name, data := range files {
			if err := os.WriteFile(filepath.Join(stage, name), data, 0o644); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Join("artifacts", "private", project, filepath.Base(dir), entry)), nil
}

// publish keeps incomplete runs hidden and renames within the same filesystem.
func publish(parent string, write func(string) error) (string, error) {
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(parent, ".pending-"+time.Now().Format("20060102-150405")+"-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	if err := write(stage); err != nil {
		return "", err
	}
	dir := filepath.Join(parent, strings.TrimPrefix(filepath.Base(stage), ".pending-"))
	if err := os.Rename(stage, dir); err != nil {
		return "", err
	}
	return dir, nil
}
