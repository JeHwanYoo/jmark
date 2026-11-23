package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/artifact"
	"github.com/JeHwanYoo/jmark/internal/catalog"
	"github.com/JeHwanYoo/jmark/internal/report"
)

// errSoftQuit is Esc, Ctrl-C, or q in fzf.
// The process exits 0 so a cancelled picker does not look like a failed command.
var errSoftQuit = errors.New("quit")

var errFzfMissing = errors.New("fzf missing")

const openUsage = `usage: open [<id>]

fzf multi-select artifacts/<private|public>/<id>/<timestamp> runs (Tab to toggle).
Each row shows the run's comment, or the project's AGENTS.md summary when it has none.
Enter → chart compare. Ctrl-E → comment selected runs. Ctrl-X → delete selected runs. q → quit.
`

const deleteUsage = `usage: delete [<id>]

fzf multi-select artifacts/<private|public>/<id>/<timestamp> runs (Tab to toggle, Enter to delete).
q → quit. Confirms with y, then removes each selected run directory.
`

// artifactAction is what the shared run picker should do with the selection.
type artifactAction int

const (
	artifactQuit artifactAction = iota
	artifactCompare
	artifactDelete
	artifactComment
)

// open uses Ctrl-X for delete. fzf's default Ctrl-D is delete-char/eof and
// aborts when the query is empty, so multi-select never reaches deleteConfirmed.
const openDeleteKey = "ctrl-x"

// openCommentKey edits the summary of the selected runs from the open picker.
const openCommentKey = "ctrl-e"

func (a *app) cmdOpen(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(a.stdout, openUsage)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: open [<id>]")
	}
	filter := ""
	if len(args) == 1 {
		filter = args[0]
	}
	action, root, dirs, err := a.pickArtifactRuns(filter, true)
	if err != nil {
		return a.fzfErr(err, "open needs fzf (https://github.com/junegunn/fzf)")
	}
	switch action {
	case artifactQuit:
		return errSoftQuit
	case artifactDelete:
		return a.deleteConfirmed(root, dirs)
	case artifactComment:
		return a.commentRuns(root, dirs)
	default:
		return a.openCompare(root, dirs)
	}
}

func (a *app) cmdDelete(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(a.stdout, deleteUsage)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("usage: delete [<id>]")
	}
	filter := ""
	if len(args) == 1 {
		filter = args[0]
	}
	action, root, dirs, err := a.pickArtifactRuns(filter, false)
	if err != nil {
		return a.fzfErr(err, "delete needs fzf (https://github.com/junegunn/fzf)")
	}
	if action == artifactQuit {
		return errSoftQuit
	}
	return a.deleteConfirmed(root, dirs)
}

// pickArtifactRuns is the one multi-select fzf over artifacts/.
// withCompare enables Enter=compare and Ctrl-X=delete (open).
// Otherwise Enter deletes (delete command).
func (a *app) pickArtifactRuns(filter string, withCompare bool, verb ...string) (artifactAction, string, []string, error) {
	root, runs, err := a.artifactRuns(filter)
	if err != nil {
		return artifactQuit, "", nil, err
	}
	if err := a.requireFzf(); err != nil {
		return artifactQuit, "", nil, errFzfMissing
	}
	var fzfArgs []string
	if withCompare {
		fzfArgs = []string{
			"-m", "--expect=" + openDeleteKey + "," + openCommentKey + ",q",
			"--prompt=jmark open (tab=multi)> ",
			"--height=40%", "--reverse", "--border",
			"--header=Tab: toggle · Enter: compare · Ctrl-E: comment · Ctrl-X: delete · q: quit",
		}
	} else if len(verb) == 1 && verb[0] == "comment" {
		fzfArgs = []string{
			"-m", "--expect=q",
			"--prompt=jmark comment (tab=multi)> ",
			"--height=40%", "--reverse", "--border",
			"--header=Tab: toggle · Enter: comment · q: quit",
		}
	} else {
		fzfArgs = []string{
			"-m", "--expect=q",
			"--prompt=jmark delete (tab=multi)> ",
			"--height=40%", "--reverse", "--border",
			"--header=Tab: toggle · Enter: delete (asks y) · q: quit",
		}
	}
	key, picks, err := a.runFzf(a.runRows(runs), append(fzfArgs, runRowArgs...)...)
	if err != nil {
		return artifactQuit, "", nil, err
	}
	if key == "q" || len(picks) == 0 {
		return artifactQuit, root, nil, nil
	}
	dirs, err := resolvePicks(root, picks)
	if err != nil {
		return artifactQuit, "", nil, err
	}
	if len(dirs) == 0 {
		cancel := "open cancelled"
		if !withCompare {
			cancel = "delete cancelled"
		}
		return artifactQuit, "", nil, fmt.Errorf("%s", cancel)
	}
	if withCompare && key == openDeleteKey {
		return artifactDelete, root, dirs, nil
	}
	if withCompare && key == openCommentKey {
		return artifactComment, root, dirs, nil
	}
	if withCompare {
		return artifactCompare, root, dirs, nil
	}
	if len(verb) == 1 && verb[0] == "comment" {
		return artifactComment, root, dirs, nil
	}
	return artifactDelete, root, dirs, nil
}

func (a *app) benchPickID() (string, error) {
	if err := a.requireFzf(); err != nil {
		return "", fmt.Errorf("bench needs fzf to pick a project (or pass: just bench <id>)")
	}
	ids, err := a.cat.ListProjects()
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no projects yet (run: just create <id> <runtime>...)")
	}
	preview := fmt.Sprintf("[ -f %s/benches/{}/AGENTS.md ] && cat %s/benches/{}/AGENTS.md", a.root, a.root)
	key, lines, err := a.runFzf(linesInput(ids),
		"--expect=q",
		"--prompt=jmark bench> ",
		"--height=40%", "--reverse", "--border",
		"--header=Enter: choose options · q: quit",
		"--preview="+preview,
		"--preview-window=down,3,wrap",
	)
	if err != nil {
		return "", a.fzfErr(err, "bench needs fzf to pick a project (or pass: just bench <id>)")
	}
	if key == "q" || len(lines) != 1 {
		return "", errSoftQuit
	}
	return lines[0], nil
}

// Only options omitted from the command line are prompted for.
func (a *app) benchPickOptions(opts benchOpts) (benchOpts, error) {
	if !opts.runtimeSet {
		runtimes, err := a.benchRuntimes(opts)
		if err != nil {
			return opts, err
		}
		choices := []string{"all\tEvery available runtime (default)"}
		for _, runtime := range runtimes {
			choices = append(choices, runtime+"\t--runtime "+runtime)
		}
		runtime, err := a.benchPickOption("runtime", opts.id, choices, "Enter: choose runtime · q: quit")
		if err != nil {
			return opts, err
		}
		if runtime != "all" {
			opts.runtime, opts.runtimeSet = runtime, true
		}
	} else if _, err := a.benchRuntimes(opts); err != nil {
		return opts, err
	}
	if !opts.formatSet {
		format, err := a.benchPickOption("format", opts.id, []string{
			"stream\tResult NDJSON on stdout (default)",
			"ndjson\tSave result.ndjson; print path",
			"html\tSave result.ndjson + report.html; print HTML path",
		}, "Enter: run · q: quit")
		if err != nil {
			return opts, err
		}
		opts.format = format
	}
	return opts, nil
}

func (a *app) benchPickOption(option, id string, choices []string, header string) (string, error) {
	key, picks, err := a.runFzf(linesInput(choices),
		"--expect=q", "--prompt=jmark bench "+option+"> ",
		"--height=40%", "--reverse", "--border", "--delimiter=\t",
		"--header="+id+" · "+header,
	)
	if err != nil {
		return "", err
	}
	if key == "q" || len(picks) != 1 {
		return "", errSoftQuit
	}
	for _, choice := range choices {
		if picks[0] == choice {
			return strings.SplitN(choice, "\t", 2)[0], nil
		}
	}
	return "", fmt.Errorf("bench: invalid %s selection", option)
}

func (a *app) fzfErr(err error, missing string) error {
	if errors.Is(err, errFzfMissing) {
		return fmt.Errorf("%s", missing)
	}
	return err
}

func (a *app) requireFzf() error {
	if a.fzf != nil {
		return nil
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		return errFzfMissing
	}
	return nil
}

func (a *app) runFzf(input string, args ...string) (string, []string, error) {
	if a.fzf != nil {
		return a.fzf(input, args)
	}
	return execFzf(input, args)
}

func (a *app) browserOpen(path string) error {
	if a.browse != nil {
		return a.browse(path)
	}
	return report.BrowserOpen(path)
}

func execFzf(input string, args []string) (string, []string, error) {
	path, err := exec.LookPath("fzf")
	if err != nil {
		return "", nil, errFzfMissing
	}
	cmd := exec.Command(path, args...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Stderr = os.Stderr
	var out bytes.Buffer
	cmd.Stdout = &out
	runErr := cmd.Run()
	if err := fzfStatus(runErr); err != nil {
		return "", nil, err
	}
	key, lines := splitExpect(out.String())
	return key, lines, nil
}

// fzf uses exit 1 for Esc and 130 for Ctrl-C. Both cancel the picker.
func fzfStatus(err error) error {
	if err == nil {
		return nil
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		switch ee.ExitCode() {
		case 1, 130:
			return errSoftQuit
		default:
			return fmt.Errorf("fzf failed (exit %d)", ee.ExitCode())
		}
	}
	return err
}

func splitExpect(out string) (string, []string) {
	out = strings.TrimSuffix(out, "\n")
	if out == "" {
		return "", nil
	}
	parts := strings.Split(out, "\n")
	var lines []string
	for _, p := range parts[1:] {
		if p != "" {
			lines = append(lines, p)
		}
	}
	return parts[0], lines
}

func linesInput(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

func (a *app) artifactRuns(filter string) (string, []string, error) {
	root := filepath.Join(a.root, "artifacts")
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return "", nil, fmt.Errorf("no artifacts/ yet (run: just bench <id> --format ndjson|html)")
	}
	if filter != "" {
		if !catalog.ValidID(filter) {
			return "", nil, fmt.Errorf("invalid project id '%s'", filter)
		}

	}
	runs, err := listRuns(root, filter)
	if err != nil {
		return "", nil, err
	}
	if filter != "" && len(runs) == 0 {
		return "", nil, fmt.Errorf("no artifacts for %s", filter)
	}
	return root, runs, nil
}

func listRuns(root, filter string) ([]string, error) {
	return artifact.List(filepath.Dir(root), "", filter)
}

func resolvePicks(root string, picks []string) ([]string, error) {
	var dirs []string
	for _, pick := range picks {
		pick = runKey(pick)
		if pick == "" {
			continue
		}
		dir, err := artifact.Resolve(filepath.Dir(root), pick)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, dir)
	}
	return dirs, nil
}

func (a *app) deleteConfirmed(root string, dirs []string) error {
	rels := make([]string, len(dirs))
	fmt.Fprintf(a.stderr, "jmark: delete %d run(s):\n", len(dirs))
	for i, dir := range dirs {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}
		rels[i] = filepath.ToSlash(rel)
		fmt.Fprintf(a.stderr, "  %s\n", rels[i])
	}
	fmt.Fprint(a.stderr, "jmark: type y to confirm: ")
	ans, _ := readLine(a.in)
	if ans != "y" {
		return fmt.Errorf("delete cancelled")
	}
	for i, dir := range dirs {
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		_ = os.Remove(filepath.Dir(dir))
		fmt.Fprintf(a.stderr, "jmark: deleted %s\n", rels[i])
	}
	return nil
}

func readLine(in io.Reader) (string, error) {
	if in == nil {
		return "", io.EOF
	}
	s, err := bufio.NewReader(in).ReadString('\n')
	s = strings.TrimRight(s, "\r\n")
	if s != "" {
		return s, nil
	}
	if err != nil {
		return "", err
	}
	return "", nil
}

func (a *app) openCompare(root string, dirs []string) error {
	runs := make([]report.CompareRun, 0, len(dirs))
	for _, dir := range dirs {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return fmt.Errorf("could not render compare view")
		}
		runs = append(runs, report.CompareRun{
			Label: filepath.ToSlash(rel),
			Path:  filepath.Join(dir, "result.ndjson"),
		})
	}
	html, err := report.RenderCompare(runs)
	if err != nil {
		return fmt.Errorf("could not render compare view")
	}
	f, err := os.CreateTemp("", "jmark-compare-*.html")
	if err != nil {
		return fmt.Errorf("could not render compare view")
	}
	name := f.Name()
	if _, err := io.WriteString(f, html); err != nil {
		f.Close()
		return fmt.Errorf("could not render compare view")
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("could not render compare view")
	}
	if err := a.browserOpen(name); err != nil {
		if strings.HasPrefix(err.Error(), "no browser opener") {
			a.writeErr(err)
		}
		return fmt.Errorf("could not open compare view")
	}
	fmt.Fprintf(a.stderr, "jmark: opened compare of %d run(s)\n", len(dirs))
	return nil
}

// runRowArgs show the summary column and keep it out of the selection key.
var runRowArgs = []string{"--delimiter=\t", "--ansi", "--preview=printf '%s\\n' {2..}", "--preview-window=down,3,wrap"}

// runRows renders one "key<TAB>summary" row per run. A run's own comment wins;
// otherwise the project's AGENTS.md line is shown dimmed. Runs with neither stay bare keys.
func (a *app) runRows(runs []string) string {
	rows := make([]string, len(runs))
	projects := map[string]string{}
	for i, key := range runs {
		rows[i] = key
		parts := strings.Split(key, "/")
		if len(parts) != 3 {
			continue
		}
		if comment := artifact.Comment(filepath.Join(a.root, "artifacts", parts[0], parts[1], parts[2])); comment != "" {
			rows[i] = key + "\t" + comment
			continue
		}
		summary, ok := projects[parts[1]]
		if !ok {
			summary = artifact.ProjectSummary(a.root, parts[1])
			projects[parts[1]] = summary
		}
		if summary != "" {
			rows[i] = key + "\t\x1b[2m" + summary + "\x1b[0m"
		}
	}
	return linesInput(rows)
}

// runKey drops the summary column from a picked row.
func runKey(row string) string {
	key, _, _ := strings.Cut(row, "\t")
	return strings.TrimSpace(key)
}

// promptComment asks for a one-line summary on stderr; Enter keeps current.
func (a *app) promptComment(label, current string) string {
	if label != "" {
		fmt.Fprintf(a.stderr, "jmark: %s\n", label)
	}
	if current != "" {
		fmt.Fprintf(a.stderr, "jmark: current: %s\n", current)
		fmt.Fprint(a.stderr, "jmark: comment (Enter = keep, - = clear): ")
	} else {
		fmt.Fprint(a.stderr, "jmark: comment (Enter = skip): ")
	}
	ans, _ := readLine(a.in)
	switch ans = strings.TrimSpace(ans); ans {
	case "":
		return current
	case "-":
		return ""
	default:
		return ans
	}
}

func (a *app) commentRuns(root string, dirs []string) error {
	keys := make([]string, len(dirs))
	current := ""
	for i, dir := range dirs {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return err
		}
		keys[i] = filepath.ToSlash(rel)
		if i == 0 {
			current = artifact.Comment(dir)
		}
	}
	label := "comment " + keys[0]
	if len(keys) > 1 {
		label = fmt.Sprintf("comment %d run(s): %s", len(keys), strings.Join(keys, " "))
		current = ""
	}
	text := a.promptComment(label, current)
	if len(keys) > 1 && text == "" {
		return fmt.Errorf("comment cancelled")
	}
	for _, key := range keys {
		if err := artifact.SetComment(a.root, key, text); err != nil {
			return err
		}
		fmt.Fprintf(a.stderr, "jmark: commented %s\n", key)
	}
	return nil
}
