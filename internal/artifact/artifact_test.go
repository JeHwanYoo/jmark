package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWrite(t *testing.T) {
	root := t.TempDir()
	rel, err := Write(root, "demo", []byte(parityNDJSON), "html")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, "artifacts/private/demo/") || !strings.HasSuffix(rel, "/report.html") {
		t.Fatalf("path %s", rel)
	}
	html, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), "report pending") || !strings.Contains(string(html), `id="jmark-data"`) {
		t.Fatalf("report missing chart payload")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.Dir(rel), "result.ndjson")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, filepath.Dir(rel), "env.json")); err != nil {
		t.Fatal(err)
	}
	ndRel, err := Write(root, "demo", []byte(parityNDJSON), "ndjson")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(ndRel) == filepath.Dir(rel) {
		t.Fatal("artifact directory reused")
	}
	if !strings.HasSuffix(ndRel, "/result.ndjson") {
		t.Fatalf("path %s", ndRel)
	}
	if _, err := Write(root, "demo", []byte(parityNDJSON), "stream"); err == nil {
		t.Fatal("expected bad mode")
	}
}

const parityNDJSON = `{"project":"demo","runtime":"node","metric":"us_per_op","results":[{"name":"for-loop","value":1.25,"iters":2,"total_ms":3.5,"sink":9},{"name":"map","value":4,"iters":2,"total_ms":8,"sink":9}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T00:00:00Z","runtime":"</script>","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}
{"project":"demo","runtime":"py","metric":"us_per_op","results":[{"name":"for-loop","value":5,"iters":2,"total_ms":10,"sink":9},{"name":"map","value":6.5,"iters":2,"total_ms":13,"sink":9}],"meta":{"n":10,"warmup":1,"iters":2,"runs":3,"captured_at":"2026-10-27T00:00:01Z","runtime":"3.12","host":{"os":"Darwin","arch":"arm64","ncpu":2}}}
`

func TestPublicationFailureCleansStage(t *testing.T) {
	parent := t.TempDir()
	_, err := publish(parent, func(dir string) error {
		if !strings.HasPrefix(filepath.Base(dir), ".pending-") {
			t.Fatal("visible staging directory")
		}
		if err := os.WriteFile(filepath.Join(dir, "result.ndjson"), []byte("partial"), 0644); err != nil {
			t.Fatal(err)
		}
		return os.ErrPermission
	})
	if !os.IsPermission(err) {
		t.Fatalf("failure lost: %v", err)
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 0 {
		t.Fatalf("partial publication: %v %v", entries, err)
	}
}

func TestInvalidInputDoesNotPublish(t *testing.T) {
	for _, input := range []string{parityNDJSON + "]", parityNDJSON + "}", parityNDJSON + "garbage"} {
		root := t.TempDir()
		if _, err := Write(root, "demo", []byte(input), "html"); err == nil {
			t.Fatal("malformed result accepted")
		}
		if _, err := os.Stat(filepath.Join(root, "artifacts")); !os.IsNotExist(err) {
			t.Fatal("failed report created artifacts")
		}
	}
	if _, err := Write(t.TempDir(), "../outside", []byte(parityNDJSON), "html"); err == nil {
		t.Fatal("invalid project accepted")
	}
}
