// Package benchmark defines the values shared by planning, measurement, and result conversion.
package benchmark

import "encoding/json"

// Params is the bench.toml workload.
type Params struct {
	N       int      `json:"n"`
	Warmup  int      `json:"warmup"`
	Iters   int      `json:"iters"`
	Runs    int      `json:"runs"`
	Metrics []string `json:"metrics,omitempty"`
	Probes  []string `json:"probes"`
}

// Plan describes one workload on one runtime before execution.
type Plan struct {
	Project           string      `json:"project"`
	Runtime           PlanRuntime `json:"runtime"`
	Family            PlanFamily  `json:"family"`
	Input             string      `json:"input"`
	Params            Params      `json:"params"`
	Probes            []PlanProbe `json:"probes"`
	ProbesUnavailable []string    `json:"probes_unavailable"`
	Approaches        []Approach  `json:"approaches"`
}

// PlanRuntime is the runtime slice of a Plan.
type PlanRuntime struct {
	ID         string   `json:"id"`
	VersionCmd string   `json:"version_cmd"`
	Family     string   `json:"family"`
	Needs      string   `json:"needs"`
	Probes     []string `json:"probes"`
}

// PlanFamily is the family slice of a Plan.
type PlanFamily struct {
	ID     string `json:"id"`
	Ext    string `json:"ext"`
	Runner string `json:"runner"`
}

// PlanProbe is one probe hook selected for a run.
type PlanProbe struct {
	Name string `json:"name"`
	Hook string `json:"hook"`
}

// Approach is one timed implementation inside a Plan.
type Approach struct {
	Name     string  `json:"name"`
	Path     string  `json:"path"`
	Bin      string  `json:"bin"`
	BuildCmd *string `json:"build_cmd"`
	TimedCmd string  `json:"timed_cmd"`
	ProbeCmd *string `json:"probe_cmd"`
}

// Sample holds collector observations and a single-call correctness sink.
type ApproachObservation struct {
	Name         string
	Sink         json.Number
	Definitions  map[string]MetricDefinition
	Observations map[string]Observation
}

// ProbeSample contains counter medians from separate baseline and workload runs.
type ProbeSample struct {
	Name, Counter   string
	Unit, Direction string
	Lo, Hi          float64
	Observation     Observation
}

// Measurement holds timing and probe observations from the same execution workspace.
type Measurement struct {
	Approaches []ApproachObservation
}

// MetricDefinition describes a scalar independently of its collector or presentation.
type MetricDefinition struct {
	Label     string `json:"label"`
	Unit      string `json:"unit"`
	Direction string `json:"direction"`
	Method    string `json:"method"`
	Scope     string `json:"scope"`
	Source    string `json:"source,omitempty"`
}
type Observation struct {
	Baseline []float64 `json:"baseline"`
	Workload []float64 `json:"workload"`
}
type MetricResult struct {
	Status string   `json:"status"`
	Value  *float64 `json:"value,omitempty"`
	Reason string   `json:"reason,omitempty"`
}

var DefaultMetrics = []string{"wall_time", "cpu_time", "peak_rss", "throughput"}
var Metrics = map[string]MetricDefinition{
	"wall_time":        {"Wall time", "s/op", "lower", "difference_per_op", "operation", "time_wall_clock"},
	"cpu_time":         {"CPU time", "s/op", "lower", "difference_per_op", "operation", "time_cpu"},
	"user_time":        {"User CPU time", "s/op", "lower", "difference_per_op", "operation", "time_user"},
	"system_time":      {"System CPU time", "s/op", "lower", "difference_per_op", "operation", "time_system"},
	"peak_rss":         {"Peak RSS", "B", "lower", "workload_median", "process_peak", "memory_peak_resident"},
	"throughput":       {"Throughput", "op/s", "higher", "inverse_wall_time", "operation", "time_wall_clock"},
	"instructions":     {"Instructions", "instructions/op", "lower", "difference_per_op", "hardware", "instructions"},
	"cpu_cycles":       {"CPU cycles", "cycles/op", "lower", "difference_per_op", "hardware", "cpu_cycles"},
	"cache_references": {"Cache references", "references/op", "neutral", "difference_per_op", "hardware", "cache_references"},
	"cache_misses":     {"Cache misses", "misses/op", "lower", "difference_per_op", "hardware", "cache_misses"},
	"branch_misses":    {"Branch misses", "misses/op", "lower", "difference_per_op", "hardware", "branch_misses"},
	"ipc":              {"IPC", "instructions/cycle", "neutral", "ratio", "hardware", ""},
	"cache_miss_rate":  {"Cache miss rate", "ratio", "lower", "ratio", "hardware", ""},
}

// Result is the persisted contract. Metric identities are extensible data.
type Result struct {
	SchemaVersion int                         `json:"schema_version"`
	Project       string                      `json:"project"`
	Runtime       string                      `json:"runtime"`
	Workload      map[string]any              `json:"workload"`
	Metrics       map[string]MetricDefinition `json:"metrics"`
	Results       []any                       `json:"results"`
	Meta          map[string]any              `json:"meta"`
}

func AttachProbes(samples []ApproachObservation, probes []ProbeSample) {
	for i := range samples {
		s := &samples[i]
		if s.Observations == nil {
			s.Observations = map[string]Observation{}
		}
		if s.Definitions == nil {
			s.Definitions = map[string]MetricDefinition{}
		}
		for _, p := range probes {
			if p.Name != s.Name {
				continue
			}
			id := "probe." + p.Counter
			s.Definitions[id] = MetricDefinition{Label: p.Counter, Unit: p.Unit + "/op", Direction: p.Direction, Method: "difference_per_op", Scope: "operation", Source: id}
			o := p.Observation
			if len(o.Baseline) == 0 && len(o.Workload) == 0 {
				o = Observation{Baseline: []float64{p.Lo}, Workload: []float64{p.Hi}}
			}
			s.Observations[id] = o
		}
	}
}
