// Package report projects saved results into standalone HTML and environment metadata.
package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// CompareRun is one result.ndjson file.
// Label is the run id on the chart, the same project/timestamp string `open` prints.
type CompareRun struct {
	Label string
	Path  string
}

// RenderReport builds the single-run HTML page from Result NDJSON.
type Options struct {
	IndexHref string
	Sources   string
}

func RenderReport(ndjson []byte) (string, error) {
	return RenderReportWithOptions(ndjson, Options{})
}

func RenderReportWithOptions(ndjson []byte, options Options) (string, error) {
	rows, project, err := flatten(ndjson, "")
	if err != nil {
		return "", err
	}
	return fillPageData(project, string(joinArray(rows)), "jmark-data", options), nil
}

// RenderCompare builds the multi-run HTML page.
// A missing path is skipped. The page fails when no result rows remain.
func RenderCompare(runs []CompareRun) (string, error) {
	var all [][]byte
	var sources strings.Builder
	project := "jmark"
	for _, run := range runs {
		if run.Path == "" {
			continue
		}
		body, err := os.ReadFile(run.Path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", err
		}
		rows, proj, err := flatten(body, run.Label)
		if err != nil {
			return "", err
		}
		if len(all) == 0 && len(rows) > 0 {
			project = proj
		}
		all = append(all, rows...)
		source, err := os.ReadFile(filepath.Join(filepath.Dir(run.Path), "sources.md"))
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if len(source) > 0 {
			fmt.Fprintf(&sources, "# %s\n\n%s\n", run.Label, source)
		}
	}
	if len(all) == 0 {
		return "", fmt.Errorf("no results")
	}
	return fillPageData(project, string(joinArray(all)), "jmark-data", Options{Sources: sources.String()}), nil
}

// BrowserOpen opens path with /usr/bin/open, then open, then xdg-open.
func BrowserOpen(path string) error {
	bin, err := findOpener()
	if err != nil {
		return fmt.Errorf("no browser opener (open/xdg-open); file is %s", path)
	}
	return exec.Command(bin, path).Run()
}

func findOpener() (string, error) {
	if st, err := os.Stat("/usr/bin/open"); err == nil && st.Mode()&0o111 != 0 {
		return "/usr/bin/open", nil
	}
	if p, err := exec.LookPath("open"); err == nil {
		return p, nil
	}
	if p, err := exec.LookPath("xdg-open"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("no browser opener")
}

func flatten(ndjson []byte, run string) ([][]byte, string, error) {
	project := "jmark"
	dec := json.NewDecoder(bytes.NewReader(ndjson))
	var rows [][]byte
	for {
		var raw json.RawMessage
		err := dec.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
		doc, err := resultDocument(raw)
		if err != nil {
			return nil, "", err
		}
		part, err := flattenDoc(doc, run)
		if err != nil {
			return nil, "", err
		}
		if len(rows) == 0 && len(part) > 0 {
			project = projectTitle(doc["project"])
		}
		rows = append(rows, part...)
	}
	return rows, project, nil
}

func flattenDoc(doc map[string]json.RawMessage, run string) ([][]byte, error) {
	results, err := rawArray(doc["results"])
	if err != nil {
		return nil, err
	}
	project, err := fieldRaw(doc, "project")
	if err != nil {
		return nil, err
	}
	runtime, err := fieldRaw(doc, "runtime")
	if err != nil {
		return nil, err
	}
	metric, err := fieldRaw(doc, "metric")
	if err != nil {
		return nil, err
	}
	meta, err := metaFields(doc)
	if err != nil {
		return nil, err
	}
	n, err := fieldRaw(meta, "n")
	if err != nil {
		return nil, err
	}
	warmup, err := fieldRaw(meta, "warmup")
	if err != nil {
		return nil, err
	}
	itersMeta, err := fieldRaw(meta, "iters")
	if err != nil {
		return nil, err
	}
	runs, err := fieldRaw(meta, "runs")
	if err != nil {
		return nil, err
	}
	captured, err := fieldRaw(meta, "captured_at")
	if err != nil {
		return nil, err
	}
	runtimeVersion, err := fieldRaw(meta, "runtime")
	if err != nil {
		return nil, err
	}
	host, err := fieldRaw(meta, "host")
	if err != nil {
		return nil, err
	}
	var out [][]byte
	for _, el := range results {
		item, err := rawObject(el)
		if err != nil {
			return nil, err
		}
		name, err := fieldRaw(item, "name")
		if err != nil {
			return nil, err
		}
		value, err := fieldRaw(item, "value")
		if err != nil {
			return nil, err
		}
		iters, err := fieldRaw(item, "iters")
		if err != nil {
			return nil, err
		}
		total, err := fieldRaw(item, "total_ms")
		if err != nil {
			return nil, err
		}
		sink, err := fieldRaw(item, "sink")
		if err != nil {
			return nil, err
		}
		keys := []string{
			"project", "runtime", "metric", "name", "value", "iters", "total_ms", "sink",
			"n", "warmup", "iters_meta", "runs", "captured_at", "runtime_version", "host",
		}
		vals := []json.RawMessage{
			project, runtime, metric, name, value, iters, total, sink,
			n, warmup, itersMeta, runs, captured, runtimeVersion, host,
		}
		if run != "" {
			keys = append([]string{"run"}, keys...)
			vals = append([]json.RawMessage{jsonString(run)}, vals...)
		}
		out = append(out, encodeObject(keys, vals))
	}
	return out, nil
}

func projectTitle(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("false")) {
		return "jmark"
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "jmark"
	}
	return s
}

func joinArray(rows [][]byte) []byte {
	var b bytes.Buffer
	b.WriteByte('[')
	for i, row := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.Write(row)
	}
	b.WriteByte(']')
	return b.Bytes()
}

func encodeObject(keys []string, vals []json.RawMessage) []byte {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.Quote(key))
		b.WriteByte(':')
		b.Write(vals[i])
	}
	b.WriteByte('}')
	return b.Bytes()
}

func jsonString(s string) json.RawMessage {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return bytes.TrimRight(buf.Bytes(), "\n")
}

// resultDocument accepts the previous top-level "lang" key when opening saved runs.
// New reports use "runtime"; meta.runtime remains the version string.
func resultDocument(raw json.RawMessage) (map[string]json.RawMessage, error) {
	doc, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	if _, exists := doc["runtime"]; !exists {
		doc["runtime"] = doc["lang"]
	}
	delete(doc, "lang")
	return doc, nil
}

func rawObject(raw json.RawMessage) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, ok := tok.(json.Delim)
	if !ok || d != '{' {
		return nil, fmt.Errorf("expected object")
	}
	out := map[string]json.RawMessage{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := kt.(string)
		if !ok {
			return nil, fmt.Errorf("expected string key")
		}
		var val json.RawMessage
		if err := dec.Decode(&val); err != nil {
			return nil, err
		}
		out[key] = val
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return out, nil
}

func rawArray(raw json.RawMessage) ([]json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, fmt.Errorf("missing results")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	d, ok := tok.(json.Delim)
	if !ok || d != '[' {
		return nil, fmt.Errorf("results must be an array")
	}
	var out []json.RawMessage
	for dec.More() {
		var el json.RawMessage
		if err := dec.Decode(&el); err != nil {
			return nil, err
		}
		out = append(out, el)
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return out, nil
}

func metaFields(doc map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	raw := bytes.TrimSpace(doc["meta"])
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return map[string]json.RawMessage{}, nil
	}
	meta, err := rawObject(raw)
	if err != nil {
		return nil, err
	}
	if host := bytes.TrimSpace(meta["host"]); len(host) > 0 && host[0] == '{' {
		fields, err := rawObject(host)
		if err != nil {
			return nil, err
		}
		if _, found := fields["hostname"]; found {
			delete(fields, "hostname")
			meta["host"], err = json.Marshal(fields)
			if err != nil {
				return nil, err
			}
		}
	}
	return meta, nil
}

func fieldRaw(m map[string]json.RawMessage, key string) (json.RawMessage, error) {
	raw := bytes.TrimSpace(m[key])
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return json.RawMessage("null"), nil
	}
	return compactRaw(raw)
}

func compactRaw(raw json.RawMessage) (json.RawMessage, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Environment summarizes host and runtime metadata from saved results.
func Environment(ndjson []byte) ([]byte, error) {
	docs, err := parseDocs(ndjson)
	if err != nil {
		return nil, err
	}
	captured := json.RawMessage("null")
	host := json.RawMessage("null")
	var runtimes []namedRaw
	index := map[string]int{}
	for _, doc := range docs {
		meta, err := metaFields(doc)
		if err != nil {
			return nil, err
		}
		if bytes.Equal(captured, []byte("null")) {
			if s, ok := jsonStringValue(meta["captured_at"]); ok && s != "" {
				captured, err = compactRaw(meta["captured_at"])
				if err != nil {
					return nil, err
				}
			}
		}
		if bytes.Equal(host, []byte("null")) {
			if h, ok, err := usefulHost(meta["host"]); err != nil {
				return nil, err
			} else if ok {
				host = h
			}
		}
		runtime, ok := jsonStringValue(doc["runtime"])
		if !ok {
			continue
		}
		rt, err := runtimeValue(meta["runtime"])
		if err != nil {
			return nil, err
		}
		if i, seen := index[runtime]; seen {
			runtimes[i].raw = rt
			continue
		}
		index[runtime] = len(runtimes)
		runtimes = append(runtimes, namedRaw{runtime: runtime, raw: rt})
	}
	rtRaw, err := runtimesObject(runtimes)
	if err != nil {
		return nil, err
	}
	var b bytes.Buffer
	b.WriteString("{\n  \"captured_at\": ")
	b.Write(captured)
	b.WriteString(",\n  \"host\": ")
	if err := writePretty(&b, host, "  "); err != nil {
		return nil, err
	}
	b.WriteString(",\n  \"runtimes\": ")
	if err := writePretty(&b, rtRaw, "  "); err != nil {
		return nil, err
	}
	b.WriteString("\n}\n")
	return b.Bytes(), nil
}

type namedRaw struct {
	runtime string
	raw     json.RawMessage
}

func parseDocs(ndjson []byte) ([]map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(ndjson))
	var docs []map[string]json.RawMessage
	for {
		var raw json.RawMessage
		err := dec.Decode(&raw)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		doc, err := resultDocument(raw)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func jsonStringValue(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

func usefulHost(raw json.RawMessage) (json.RawMessage, bool, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("false")) {
		return nil, false, nil
	}
	c, err := compactRaw(raw)
	if err != nil {
		return nil, false, err
	}
	if bytes.Equal(c, []byte("{}")) {
		return nil, false, nil
	}
	return c, true, nil
}

func runtimeValue(raw json.RawMessage) (json.RawMessage, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || bytes.Equal(raw, []byte("false")) {
		return json.RawMessage("null"), nil
	}
	return compactRaw(raw)
}

func runtimesObject(rows []namedRaw) (json.RawMessage, error) {
	var b bytes.Buffer
	b.WriteByte('{')
	for i, row := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		key, err := marshalString(row.runtime)
		if err != nil {
			return nil, err
		}
		b.Write(key)
		b.WriteByte(':')
		b.Write(row.raw)
	}
	b.WriteByte('}')
	return b.Bytes(), nil
}

func marshalString(s string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func writePretty(b *bytes.Buffer, raw []byte, prefix string) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		b.WriteString("null")
		return nil
	}
	scalar := raw[0] != '{' && raw[0] != '['
	empty := bytes.Equal(raw, []byte("{}")) || bytes.Equal(raw, []byte("[]"))
	if scalar || empty {
		b.Write(raw)
		return nil
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, prefix, "  "); err != nil {
		return err
	}
	b.Write(buf.Bytes())
	return nil
}
