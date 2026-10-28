// Package result converts benchmark observations into persisted results and validates them.
package result

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"runtime"
	"sort"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

// Build converts collector observations into metric-independent persisted results.
func Build(plan benchmark.Plan, observations benchmark.Measurement) (map[string]any, error) {
	if len(observations.Approaches) == 0 {
		return nil, fmt.Errorf("no samples")
	}
	if plan.Params.Iters < 1 {
		return nil, fmt.Errorf("iters must be positive")
	}
	ids := plan.Params.Metrics
	if len(ids) == 0 {
		ids = benchmark.DefaultMetrics
	}
	definitions := map[string]benchmark.MetricDefinition{}
	for _, id := range ids {
		d, ok := benchmark.Metrics[id]
		if !ok {
			return nil, fmt.Errorf("unknown metric %s", id)
		}
		if d.Scope == "hardware" {
			d.Scope = "hardware:" + runtime.GOOS
		}
		definitions[id] = d
	}
	for _, sample := range observations.Approaches {
		for id, d := range sample.Definitions {
			definitions[id] = d
		}
	}
	results := []any{}
	for _, r := range observations.Approaches {
		if !sameNumber(r.Sink, observations.Approaches[0].Sink) {
			return nil, fmt.Errorf("approaches disagree on sink")
		}
		raw := map[string]benchmark.Observation{}
		for id, o := range r.Observations {
			raw[id] = o
		}
		if len(raw) == 0 {
			return nil, fmt.Errorf("missing collector observations")
		}
		values := map[string]benchmark.MetricResult{}
		stats := map[string]any{}
		for id, o := range raw {
			for _, xs := range [][]float64{o.Baseline, o.Workload} {
				for _, v := range xs {
					if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
						return nil, fmt.Errorf("invalid %s sample", id)
					}
				}
			}
			stats[id] = map[string]any{"baseline": statistics(o.Baseline), "workload": statistics(o.Workload)}
		}
		for id, d := range definitions {
			v, reason := metricValue(d, raw, float64(plan.Params.Iters))
			if reason != "" {
				values[id] = benchmark.MetricResult{Status: "unavailable", Reason: reason}
			} else {
				values[id] = benchmark.MetricResult{Status: "ok", Value: &v}
			}
		}
		results = append(results, map[string]any{"name": r.Name, "sink": r.Sink, "metrics": values, "observations": raw, "statistics": stats})
	}
	doc := benchmark.Result{SchemaVersion: 2, Project: plan.Project, Runtime: plan.Runtime.ID, Workload: map[string]any{"parameters": map[string]any{"n": plan.Params.N}, "warmup": plan.Params.Warmup, "iters": plan.Params.Iters, "runs": plan.Params.Runs}, Metrics: definitions, Results: results, Meta: map[string]any{"collector": "hyperfine", "probes_unavailable": plan.ProbesUnavailable}}
	// The CLI's metadata composition boundary uses a map; values retain their typed forms.
	return map[string]any{"schema_version": doc.SchemaVersion, "project": doc.Project, "runtime": doc.Runtime, "workload": doc.Workload, "metrics": doc.Metrics, "results": doc.Results, "meta": doc.Meta}, nil
}

func midpoint(xs []float64) float64 {
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// missing separates readings the platform cannot report from ones withheld at collection time.
func missing(source string) string {
	if benchmark.ReadingKnown(source) && !benchmark.ReadingSupported(source, runtime.GOOS) {
		return "unsupported_on_platform"
	}
	return "not_reported_by_collector"
}
func difference(raw map[string]benchmark.Observation, source string, n float64) (float64, string) {
	o := raw[source]
	if len(o.Baseline) == 0 || len(o.Workload) == 0 {
		return 0, missing(source)
	}
	v := (midpoint(o.Workload) - midpoint(o.Baseline)) / n
	if v < 0 || (v == 0 && strings.HasPrefix(source, "time_")) {
		return 0, "below_noise_floor"
	}
	return v, ""
}
func metricValue(d benchmark.MetricDefinition, raw map[string]benchmark.Observation, n float64) (float64, string) {
	if d.Method == "workload_median" {
		o := raw[d.Source]
		if len(o.Workload) == 0 {
			return 0, missing(d.Source)
		}
		return midpoint(o.Workload), ""
	}
	if d.Method == "ratio" {
		x, e := difference(raw, d.Readings[0], n)
		if e != "" {
			return 0, e
		}
		y, e := difference(raw, d.Readings[1], n)
		if e != "" {
			return 0, e
		}
		if y <= 0 {
			return 0, "zero_denominator"
		}
		return x / y, ""
	}
	v, e := difference(raw, d.Source, n)
	if e != "" {
		return 0, e
	}
	if d.Method == "inverse_wall_time" {
		if v <= 0 {
			return 0, "zero_denominator"
		}
		return 1 / v, ""
	}
	return v, ""
}
func statistics(xs []float64) any {
	if len(xs) == 0 {
		return nil
	}
	sum := 0.0
	min, max := xs[0], xs[0]
	for _, x := range xs {
		sum += x
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
	}
	mean := sum / float64(len(xs))
	var sd, cv any
	if len(xs) > 1 {
		variance := 0.0
		for _, x := range xs {
			variance += (x - mean) * (x - mean)
		}
		v := math.Sqrt(variance / float64(len(xs)-1))
		sd = v
		if mean != 0 {
			cv = v / mean
		}
	}
	return map[string]any{"count": len(xs), "mean": mean, "median": midpoint(xs), "stddev": sd, "min": min, "max": max, "cv": cv}
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
