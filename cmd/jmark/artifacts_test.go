package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/artifact"
)

func TestVisibilityCommandsAndPicker(t *testing.T) {
	root := t.TempDir()
	private := makeRun(t, root, "demo", "run-one")
	out, errOut, code := runOut(t, root, "public", "demo/run-one")
	if code != 0 || errOut != "" || out != "artifacts/public/demo/run-one\n" {
		t.Fatalf("public: %d %q %q", code, out, errOut)
	}
	if _, err := artifact.Resolve(root, private); err == nil {
		t.Fatal("private run remains")
	}
	a := &app{root: root, fzf: func(input string, args []string) (string, []string, error) {
		if input != "public/demo/run-one\n" {
			t.Fatal("wrong picker scope", input)
		}
		return "", []string{"public/demo/run-one"}, nil
	}}
	out, errOut, code = finishApp(t, a, "private")
	if code != 0 || errOut != "" || out != "artifacts/private/demo/run-one\n" {
		t.Fatalf("private: %d %q %q", code, out, errOut)
	}
	a.fzf = func(string, []string) (string, []string, error) { return "q", nil, nil }
	_, _, code = finishApp(t, a, "public")
	if code != 0 {
		t.Fatal("cancel failed")
	}
	if _, err := os.Stat(filepath.Join(root, "artifacts", private, "result.ndjson")); err != nil {
		t.Fatal("cancel moved result")
	}
	out, errOut, code = runOut(t, root, "site")
	if code != 0 || errOut != "" || !strings.Contains(out, ".jmark/site/index.html") {
		t.Fatalf("site: %d %q %q", code, out, errOut)
	}
}
