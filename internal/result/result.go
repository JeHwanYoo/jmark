// Package result converts benchmark observations into persisted results and validates them.
package result

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

// Build converts observations into one Result. Metadata about the host is added by the caller.
func Build(plan benchmark.Plan, observations benchmark.Measurement) (map[string]any, error) {
	rows := observations.Samples
	if len(rows) == 0 {
		return nil, fmt.Errorf("no samples")
	}
	if plan.Params.Iters < 1 {
		return nil, fmt.Errorf("iters must be positive")
	}
	sink0 := rows[0].Sink
	for _, r := range rows {
		if !sameNumber(r.Sink, sink0) {
			var parts []string
			for _, x := range rows {
				parts = append(parts, fmt.Sprintf("%s=%v", x.Name, x.Sink))
			}
			return nil, fmt.Errorf("approaches disagree on sink: %s", strings.Join(parts, ", "))
		}
	}

	results := make([]any, 0, len(rows))
	for _, r := range rows {
		if r.Lo < 0 || r.Hi < 0 || math.IsNaN(r.Lo) || math.IsNaN(r.Hi) || math.IsInf(r.Lo, 0) || math.IsInf(r.Hi, 0) {
			return nil, fmt.Errorf("%s: invalid timing sample", r.Name)
		}
		d := r.Hi - r.Lo
		if d <= 0 {
			return nil, fmt.Errorf("%s: below noise floor (T[W+N] <= T[W]); raise iters in bench.toml", r.Name)
		}
		results = append(results, map[string]any{
			"name":     r.Name,
			"value":    d / float64(plan.Params.Iters) * 1e6,
			"iters":    plan.Params.Iters,
			"total_ms": d * 1e3,
			"sink":     r.Sink,
		})
	}

	meta := map[string]any{
		"n":      plan.Params.N,
		"warmup": plan.Params.Warmup,
		"iters":  plan.Params.Iters,
		"runs":   plan.Params.Runs,
		"clock":  "hyperfine",
		"method": "differential-median",
	}
	if len(plan.Params.Probes) > 0 {
		ps := observations.Probes
		byName := map[string][]benchmark.ProbeSample{}
		order := []string{}
		seen := map[string]bool{}
		for _, p := range ps {
			if !seen[p.Name] {
				seen[p.Name] = true
				order = append(order, p.Name)
			}
			byName[p.Name] = append(byName[p.Name], p)
		}
		probesOut := map[string]any{}
		for _, name := range order {
			group := byName[name]
			entry := map[string]any{}
			for _, p := range group {
				entry[p.Counter+"_per_op"] = (p.Hi - p.Lo) / float64(plan.Params.Iters)
			}
			probesOut[name] = entry
		}
		meta["probe_method"] = "differential-snapshot"
		meta["probes"] = probesOut
		// Preserve unavailable probe names even when the list is empty.
		una := plan.ProbesUnavailable
		if una == nil {
			una = []string{}
		}
		meta["probes_unavailable"] = una
	}

	return map[string]any{
		"project": plan.Project,
		"runtime": plan.Runtime.ID,
		"metric":  "us_per_op",
		"results": results,
		"meta":    meta,
	}, nil
}

// MarshalCanonical sorts object keys while preserving JSON number precision.
func MarshalCanonical(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var normalized any
	if err := decoder.Decode(&normalized); err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

// sameNumber preserves exact sink agreement, including integers beyond float precision.
func sameNumber(a, b json.Number) bool {
	x, okX := new(big.Rat).SetString(string(a))
	y, okY := new(big.Rat).SetString(string(b))
	return okX && okY && x.Cmp(y) == 0
}
