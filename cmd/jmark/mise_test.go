package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeMiseDelegation(t *testing.T) {
	root := t.TempDir()
	writeRepo(t, root, map[string]string{
		"runtimes/node.toml": "family = \"js\"\nneeds = \"absent-node\"\n",
		"runtimes/py.toml":   "family = \"py\"\nneeds = \"absent-python\"\n",
		"scripts/mise":       "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\"\n",
	})
	path := filepath.Join(root, "scripts", "mise")
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"runtimes", "lock", "node", "py"}, "lock\nnode\npython\n"},
		{[]string{"runtimes", "install"}, "install\n--locked\n"},
	} {
		out, stderr, code := runOut(t, root, tc.args...)
		if code != 0 || out != root+"\n"+tc.want {
			t.Fatalf("%v: %d %q %q", tc.args, code, out, stderr)
		}
	}
	for _, args := range [][]string{{"runtimes", "check"}, {"runtimes", "lock", "unknown"}} {
		if _, _, code := runOut(t, root, args...); code == 0 {
			t.Fatalf("accepted %v", args)
		}
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho download-failed >&2\nexit 3\n"), 0755); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runOut(t, root, "runtimes", "install")
	if code == 0 || !strings.Contains(stderr, "download-failed") {
		t.Fatalf("%d %s", code, stderr)
	}
}

func TestManagedBenchCleanup(t *testing.T) {
	for _, status := range []string{"0", "7"} {
		t.Run(status, func(t *testing.T) {
			root := t.TempDir()
			writeRepo(t, root, map[string]string{"scripts/mise": "#!/bin/sh\nprintf '%s\\n' \"$JMARK_MISE_DATA_DIR\" \"$JMARK_MISE_ACTIVE\" \"$@\" > invocation\nexit " + status + "\n"})
			if err := os.Chmod(filepath.Join(root, "scripts", "mise"), 0755); err != nil {
				t.Fatal(err)
			}
			code := managedBench([]string{"bench", "demo", "--runtime", "node"}, root)
			if (code == 0) != (status == "0") {
				t.Fatalf("exit %d", code)
			}
			data, err := os.ReadFile(filepath.Join(root, "invocation"))
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(data)), "\n")
			if len(lines) != 10 || strings.Join(lines[1:5], " ") != "1 exec --locked --" || strings.Join(lines[6:], " ") != "bench demo --runtime node" {
				t.Fatalf("invocation %q", data)
			}
			if _, err := os.Stat(lines[0]); !os.IsNotExist(err) {
				t.Fatalf("temporary install remains: %v", err)
			}
		})
	}
}
