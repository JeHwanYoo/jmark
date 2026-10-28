package result_test

import (
	"encoding/json"
	"github.com/JeHwanYoo/jmark/internal/benchmark"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/JeHwanYoo/jmark/internal/result"
)

func TestBuildMatchesGolden(t *testing.T) {
	for _, name := range []string{"plain", "probed"} {
		t.Run(name, func(t *testing.T) {
			dir := filepath.Join("testdata", "golden", name)
			plan, err := os.ReadFile(filepath.Join(dir, "plan.json"))
			if err != nil {
				t.Fatal(err)
			}
			samples, err := os.ReadFile(filepath.Join(dir, "samples.tsv"))
			if err != nil {
				t.Fatal(err)
			}
			probes, err := os.ReadFile(filepath.Join(dir, "probes.tsv"))
			if err != nil {
				t.Fatal(err)
			}
			wantRaw, err := os.ReadFile(filepath.Join(dir, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			var execution benchmark.Plan
			if err := json.Unmarshal(plan, &execution); err != nil {
				t.Fatal(err)
			}
			observations := fixtureSamples(t, string(samples), string(probes))
			got, err := result.Build(execution, observations)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			if err := result.CheckSchema(got); err != nil {
				t.Fatalf("schema: %v", err)
			}
			gotJSON, err := result.MarshalCanonical(got)
			if err != nil {
				t.Fatal(err)
			}
			var want map[string]any
			if err := json.Unmarshal(wantRaw, &want); err != nil {
				t.Fatal(err)
			}
			if err := result.CheckSchema(want); err != nil {
				t.Fatalf("expected schema: %v", err)
			}
			wantJSON, err := result.MarshalCanonical(want)
			if err != nil {
				t.Fatal(err)
			}
			if string(gotJSON) != string(wantJSON) {
				t.Fatalf("golden mismatch\n got: %s\nwant: %s", gotJSON, wantJSON)
			}
		})
	}
}

func fixtureSamples(t *testing.T, samples, probes string) benchmark.Measurement {
	t.Helper()
	var out benchmark.Measurement
	parse := func(s string) float64 {
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	for _, line := range strings.Split(strings.TrimSpace(samples), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 4 {
			t.Fatalf("invalid sample fixture %q", line)
		}
		out.Samples = append(out.Samples, benchmark.Sample{Name: f[0], Sink: json.Number(f[1]), Lo: parse(f[2]), Hi: parse(f[3])})
	}
	for _, line := range strings.Split(strings.TrimSpace(probes), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 4 {
			t.Fatalf("invalid probe fixture %q", line)
		}
		out.Probes = append(out.Probes, benchmark.ProbeSample{Name: f[0], Counter: f[1], Lo: parse(f[2]), Hi: parse(f[3])})
	}
	return out
}

func TestSchemaRejectsUnknownFieldsAndNonfiniteNumbers(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "golden", "plain", "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(map[string]any){
		"unknown root":     func(v map[string]any) { v["extra"] = true },
		"unknown row":      func(v map[string]any) { v["results"].([]any)[0].(map[string]any)["extra"] = true },
		"infinite":         func(v map[string]any) { v["results"].([]any)[0].(map[string]any)["value"] = math.Inf(1) },
		"fractional count": func(v map[string]any) { v["meta"].(map[string]any)["iters"] = 1.5 },
		"missing field":    func(v map[string]any) { delete(v, "runtime") },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			var doc map[string]any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatal(err)
			}
			change(doc)
			if err := result.CheckSchema(doc); err == nil {
				t.Fatal("invalid result accepted")
			}
		})
	}
}

func TestBuildExactSinkAgreement(t *testing.T) {
	plan := benchmark.Plan{Params: benchmark.Params{Iters: 1}}
	for _, tc := range []struct {
		a, b  string
		valid bool
	}{
		{"9007199254740992", "9007199254740993", false},
		{"9007199254740993", "9007199254740993", true},
		{"1", "1.0", true},
	} {
		observations := benchmark.Measurement{Samples: []benchmark.Sample{
			{Name: "a", Sink: json.Number(tc.a), Lo: 1, Hi: 2},
			{Name: "b", Sink: json.Number(tc.b), Lo: 1, Hi: 2},
		}}
		got, err := result.Build(plan, observations)
		if (err == nil) != tc.valid {
			t.Fatalf("sinks %s %s: %v", tc.a, tc.b, err)
		}
		if tc.valid {
			sink := got["results"].([]any)[0].(map[string]any)["sink"]
			if sink != json.Number(tc.a) {
				t.Fatalf("sink lost precision: %v", sink)
			}
		}
	}
}
