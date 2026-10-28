//go:build live

package measure

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/catalog"
)

func TestLiveSumLargeDoublesNode(t *testing.T) {
	root := repoRoot(t)
	plan, err := catalog.New(root).Plan("sum-large-doubles", "node")
	if err != nil {
		t.Fatal(err)
	}
	doc := plan
	if doc.Params.Runs != 10 {
		t.Fatalf("runs %d, bench.toml default is 10", doc.Params.Runs)
	}
	var stderr bytes.Buffer
	m := &Measure{Root: root, Stderr: &stderr}
	observations, err := m.Run(plan)
	if err != nil {
		t.Fatalf("plan: %v\nstderr: %s", err, stderr.String())
	}
	lines := sampleLines(observations.Samples)
	if len(lines) != len(doc.Approaches) || len(lines) != 2 {
		t.Fatalf("lines %#v approaches %d", lines, len(doc.Approaches))
	}
	var sink string
	for i, line := range lines {
		f := strings.Split(line, "\t")
		if len(f) != 4 || f[0] != doc.Approaches[i].Name {
			t.Fatalf("line %q", line)
		}
		if i == 0 {
			sink = f[1]
		} else if f[1] != sink {
			t.Fatalf("sinks %q and %q", sink, f[1])
		}
		for _, col := range f[2:] {
			v, err := strconv.ParseFloat(col, 64)
			if err != nil || v <= 0 {
				t.Fatalf("time %q in %q", col, line)
			}
		}
		if !strings.Contains(stderr.String(), "→ sum-large-doubles node "+f[0]+"\n") {
			t.Fatalf("progress %q", stderr.String())
		}
	}
}
