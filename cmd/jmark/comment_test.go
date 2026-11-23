package main

import (
	"os"
	"strings"
	"testing"
)

func TestCommentCommandAndRows(t *testing.T) {
	root := t.TempDir()
	commented := makeRun(t, root, "demo", "20260102-000000")
	bare := makeRun(t, root, "demo", "20260101-000000")
	other := makeRun(t, root, "beta", "20260103-000000")
	writeRepo(t, root, map[string]string{"benches/beta/AGENTS.md": "Beta project summary.\n"})

	out, errOut, code := runOut(t, root, "comment", "demo/20260102-000000", "json", "vs", "msgpack")
	if code != 0 || errOut != "" || out != "artifacts/"+commented+"\n" {
		t.Fatalf("set: %d %q %q", code, out, errOut)
	}
	out, _, code = runOut(t, root, "comment", commented)
	if code != 0 || out != "json vs msgpack\n" {
		t.Fatalf("get: %d %q", code, out)
	}

	var listed string
	var fzfArgs []string
	a := &app{root: root, fzf: func(input string, args []string) (string, []string, error) {
		listed, fzfArgs = input, args
		return openCommentKey, []string{bare + "\t", commented + "\tjson vs msgpack"}, nil
	}}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	w.Write([]byte("batch size sweep\n"))
	w.Close()
	a.in = r
	_, errOut, code = finishApp(t, a, "open")
	if code != 0 || !strings.Contains(errOut, "commented "+bare) {
		t.Fatalf("open ctrl-e: %d %q", code, errOut)
	}
	want := commented + "\tjson vs msgpack\n" + bare + "\n" + other + "\t\x1b[2mBeta project summary.\x1b[0m\n"
	if listed != want {
		t.Fatalf("rows\n got %q\nwant %q", listed, want)
	}
	if !strings.Contains(strings.Join(fzfArgs, " "), "--delimiter=\t") {
		t.Fatalf("args %v", fzfArgs)
	}
	for _, key := range []string{bare, commented} {
		out, _, _ = runOut(t, root, "comment", key)
		if out != "batch size sweep\n" {
			t.Fatalf("%s: %q", key, out)
		}
	}

	out, _, code = runOut(t, root, "comment", bare, "--clear")
	if code != 0 {
		t.Fatal("clear failed")
	}
	out, _, _ = runOut(t, root, "comment", bare)
	if out != "" {
		t.Fatalf("cleared %q", out)
	}
	_, errOut, code = runOut(t, root, "comment", "demo/missing", "x")
	if code != 1 || !strings.Contains(errOut, "missing or unsafe") {
		t.Fatalf("missing: %d %q", code, errOut)
	}
}

func TestBenchCommentFlag(t *testing.T) {
	o, err := parseBench([]string{"demo", "--comment", "a b", "--format", "html"})
	if err != nil || o.comment != "a b" || !o.commentSet {
		t.Fatalf("%+v %v", o, err)
	}
	if o, err = parseBench([]string{"--comment=x", "demo"}); err != nil || o.comment != "x" {
		t.Fatalf("%+v %v", o, err)
	}
	if _, err = parseBench([]string{"demo", "--comment"}); err == nil {
		t.Fatal("missing value accepted")
	}
	_, errOut, code := runOut(t, t.TempDir(), "bench", "demo", "--format", "stream", "--comment", "x")
	if code != 1 || !strings.Contains(errOut, "--comment needs --format") {
		t.Fatalf("stream: %d %q", code, errOut)
	}
}
