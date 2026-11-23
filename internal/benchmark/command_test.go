package benchmark

import (
	"slices"
	"testing"
)

func TestQuoteArgRoundTrip(t *testing.T) {
	for _, arg := range []string{"plain", "{build_dir}/loop", "/repo/0. WORKING/runner.rs", "it's", "a\"b", "", "  "} {
		cmd := "run " + QuoteArg(arg) + " 3"
		if got := SplitArgs(cmd); !slices.Equal(got, []string{"run", arg, "3"}) {
			t.Fatalf("%q split into %q", cmd, got)
		}
	}
	if QuoteArg("/plain/path") != "/plain/path" {
		t.Fatal("plain path quoted")
	}
}

func TestSplitArgsJoinsAdjacentParts(t *testing.T) {
	got := SplitArgs("'/a b'/loop  {iters}")
	if !slices.Equal(got, []string{"/a b/loop", "{iters}"}) {
		t.Fatalf("%q", got)
	}
}
