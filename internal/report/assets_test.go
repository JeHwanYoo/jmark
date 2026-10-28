package report

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/JeHwanYoo/jmark/internal/benchmark"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedAssetsMatchSources(t *testing.T) {
	raw, err := os.ReadFile("ui/dist/manifest.json")
	if err != nil {
		t.Fatalf("UI build manifest: %v; run just ui-build", err)
	}
	var hashes map[string]string
	if err := json.Unmarshal(raw, &hashes); err != nil {
		t.Fatal(err)
	}
	files := []string{"../../../METRICS.md", "build.mjs", "package.json", "package-lock.json", "dist/app.js", "dist/app.css"}
	err = filepath.WalkDir("ui/src", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			rel, err := filepath.Rel("ui", path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != len(hashes) {
		t.Fatal("UI source list changed; run just ui-build")
	}
	for _, path := range files {
		body, err := os.ReadFile(filepath.Join("ui", filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if hashes[path] != hex.EncodeToString(sum[:]) {
			t.Errorf("UI build does not match %s; run just ui-build", path)
		}
	}
}

func TestMetricGuideCoversRegistry(t *testing.T) {
	document, err := os.ReadFile("../../METRICS.md")
	if err != nil {
		t.Fatal(err)
	}
	for id := range benchmark.Metrics {
		if !strings.Contains(string(document), "(`"+id+"`)") {
			t.Errorf("missing guide for %s", id)
		}
	}
}
