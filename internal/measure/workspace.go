package measure

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

// Run keeps build products alive across timing and probe passes, then removes them.
func (m *Measure) Run(plan benchmark.Plan) (benchmark.Measurement, error) {
	plan, cleanup, err := m.prepare(plan)
	if err != nil {
		return benchmark.Measurement{}, err
	}
	defer cleanup()
	rows := []benchmark.ApproachObservation{}
	index := map[string]int{}
	for _, collector := range []Collector{hyperfineCollector{m}, probeCollector{m}} {
		collected, err := collector.Collect(plan)
		if err != nil {
			return benchmark.Measurement{}, err
		}
		for _, row := range collected {
			i, ok := index[row.Name]
			if !ok {
				index[row.Name] = len(rows)
				rows = append(rows, row)
				continue
			}
			if rows[i].Observations == nil {
				rows[i].Observations = map[string]benchmark.Observation{}
			}
			if rows[i].Definitions == nil {
				rows[i].Definitions = map[string]benchmark.MetricDefinition{}
			}
			for id, o := range row.Observations {
				rows[i].Observations[id] = o
			}
			for id, d := range row.Definitions {
				rows[i].Definitions[id] = d
			}
		}
	}
	return benchmark.Measurement{Approaches: rows}, nil
}

func (m *Measure) prepare(plan benchmark.Plan) (benchmark.Plan, func(), error) {
	if plan.Params.Iters < 1 || plan.Params.Warmup < 0 || plan.Params.Runs < 1 || plan.Params.Warmup > int(^uint(0)>>1)-plan.Params.Iters {
		return benchmark.Plan{}, nil, fmt.Errorf("invalid measurement counts")
	}
	parent := filepath.Join(m.Root, ".jmark", "build")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return benchmark.Plan{}, nil, err
	}
	dir, err := os.MkdirTemp(parent, "run-")
	if err != nil {
		return benchmark.Plan{}, nil, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	// Copy before binding paths so callers can reuse a plan concurrently.
	plan.Approaches = append([]benchmark.Approach(nil), plan.Approaches...)
	bind := func(s string) string { return strings.ReplaceAll(s, "{build_dir}", dir) }
	bindOptional := func(s *string) *string {
		if s == nil {
			return nil
		}
		value := bind(*s)
		return &value
	}
	for i, approach := range plan.Approaches {
		approach.Bin = bind(approach.Bin)
		approach.BuildCmd = bindOptional(approach.BuildCmd)
		approach.TimedCmd = bind(approach.TimedCmd)
		approach.ProbeCmd = bindOptional(approach.ProbeCmd)
		plan.Approaches[i] = approach
		if err := m.Build(approach); err != nil {
			cleanup()
			return benchmark.Plan{}, nil, err
		}
	}
	return plan, cleanup, nil
}
