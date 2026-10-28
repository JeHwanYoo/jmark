package measure

import "github.com/JeHwanYoo/jmark/internal/benchmark"

// Collector returns normalized observations without owning build products or persistence.
type Collector interface {
	Collect(benchmark.Plan) ([]benchmark.ApproachObservation, error)
}
type hyperfineCollector struct{ measure *Measure }

func (c hyperfineCollector) Collect(plan benchmark.Plan) ([]benchmark.ApproachObservation, error) {
	return c.measure.timings(plan)
}

type probeCollector struct{ measure *Measure }

func (c probeCollector) Collect(plan benchmark.Plan) ([]benchmark.ApproachObservation, error) {
	probes, err := c.measure.probes(plan)
	if err != nil {
		return nil, err
	}
	rows := make([]benchmark.ApproachObservation, len(plan.Approaches))
	for i, a := range plan.Approaches {
		rows[i].Name = a.Name
	}
	benchmark.AttachProbes(rows, probes)
	return rows, nil
}
