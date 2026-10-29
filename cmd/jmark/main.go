// Command jmark is the host CLI.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/JeHwanYoo/jmark/internal/catalog"
)

const usage = "usage: doctor | list | runtimes [lock|install [<runtime>...]] | create <id> <runtime>... | bench [<id>] [--runtime <runtime>] [--format stream|ndjson|html] | open [<id>] | delete [<id>] | public [<project>/<run>] | private [<project>/<run>] | site [--base-url <url>]"

type app struct {
	root   string
	cat    *catalog.Catalog
	in     *os.File
	stdout io.Writer
	stderr io.Writer
	// fzf, browse, and tty are nil in production. Tests set them so picker flows
	// can run without a terminal.
	fzf    func(input string, args []string) (string, []string, error)
	browse func(string) error
	tty    *bool
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "bench" && os.Getenv("JMARK_MISE_ACTIVE") != "1" && !benchHelp(os.Args[2:]) {
		root, err := findRoot()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(managedBench(os.Args[1:], root))
	}
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, ""))
}

func run(args []string, in *os.File, stdout, stderr io.Writer, root string) int {
	if root == "" {
		found, err := findRoot()
		if err != nil {
			fmt.Fprintf(stderr, "jmark: %s\n", err)
			return 1
		}
		root = found
	}
	if strings.IndexFunc(root, unicode.IsSpace) >= 0 {
		fmt.Fprintf(stderr, "jmark: repo path must not contain whitespace: %s\n", root)
		return 1
	}
	a := &app{
		root:   root,
		cat:    catalog.New(root),
		in:     in,
		stdout: stdout,
		stderr: stderr,
	}
	return finish(a, args)
}

func finish(a *app, args []string) int {
	if err := a.dispatch(args); err != nil {
		if errors.Is(err, errSoftQuit) {
			fmt.Fprintln(a.stderr, "jmark: quit")
			return 0
		}
		msg := err.Error()
		if !strings.HasPrefix(msg, "jmark: ") {
			msg = "jmark: " + msg
		}
		fmt.Fprintln(a.stderr, msg)
		return 1
	}
	return 0
}

func (a *app) dispatch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("%s", usage)
	}
	switch args[0] {
	case "doctor":
		return a.cmdDoctor()
	case "list":
		return a.cmdList()
	case "runtimes":
		return a.cmdRuntimes(args[1:])
	case "create":
		return a.cmdCreate(args[1:])
	case "bench":
		return a.cmdBench(args[1:])
	case "open":
		return a.cmdOpen(args[1:])
	case "public", "private":
		return a.cmdVisibility(args[0], args[1:])
	case "site":
		return a.cmdSite(args[1:])
	case "delete":
		return a.cmdDelete(args[1:])
	default:
		return fmt.Errorf("%s", usage)
	}
}

func (a *app) cmdList() error {
	ids, err := a.cat.ListProjects()
	if err != nil {
		return err
	}
	for _, id := range ids {
		fmt.Fprintln(a.stdout, id)
	}
	return nil
}

func (a *app) cmdRuntimes(args []string) error {
	if len(args) > 0 {
		return a.cmdRuntimeMise(args)
	}
	ids, err := a.cat.ListRuntimes()
	if err != nil {
		return err
	}
	for _, id := range ids {
		runtime, err := a.cat.Runtime(id)
		if err != nil {
			return err
		}
		missing, err := a.cat.RuntimeMissing(id)
		if err != nil {
			return err
		}
		if len(missing) == 0 {
			fmt.Fprintf(a.stdout, "%s\t%s\tfound\n", id, runtime.Family)
			continue
		}
		fmt.Fprintf(a.stdout, "%s\t%s\tmissing: %s\n", id, runtime.Family, strings.Join(missing, " "))
	}
	return nil
}

func (a *app) cmdCreate(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: create <id> <runtime>...")
	}
	id := args[0]
	if !catalog.ValidID(id) {
		return fmt.Errorf("invalid project id '%s' (must match ^[a-z0-9][a-z0-9-]*$)", id)
	}
	var fams []string
	seen := map[string]bool{}
	for _, runtime := range args[1:] {
		if !a.cat.RuntimeExists(runtime) {
			return a.unknownRuntime(runtime)
		}
		rec, err := a.cat.Runtime(runtime)
		if err != nil {
			return err
		}
		if !seen[rec.Family] {
			seen[rec.Family] = true
			fams = append(fams, rec.Family)
		}
	}
	if err := a.cat.Scaffold(id, fams...); err != nil {
		return err
	}
	fmt.Fprintf(a.stdout, "created benches/%s (families: %s)\n", id, strings.Join(fams, " "))
	return nil
}

func (a *app) unknownRuntime(runtime string) error {
	ids, err := a.cat.ListRuntimes()
	if err != nil {
		return err
	}
	return fmt.Errorf("unknown runtime '%s' (have: %s)", runtime, bashWords(ids))
}

func (a *app) warn(format string, args ...any) {
	fmt.Fprintf(a.stderr, "jmark: "+format+"\n", args...)
}

func (a *app) writeErr(err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	if !strings.HasPrefix(msg, "jmark: ") {
		msg = "jmark: " + msg
	}
	fmt.Fprintln(a.stderr, msg)
}

func (a *app) stdinTTY() bool {
	if a.tty != nil {
		return *a.tty
	}
	if a.in == nil {
		return false
	}
	fi, err := a.in.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// bashWords matches tr '\n' ' ' on a newline-terminated id list, including the trailing space.
func bashWords(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return strings.Join(ids, " ") + " "
}

// findRoot locates the repo that holds justfile, runtimes/, and families/.
// go run and go test place the binary under a go-build cache, not the module.
// Those invocations resolve the root from the working directory.
// A binary built inside the repo walks up from the executable.
func findRoot() (string, error) {
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		if !inGoBuildCache(exe) {
			if root, ok := walkUp(filepath.Dir(exe)); ok {
				return root, nil
			}
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if root, ok := walkUp(cwd); ok {
		return root, nil
	}
	return "", fmt.Errorf("could not find repo root (need justfile, runtimes/, and families/)")
}

func inGoBuildCache(path string) bool {
	// Cache directories are named go-build plus a random suffix, not a clean path segment.
	return strings.Contains(path, string(filepath.Separator)+"go-build")
}

func walkUp(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		if isRepoRoot(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func isRepoRoot(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, "justfile"))
	if err != nil || !st.Mode().IsRegular() {
		return false
	}
	for _, name := range []string{"runtimes", "families"} {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || !st.IsDir() {
			return false
		}
	}
	return true
}

func benchHelp(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}
