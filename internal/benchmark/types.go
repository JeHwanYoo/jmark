// Package benchmark defines the values shared by planning, measurement, and result conversion.
package benchmark

import "encoding/json"

// Params is the bench.toml workload.
type Params struct {
	N      int      `json:"n"`
	Warmup int      `json:"warmup"`
	Iters  int      `json:"iters"`
	Runs   int      `json:"runs"`
	Probes []string `json:"probes"`
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

// Sample contains external wall-clock medians in seconds and a single-call sink.
type Sample struct {
	Name   string
	Sink   json.Number
	Lo, Hi float64
}

// ProbeSample contains counter medians from separate baseline and workload runs.
type ProbeSample struct {
	Name, Counter string
	Lo, Hi        float64
}

// Measurement holds timing and probe observations from the same execution workspace.
type Measurement struct {
	Samples []Sample
	Probes  []ProbeSample
}
