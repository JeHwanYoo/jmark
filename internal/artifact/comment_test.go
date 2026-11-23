package artifact

import (
	"os"
	"path/filepath"
	"testing"
)

func TestComment(t *testing.T) {
	root := t.TempDir()
	rel, err := Write(root, "demo", []byte(parityNDJSON), "ndjson", "  first\n line\t")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, filepath.Dir(rel))
	if got := Comment(dir); got != "first line" {
		t.Fatalf("comment %q", got)
	}
	key := "private/demo/" + filepath.Base(dir)
	if err := SetComment(root, key, "second"); err != nil || Comment(dir) != "second" {
		t.Fatalf("set %v %q", err, Comment(dir))
	}
	moved, err := Move(root, key, "public")
	if err != nil || Comment(filepath.Join(root, "artifacts", moved)) != "second" {
		t.Fatalf("move lost comment: %v", err)
	}
	if err := SetComment(root, moved, " "); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", moved, CommentFile)); !os.IsNotExist(err) {
		t.Fatal("clear kept file")
	}
	if err := SetComment(root, "private/demo/missing", "x"); err == nil {
		t.Fatal("missing run accepted")
	}
	if err := os.MkdirAll(filepath.Join(root, "benches", "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "benches", "demo", "AGENTS.md"), []byte("\nCompare a vs b.\nMore.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := ProjectSummary(root, "demo"); got != "Compare a vs b." {
		t.Fatalf("summary %q", got)
	}
}
