package artifact

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/report"
)

// BuildSite creates a static site exclusively from public result files.
func BuildSite(root string) (string, error) {
	runs, err := List(root, "public", "")
	if err != nil {
		return "", err
	}
	parent := filepath.Join(root, ".jmark")
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(parent, "site-build-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	var index []report.IndexRun
	for _, key := range runs {
		source, err := Resolve(root, key)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(filepath.Join(source, "result.ndjson"))
		if err != nil {
			return "", err
		}
		markdown, err := optionalFile(filepath.Join(source, "sources.md"))
		if err != nil {
			return "", err
		}
		html, err := report.RenderReportWithOptions(data, report.Options{IndexHref: "../../index.html", Sources: string(markdown)})
		if err != nil {
			return "", fmt.Errorf("%s: %w", key, err)
		}
		parts := strings.Split(key, "/")
		destination := filepath.Join(stage, parts[1], parts[2])
		if err := os.MkdirAll(destination, 0755); err != nil {
			return "", err
		}
		env, err := report.Environment(data)
		if err != nil {
			return "", err
		}
		files := map[string][]byte{"report.html": []byte(html), "result.ndjson": data, "env.json": env}
		if len(markdown) > 0 {
			files["sources.md"] = markdown
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(destination, name), body, 0644); err != nil {
				return "", err
			}
		}
		entry := report.IndexRun{Project: parts[1], Run: parts[2], Href: parts[1] + "/" + parts[2] + "/report.html", Runtimes: []string{}}
		scanner := bufio.NewScanner(bytes.NewReader(data))
		scanner.Buffer(make([]byte, 4096), 16*1024*1024)
		seen := map[string]bool{}
		for scanner.Scan() {
			if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
				continue
			}
			var doc struct {
				Runtime string `json:"runtime"`

				Meta struct {
					Captured string `json:"captured_at"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &doc); err != nil {
				return "", err
			}

			if !seen[doc.Runtime] {
				entry.Runtimes = append(entry.Runtimes, doc.Runtime)
				seen[doc.Runtime] = true
			}
			if entry.CapturedAt == "" {
				entry.CapturedAt = doc.Meta.Captured
			}
		}
		if err := scanner.Err(); err != nil {
			return "", err
		}
		index = append(index, entry)
	}
	page, err := report.RenderIndex(index)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(stage, "index.html"), []byte(page), 0644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(stage, ".nojekyll"), nil, 0644); err != nil {
		return "", err
	}
	destination := filepath.Join(parent, "site")
	if err := os.RemoveAll(destination); err != nil {
		return "", err
	}
	if err := os.Rename(stage, destination); err != nil {
		return "", err
	}
	return filepath.Join(".jmark", "site", "index.html"), nil
}

func optionalFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file: %s", path)
	}
	return os.ReadFile(path)
}
