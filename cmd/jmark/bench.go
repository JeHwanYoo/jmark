package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/JeHwanYoo/jmark/internal/artifact"
	"github.com/JeHwanYoo/jmark/internal/catalog"
	"github.com/JeHwanYoo/jmark/internal/measure"
	"github.com/JeHwanYoo/jmark/internal/result"
	"github.com/JeHwanYoo/jmark/internal/source"
)

const benchUsage = `usage: bench [<id>] [--runtime <runtime>] [--format stream|ndjson|html]

  <id>       project id; omit on a terminal to choose project, runtime, and format with fzf
  --runtime     run one runtime (default: every available runtime of the project)
  --format   stream  Result NDJSON on stdout (default, pipeline)
             ndjson  write artifacts/private/<id>/<timestamp>/result.ndjson; path on stdout
             html    write result.ndjson + report.html under that dir; html path on stdout
`

var errBenchHelp = errors.New("bench help")

type benchOpts struct {
	id         string
	runtime    string
	runtimeSet bool
	format     string
	formatSet  bool
}

func (a *app) cmdBench(args []string) error {
	opts, err := parseBench(args)
	if errors.Is(err, errBenchHelp) {
		fmt.Fprint(a.stdout, benchUsage)
		return nil
	}
	if err != nil {
		return err
	}
	switch opts.format {
	case "stream", "ndjson", "html":
	default:
		return fmt.Errorf("bench: invalid --format '%s' (want: stream|ndjson|html)", opts.format)
	}
	interactive := opts.id == ""
	if interactive {
		if !a.stdinTTY() {
			fmt.Fprint(a.stderr, benchUsage)
			return fmt.Errorf("bench: id is required")
		}
		id, err := a.benchPickID()
		if err != nil {
			return err
		}
		opts.id = id
	}
	if !catalog.ValidID(opts.id) {
		return fmt.Errorf("invalid project id '%s'", opts.id)
	}
	if !a.cat.ProjectExists(opts.id) {
		return fmt.Errorf("unknown project '%s' (see: just list)", opts.id)
	}
	if err := a.cat.CheckRegistry(); err != nil {
		a.writeErr(err)
		return fmt.Errorf("registry errors; run: just doctor")
	}
	if err := a.cat.CheckProject(opts.id); err != nil {
		a.writeErr(err)
		return fmt.Errorf("project %s has layout errors", opts.id)
	}
	if interactive {
		opts, err = a.benchPickOptions(opts)
		if err != nil {
			return err
		}
	}
	runtimes, err := a.benchRuntimes(opts)
	if err != nil {
		return err
	}
	hostSnap, err := measure.HostSnapshot()
	if err != nil {
		a.writeErr(err)
		return fmt.Errorf("could not capture host snapshot")
	}
	m := &measure.Measure{Root: a.root, Stderr: a.stderr}
	sources := source.Snapshot{}
	var lines [][]byte
	var objs []map[string]any
	for _, runtime := range runtimes {
		plan, err := a.cat.Plan(opts.id, runtime)
		if err != nil {
			a.writeErr(err)
			return fmt.Errorf("%s: could not build plan; no results printed", runtime)
		}
		snapshot, err := source.Capture(a.root, plan)
		if err != nil {
			return err
		}
		if err := sources.Merge(snapshot); err != nil {
			return err
		}
		observations, err := m.Run(plan)
		if err != nil {
			a.writeErr(err)
			return fmt.Errorf("%s failed; no results printed", runtime)
		}
		if err := sources.Verify(a.root); err != nil {
			return err
		}
		runtimeVersion, err := measure.RuntimeVersion(plan.Runtime.VersionCmd)
		if err != nil {
			a.writeErr(err)
			return fmt.Errorf("%s: result failed; no results printed", runtime)
		}
		obj, err := result.Build(plan, observations)
		if err != nil {
			a.writeErr(err)
			return fmt.Errorf("%s: result failed; no results printed", runtime)
		}
		meta, _ := obj["meta"].(map[string]any)
		if meta == nil {
			return fmt.Errorf("%s: result failed; no results printed", runtime)
		}
		meta["host"] = json.RawMessage(hostSnap)
		meta["runtime"] = runtimeVersion
		meta["captured_at"] = time.Now().UTC().Format("2006-01-02T15:04:05Z")
		if err := result.CheckSchema(obj); err != nil {
			a.writeErr(err)
			return fmt.Errorf("%s: result failed; no results printed", runtime)
		}
		line, err := marshalCompact(obj)
		if err != nil {
			a.writeErr(err)
			return fmt.Errorf("%s: result failed; no results printed", runtime)
		}
		lines = append(lines, line)
		objs = append(objs, obj)
	}
	if msg := sinkDiff(objs); msg != "" {
		a.warn("sinks differ across runtimes: %s", msg)
	}
	switch opts.format {
	case "stream":
		return a.printLines(lines)
	default:
		rel, err := a.writeArtifacts(opts.id, opts.format, lines, sources.Markdown())
		if err != nil {
			a.writeErr(err)
			return fmt.Errorf("could not write artifacts")
		}
		fmt.Fprintln(a.stdout, rel)
		return nil
	}
}

func (a *app) benchRuntimes(opts benchOpts) ([]string, error) {
	if opts.runtimeSet {
		if !a.cat.RuntimeExists(opts.runtime) {
			return nil, a.unknownRuntime(opts.runtime)
		}
		have, err := a.cat.ProjectRuntimes(opts.id)
		if err != nil {
			return nil, err
		}
		if !hasWord(have, opts.runtime) {
			rec, err := a.cat.Runtime(opts.runtime)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("runtime %s has no sources in %s (needs benches/%s/%s/)", opts.runtime, opts.id, opts.id, rec.Family)
		}
		missing, err := a.cat.RuntimeMissing(opts.runtime)
		if err != nil {
			return nil, err
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("runtime %s unavailable (missing %s)", opts.runtime, strings.Join(missing, " "))
		}
		return []string{opts.runtime}, nil
	}
	ids, err := a.cat.ProjectRuntimes(opts.id)
	if err != nil {
		return nil, err
	}
	var runtimes []string
	for _, runtime := range ids {
		missing, err := a.cat.RuntimeMissing(runtime)
		if err != nil {
			return nil, err
		}
		if len(missing) > 0 {
			a.warn("skip: runtime %s unavailable (missing %s)", runtime, strings.Join(missing, " "))
			continue
		}
		runtimes = append(runtimes, runtime)
	}
	if len(runtimes) == 0 {
		return nil, fmt.Errorf("no available runtimes for %s", opts.id)
	}
	return runtimes, nil
}

func parseBench(args []string) (benchOpts, error) {
	o := benchOpts{format: "stream"}
	positional := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if positional {
			if err := o.addID(a); err != nil {
				return o, err
			}
			continue
		}
		switch {
		case a == "-h" || a == "--help":
			return o, errBenchHelp
		case a == "--runtime":
			if i+1 >= len(args) {
				return o, fmt.Errorf("bench: --runtime needs a value")
			}
			if o.runtimeSet {
				return o, fmt.Errorf("bench: --runtime given more than once")
			}
			o.runtime = args[i+1]
			o.runtimeSet = true
			i++
		case strings.HasPrefix(a, "--runtime="):
			if o.runtimeSet {
				return o, fmt.Errorf("bench: --runtime given more than once")
			}
			o.runtime = strings.TrimPrefix(a, "--runtime=")
			if o.runtime == "" {
				return o, fmt.Errorf("bench: --runtime needs a value")
			}
			o.runtimeSet = true
		case a == "--format":
			if i+1 >= len(args) {
				return o, fmt.Errorf("bench: --format needs a value")
			}
			o.formatSet = true
			o.format = args[i+1]
			i++
		case strings.HasPrefix(a, "--format="):
			o.formatSet = true
			o.format = strings.TrimPrefix(a, "--format=")
			if o.format == "" {
				return o, fmt.Errorf("bench: --format needs a value")
			}
		case a == "--":
			positional = true
		case strings.HasPrefix(a, "-"):
			return o, fmt.Errorf("bench: unknown option '%s' (see: bench --help)", a)
		default:
			if err := o.addID(a); err != nil {
				return o, err
			}
		}
	}
	return o, nil
}

func (o *benchOpts) addID(arg string) error {
	if o.id != "" {
		return fmt.Errorf("bench: unexpected argument '%s' (see: bench --help)", arg)
	}
	o.id = arg
	return nil
}

func (a *app) printLines(lines [][]byte) error {
	for _, line := range lines {
		if _, err := a.stdout.Write(append(append([]byte{}, line...), '\n')); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) writeArtifacts(id, format string, lines [][]byte, sources ...string) (string, error) {
	var body bytes.Buffer
	for _, line := range lines {
		body.Write(line)
		body.WriteByte('\n')
	}
	return artifact.Write(a.root, id, body.Bytes(), format, sources...)
}

func sinkDiff(objs []map[string]any) string {
	if len(objs) == 0 {
		return ""
	}
	var parts []string
	uniq := map[string]struct{}{}
	for _, obj := range objs {
		runtime, _ := obj["runtime"].(string)
		text := firstSink(obj)
		parts = append(parts, runtime+"="+text)
		uniq[text] = struct{}{}
	}
	if len(uniq) <= 1 {
		return ""
	}
	return strings.Join(parts, " ")
}

func firstSink(obj map[string]any) string {
	results, _ := obj["results"].([]any)
	if len(results) == 0 {
		return ""
	}
	row, _ := results[0].(map[string]any)
	if row == nil {
		return ""
	}
	b, err := json.Marshal(row["sink"])
	if err != nil {
		return ""
	}
	return string(b)
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

func hasWord(words []string, w string) bool {
	for _, x := range words {
		if x == w {
			return true
		}
	}
	return false
}
