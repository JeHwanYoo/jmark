package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVisibilityRoundTrip(t *testing.T) {
	root := t.TempDir()
	rel, err := Write(root, "demo", []byte(parityNDJSON), "ndjson", "## Source\n\n```text\nsnapshot\n```\n")
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimPrefix(filepath.ToSlash(filepath.Dir(rel)), "artifacts/")
	if !strings.HasPrefix(key, "private/") {
		t.Fatal("new run is public", key)
	}
	public, err := Move(root, key, "public")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(root, key); err == nil {
		t.Fatal("old private run remains")
	}
	if body, err := os.ReadFile(filepath.Join(root, "artifacts", public, "sources.md")); err != nil || !strings.Contains(string(body), "snapshot") {
		t.Fatalf("source lost: %q %v", body, err)
	}
	listed, err := List(root, "public", "demo")
	if err != nil || len(listed) != 1 || listed[0] != public {
		t.Fatalf("public listing: %v %v", listed, err)
	}
	restored, err := Move(root, public, "private")
	if err != nil || restored != key {
		t.Fatalf("restore: %s %v", restored, err)
	}
	if data, err := os.ReadFile(filepath.Join(root, rel)); err != nil || string(data) != parityNDJSON {
		t.Fatalf("result changed: %s %v", data, err)
	}
}

func TestVisibilityRejectsOverwriteAndTraversal(t *testing.T) {
	root := t.TempDir()
	rel, err := Write(root, "demo", []byte(parityNDJSON), "ndjson")
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimPrefix(filepath.ToSlash(filepath.Dir(rel)), "artifacts/")
	destination := filepath.Join(root, "artifacts", strings.Replace(key, "private/", "public/", 1))
	if err := os.MkdirAll(destination, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Move(root, key, "public"); err == nil {
		t.Fatal("overwrote destination")
	}
	if _, err := Resolve(root, key); err != nil {
		t.Fatal("source lost", err)
	}
	for _, bad := range []string{"../outside", "private/demo/../outside", "private/demo/.pending-run", "demo/run", "private/demo/run/extra"} {
		if _, err := Move(root, bad, "public"); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
	link := filepath.Join(root, "artifacts", "private", "demo", "link")
	if err := os.Symlink(filepath.Dir(filepath.Join(root, rel)), link); err != nil {
		t.Fatal(err)
	}
	if _, err := Move(root, "private/demo/link", "public"); err == nil {
		t.Fatal("accepted symbolic run")
	}
}

func TestPublicSiteContainsOnlyPublicSnapshots(t *testing.T) {
	root := t.TempDir()
	private := strings.ReplaceAll(parityNDJSON, "demo", "PRIVATE-ONLY-SENTINEL")
	if _, err := Write(root, "secret", []byte(private), "html"); err != nil {
		t.Fatal(err)
	}
	markdown := "## loop.src\n\n```text\nmeasured-source\n```\n"
	rel, err := Write(root, "demo", []byte(parityNDJSON), "ndjson", markdown)
	if err != nil {
		t.Fatal(err)
	}
	key := strings.TrimPrefix(filepath.ToSlash(filepath.Dir(rel)), "artifacts/")
	public, err := Move(root, key, "public")
	if err != nil {
		t.Fatal(err)
	}
	index, err := BuildSite(root)
	if err != nil {
		t.Fatal(err)
	}
	site := filepath.Dir(filepath.Join(root, index))
	pagePath := filepath.Join(site, strings.TrimPrefix(public, "public/"), "report.html")
	page, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `data-index-href="../../index.html"`) || !strings.Contains(string(page), "measured-source") {
		t.Fatal("missing index navigation or snapshot")
	}
	indexBody, err := os.ReadFile(filepath.Join(root, index))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(indexBody), strings.TrimPrefix(public, "public/")+"/report.html") {
		t.Fatal("index link missing")
	}
	if err := filepath.WalkDir(site, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(body), "PRIVATE-ONLY-SENTINEL") {
			t.Errorf("private data leaked to %s", path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(filepath.Join(filepath.Dir(pagePath), "sources.md"))
	if err != nil || string(stored) != markdown {
		t.Fatal("snapshot changed")
	}
	if _, err := Move(root, public, "private"); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildSite(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pagePath); !os.IsNotExist(err) {
		t.Fatal("stale public report remains")
	}
	if _, err := os.Stat(filepath.Join(site, ".nojekyll")); err != nil {
		t.Fatal(err)
	}
}
