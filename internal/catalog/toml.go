package catalog

import (
	"os"
	"regexp"
	"strings"
)

// Flat TOML only. A general TOML library would accept tables and multi-line
// values outside the registry format.

var (
	reSkip   = regexp.MustCompile(`^[ \t]*(#.*)?$`)
	reAssign = regexp.MustCompile(`^[ \t]*([a-z_]+)[ \t]*=[ \t]*(.*)$`)
	reString = regexp.MustCompile(`^"([^"]*)"[ \t]*(#.*)?$`)
	reInt    = regexp.MustCompile(`^([0-9]+)[ \t]*(#.*)?$`)
	reCanon  = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)
)

type flatFile struct {
	path string
	keys []string
	val  map[string]string
}

func (f flatFile) get(key string) string {
	return f.val[key]
}

func (f flatFile) unknown(allowed []string) []string {
	ok := membership(allowed)
	var bad []string
	for _, k := range f.keys {
		if !ok[k] {
			bad = append(bad, k)
		}
	}
	return bad
}

func parseFlat(path string) (flatFile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return flatFile{}, jerr("%s", err)
	}
	f := flatFile{path: path, val: map[string]string{}}
	for i, line := range splitLines(string(b)) {
		if reSkip.MatchString(line) {
			continue
		}
		m := reAssign.FindStringSubmatch(line)
		if m == nil {
			return flatFile{}, jerr("%s:%d: expected flat `key = value` (no tables, no multi-line)", path, i+1)
		}
		key, raw := m[1], m[2]
		var val string
		if sm := reString.FindStringSubmatch(raw); sm != nil {
			val = sm[1]
		} else if im := reInt.FindStringSubmatch(raw); im != nil {
			val = im[1]
		} else {
			return flatFile{}, jerr("%s:%d: value of %s must be a \"string\" or an integer", path, i+1, key)
		}
		if _, dup := f.val[key]; dup {
			return flatFile{}, jerr("%s:%d: duplicate key %s", path, i+1, key)
		}
		f.keys = append(f.keys, key)
		f.val[key] = val
	}
	return f, nil
}

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	if strings.HasSuffix(text, "\n") {
		text = text[:len(text)-1]
	}
	return strings.Split(text, "\n")
}
