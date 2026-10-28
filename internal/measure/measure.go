// Package measure owns the clock and the build.
// Unit conversion belongs to internal/result.
package measure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/benchmark"
)

// Measure times approaches for one repo.
// Root is the absolute repo root. Binaries land in Root/.jmark/build.
// Stderr receives progress lines. A nil Stderr uses os.Stderr.
type Measure struct {
	Root   string
	Stderr io.Writer
}

// Exec splits cmd on spaces and runs it with stdin at EOF.
// The split is not a shell. Stdout is returned raw.
func Exec(cmd string) ([]byte, error) {
	return runArgv(cmd, os.Stderr)
}

// At replaces every {iters} placeholder. Other braces stay.
func At(cmd string, iters int) string {
	return strings.ReplaceAll(cmd, "{iters}", strconv.Itoa(iters))
}

// Sink runs timedCmd at one iteration count.
// Stdout must be a single JSON number. The returned text is that stdout
// with trailing newlines removed.
func (m *Measure) Sink(timedCmd string, iters int) (string, error) {
	rendered := At(timedCmd, iters)
	out, err := m.exec(rendered)
	if err != nil {
		return "", jerr("runner failed at iters=%d: %s", iters, rendered)
	}
	text := stripTrailingNewlines(string(out))
	if strings.Contains(text, "\n") || !jsonNumber(text) {
		return "", jerr("runner stdout must be one number, got: %s", clip(text, 200))
	}
	return text, nil
}

// Build runs approach.BuildCmd with bash when it is set.
// The previous bin and bin.* files are removed first.
func (m *Measure) Build(a benchmark.Approach) error {
	if a.BuildCmd == nil || *a.BuildCmd == "" {
		return nil
	}
	cmd := *a.BuildCmd
	if a.Bin != "" {
		removeBinFamily(a.Bin)
		if err := os.MkdirAll(filepath.Dir(a.Bin), 0o755); err != nil {
			return jerr("build failed: %s", cmd)
		}
	}
	c := exec.Command("bash", "-c", cmd)
	c.Stdin = nil
	c.Stdout = m.stderr()
	c.Stderr = m.stderr()
	if err := c.Run(); err != nil {
		return jerr("build failed: %s", cmd)
	}
	return nil
}

// Time runs hyperfine on timedCmd.
// {iters} stays literal so --parameter-list can substitute it.
// lo and hi are those parameter values. The results are wall-clock median seconds.
func (m *Measure) Time(cmd string, lo, hi, runs int) (float64, float64, error) {
	observations, err := m.collect(cmd, lo, hi, runs)
	if err != nil {
		return 0, 0, err
	}
	l, err := median(observations["time_wall_clock"].Baseline)
	if err != nil {
		return 0, 0, err
	}
	h, err := median(observations["time_wall_clock"].Workload)
	return l, h, err
}
func (m *Measure) collect(timedCmd string, lo, hi, runs int) (map[string]benchmark.Observation, error) {
	tmp, err := tempFile("jmark-hf-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmp)
	errPath := tmp + ".err"
	defer os.Remove(errPath)
	errFile, err := os.Create(errPath)
	if err != nil {
		return nil, err
	}
	c := exec.Command("hyperfine",
		"-N", "--metrics", "time_wall_clock", "--style", "none", "--warmup", "1",
		"--runs", strconv.Itoa(runs),
		"--output=null",
		"--parameter-list", "iters", strconv.Itoa(lo)+","+strconv.Itoa(hi),
		"--export-json", tmp,
		timedCmd,
	)
	c.Stdin = nil
	c.Stdout = nil
	c.Stderr = errFile
	runErr := c.Run()
	errFile.Close()
	if runErr != nil {
		if b, readErr := os.ReadFile(errPath); readErr == nil {
			fmt.Fprint(m.stderr(), string(b))
		}
		return nil, jerr("hyperfine failed: %s", timedCmd)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return nil, err
	}
	return parseObservations(data, lo, hi)
}

// Proportional returns nil when sink(i)/i equals sink(j)/j.
// Integer sinks compare by exact product. Other JSON numbers use a relative 1e-9 tolerance.
func (m *Measure) Proportional(timedCmd string, i, j int) error {
	a, err := m.Sink(timedCmd, i)
	if err != nil {
		return err
	}
	b, err := m.Sink(timedCmd, j)
	if err != nil {
		return err
	}
	if sinksProportional(a, b, i, j) {
		return nil
	}
	return jerr("sink not proportional to iters (sink(%d)=%s, sink(%d)=%s); does run() mutate its input?", i, a, j, b)
}

func (m *Measure) timings(plan benchmark.Plan) ([]benchmark.ApproachObservation, error) {
	w := plan.Params.Warmup
	hi := w + plan.Params.Iters
	// Proportional sampling uses at least one iteration. Wall time still uses
	// warmup, including zero.
	lo := w
	if lo <= 0 {
		lo = 1
	}
	var samples []benchmark.ApproachObservation
	for _, a := range plan.Approaches {
		fmt.Fprintf(m.stderr(), "→ %s %s %s\n", plan.Project, plan.Runtime.ID, a.Name)
		sink, err := m.Sink(a.TimedCmd, 1)
		if err != nil {
			return nil, err
		}
		observations, err := m.collect(a.TimedCmd, w, hi, plan.Params.Runs)
		if err != nil {
			return nil, err
		}
		if err := m.Proportional(a.TimedCmd, lo, hi); err != nil {
			return nil, fmt.Errorf("%w\njmark: %s %s: failed sink proportionality", err, plan.Runtime.ID, a.Name)
		}
		samples = append(samples, benchmark.ApproachObservation{Name: a.Name, Sink: json.Number(sink), Observations: observations})
	}
	return samples, nil
}

func (m *Measure) probes(plan benchmark.Plan) ([]benchmark.ProbeSample, error) {
	if len(plan.Probes) == 0 {
		return nil, nil
	}
	w := plan.Params.Warmup
	hi := w + plan.Params.Iters
	outPath, err := tempFile("jmark-probe-*")
	if err != nil {
		return nil, err
	}
	defer os.Remove(outPath)
	var samples []benchmark.ProbeSample
	for _, a := range plan.Approaches {
		if a.ProbeCmd == nil || *a.ProbeCmd == "" {
			return nil, jerr("%s: missing probe_cmd", a.Name)
		}
		fmt.Fprintf(m.stderr(), "→ probe %s\n", a.Name)
		rows, err := m.collectProbe(a.Name, *a.ProbeCmd, outPath, w, hi, plan.Params.Runs)
		if err != nil {
			return nil, err
		}
		text, err := summarizeProbes(a.Name, rows, w, hi)
		if err != nil {
			return nil, err
		}
		samples = append(samples, text...)
	}
	return samples, nil
}

// ProbeSelftest fails when any probe on the plan moves by more than 0.001 per op.
func (m *Measure) ProbeSelftest(plan benchmark.Plan) error {
	if len(plan.Probes) == 0 {
		return nil
	}
	plan, cleanup, err := m.prepare(plan)
	if err != nil {
		return err
	}
	defer cleanup()
	samples, err := m.probes(plan)
	if err != nil {
		return err
	}
	var bad []string
	for _, sample := range samples {
		per := (sample.Hi - sample.Lo) / float64(plan.Params.Iters)
		if math.Abs(per) > 0.001 {
			bad = append(bad, fmt.Sprintf("%s=%s/op", sample.Counter, formatFloat(per)))
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("biased probe on _null: %s", strings.Join(bad, ", "))
	}
	return nil
}

func (m *Measure) exec(cmd string) ([]byte, error) {
	return runArgv(cmd, m.stderr())
}

func (m *Measure) stderr() io.Writer {
	if m != nil && m.Stderr != nil {
		return m.Stderr
	}
	return os.Stderr
}

func runArgv(cmd string, stderr io.Writer) ([]byte, error) {
	args := splitSpaces(cmd)
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	c := exec.Command(args[0], args[1:]...)
	c.Stdin = nil
	var stdout bytes.Buffer
	c.Stdout = &stdout
	if stderr == nil {
		stderr = os.Stderr
	}
	c.Stderr = stderr
	err := c.Run()
	return stdout.Bytes(), err
}

func splitSpaces(cmd string) []string {
	parts := strings.Split(cmd, " ")
	args := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			args = append(args, p)
		}
	}
	return args
}

type probeObs struct {
	iters           int
	counter         string
	unit, direction string
	delta           float64
}

func (m *Measure) collectProbe(name, cmd, outPath string, lo, hi, runs int) ([]probeObs, error) {
	var rows []probeObs
	for _, k := range []int{lo, hi} {
		for r := 0; r < runs; r++ {
			if err := os.WriteFile(outPath, nil, 0o644); err != nil {
				return nil, err
			}
			expanded := At(strings.ReplaceAll(cmd, "{probe_out}", outPath), k)
			if _, err := m.exec(expanded); err != nil {
				return nil, jerr("probe run failed: %s iters=%d", name, k)
			}
			got, err := readProbeFile(outPath, k)
			if err != nil {
				return nil, err
			}
			rows = append(rows, got...)
		}
	}
	return rows, nil
}

func readProbeFile(path string, iters int) ([]probeObs, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rows []probeObs
	for _, line := range strings.Split(string(b), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) != 4 {
			return nil, fmt.Errorf("probe output must contain counter, unit, direction, delta")
		}
		if parts[1] == "" || (parts[2] != "lower" && parts[2] != "higher" && parts[2] != "neutral") {
			return nil, fmt.Errorf("invalid probe descriptor")
		}
		delta, err := strconv.ParseFloat(parts[3], 64)
		if err != nil || math.IsNaN(delta) || math.IsInf(delta, 0) {
			return nil, fmt.Errorf("invalid probe delta: %s", parts[3])
		}
		rows = append(rows, probeObs{iters: iters, counter: parts[0], unit: parts[1], direction: parts[2], delta: delta})
	}
	return rows, nil
}

func summarizeProbes(name string, rows []probeObs, lo, hi int) ([]benchmark.ProbeSample, error) {
	type pair struct {
		unit, direction string
		lo              []float64
		hi              []float64
	}
	groups := map[string]*pair{}
	for _, row := range rows {
		g, ok := groups[row.counter]
		if !ok {
			g = &pair{unit: row.unit, direction: row.direction}
			groups[row.counter] = g
		}
		if g.unit != row.unit || g.direction != row.direction {
			return nil, fmt.Errorf("probe descriptor changed for %s", row.counter)
		}
		if row.iters == lo {
			g.lo = append(g.lo, row.delta)
		}
		if row.iters == hi {
			g.hi = append(g.hi, row.delta)
		}
	}
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var samples []benchmark.ProbeSample
	for _, k := range keys {
		g := groups[k]
		l, err := median(g.lo)
		if err != nil {
			return nil, jerr("probe %s %s: empty lo sample", name, k)
		}
		h, err := median(g.hi)
		if err != nil {
			return nil, jerr("probe %s %s: empty hi sample", name, k)
		}
		samples = append(samples, benchmark.ProbeSample{Name: name, Counter: k, Unit: g.unit, Direction: g.direction, Lo: l, Hi: h, Observation: benchmark.Observation{Baseline: g.lo, Workload: g.hi}})
	}
	return samples, nil
}

func median(xs []float64) (float64, error) {
	if len(xs) == 0 {
		return 0, fmt.Errorf("empty")
	}
	s := append([]float64(nil), xs...)
	sort.Float64s(s)
	n := len(s)
	if n%2 == 1 {
		return s[n/2], nil
	}
	return (s[n/2-1] + s[n/2]) / 2, nil
}

var intLiteral = regexp.MustCompile(`^-?[0-9]+$`)

func sinksProportional(a, b string, i, j int) bool {
	if intLiteral.MatchString(a) && intLiteral.MatchString(b) {
		ai, ok1 := new(big.Int).SetString(a, 10)
		bi, ok2 := new(big.Int).SetString(b, 10)
		if !ok1 || !ok2 {
			return false
		}
		left := new(big.Int).Mul(ai, big.NewInt(int64(j)))
		right := new(big.Int).Mul(bi, big.NewInt(int64(i)))
		return left.Cmp(right) == 0
	}
	af, aok := parseJSONFloat(a)
	bf, bok := parseJSONFloat(b)
	if !aok || !bok {
		return false
	}
	return floatProportional(af, bf, float64(i), float64(j))
}

func parseJSONFloat(s string) (float64, bool) {
	var f float64
	if err := json.Unmarshal([]byte(s), &f); err != nil {
		return 0, false
	}
	return f, true
}

func floatProportional(a, b, i, j float64) bool {
	x := a / i
	y := b / j
	diff := math.Abs(x - y)
	scale := math.Max(math.Abs(x), math.Abs(y))
	if scale < 1e-300 {
		scale = 1e-300
	}
	return diff <= 1e-9*scale
}

func jsonNumber(s string) bool {
	dec := json.NewDecoder(strings.NewReader(s))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return false
	}
	if _, ok := v.(json.Number); !ok {
		return false
	}
	var extra any
	return dec.Decode(&extra) == io.EOF
}

func removeBinFamily(bin string) {
	os.Remove(bin)
	dir := filepath.Dir(bin)
	base := filepath.Base(bin)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	prefix := base + "."
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), prefix) {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}

func tempFile(pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	name := f.Name()
	if err := f.Close(); err != nil {
		os.Remove(name)
		return "", err
	}
	return name, nil
}

func formatFloat(f float64) string {
	return strconv.FormatFloat(f, 'g', -1, 64)
}

func stripTrailingNewlines(s string) string {
	for len(s) > 0 && s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	return s
}

func clip(s string, n int) string {
	b := []byte(s)
	if len(b) <= n {
		return s
	}
	return string(b[:n])
}

func jerr(format string, args ...any) error {
	return fmt.Errorf("jmark: "+format, args...)
}

func marshalCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
