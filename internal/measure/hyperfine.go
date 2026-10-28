package measure

import (
	"encoding/json"
	"fmt"
	"github.com/JeHwanYoo/jmark/internal/benchmark"
	"math"
	"strconv"
	"strings"
)

// parseObservations retains every execution rather than discarding distributions.
func parseObservations(data []byte, lo, hi int) (map[string]benchmark.Observation, error) {
	var export struct {
		Version int `json:"schema_version"`
		Results []struct {
			Parameters map[string]struct {
				Value string `json:"value"`
			} `json:"parameters"`
			Measurements []map[string]json.RawMessage `json:"measurements"`
		} `json:"results"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, err
	}
	if export.Version != 2 {
		return nil, fmt.Errorf("hyperfine export schema 2 required (install hyperfine >=2.0.0)")
	}
	out := map[string]benchmark.Observation{}
	seen := map[int]bool{}
	for _, group := range export.Results {
		k, err := strconv.Atoi(group.Parameters["iters"].Value)
		if err != nil || (k != lo && k != hi) {
			continue
		}
		if seen[k] {
			return nil, fmt.Errorf("duplicate iters=%d", k)
		}
		seen[k] = true
		if len(group.Measurements) == 0 {
			return nil, fmt.Errorf("empty measurements at iters=%d", k)
		}
		for _, run := range group.Measurements {
			for id, raw := range run {
				if id == "exit_code" {
					var code int
					if err := json.Unmarshal(raw, &code); err != nil || code != 0 {
						return nil, fmt.Errorf("runner failed")
					}
					continue
				}
				var reading struct {
					Value *float64 `json:"value"`
					Unit  string   `json:"unit"`
				}
				if err := json.Unmarshal(raw, &reading); err != nil {
					return nil, err
				}
				if reading.Value == nil {
					continue
				}
				if strings.HasPrefix(id, "time_") && reading.Unit != "second" {
					return nil, fmt.Errorf("unexpected unit for %s", id)
				}
				if id == "memory_peak_resident" && reading.Unit != "byte" {
					return nil, fmt.Errorf("unexpected RSS unit")
				}
				value := *reading.Value
				if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
					return nil, fmt.Errorf("invalid %s sample", id)
				}
				o := out[id]
				if k == lo {
					o.Baseline = append(o.Baseline, value)
				}
				if k == hi {
					o.Workload = append(o.Workload, value)
				}
				out[id] = o
			}
		}
	}
	if !seen[lo] || !seen[hi] {
		return nil, fmt.Errorf("missing iters=%d or iters=%d", lo, hi)
	}
	if len(out["time_wall_clock"].Baseline) == 0 || len(out["time_wall_clock"].Workload) == 0 {
		return nil, fmt.Errorf("missing wall clock samples")
	}
	return out, nil
}
