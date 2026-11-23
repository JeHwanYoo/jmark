package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/artifact"
)

const commentUsage = `usage: comment [<project>/<run> | <visibility>/<project>/<run>] [<text>... | --clear]

Set the one-line summary shown beside a run in open, delete, public, and private.
  no run    pick one run with fzf, then type the comment
  no text   print the comment, or prompt for one on a terminal
  --clear   remove the comment
`

func (a *app) cmdComment(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Fprint(a.stdout, commentUsage)
		return nil
	}
	if len(args) == 0 {
		action, root, dirs, err := a.pickArtifactRuns("", false, "comment")
		if err != nil {
			return a.fzfErr(err, "comment needs fzf (or pass: comment <project>/<run> <text>)")
		}
		if action == artifactQuit {
			return errSoftQuit
		}
		return a.commentRuns(root, dirs)
	}
	key := strings.TrimSuffix(strings.TrimPrefix(args[0], "artifacts/"), "/")
	if len(strings.Split(key, "/")) == 2 {
		key = a.findRunKey(key)
	}
	dir, err := artifact.Resolve(a.root, key)
	if err != nil {
		return err
	}
	text := strings.Join(args[1:], " ")
	switch {
	case len(args) == 2 && args[1] == "--clear":
		text = ""
	case len(args) == 1 && a.stdinTTY():
		return a.commentRuns(filepath.Join(a.root, "artifacts"), []string{dir})
	case len(args) == 1:
		if comment := artifact.Comment(dir); comment != "" {
			fmt.Fprintln(a.stdout, comment)
		}
		return nil
	}
	if err := artifact.SetComment(a.root, key, text); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, "artifacts/"+key)
	return nil
}

// findRunKey adds the visibility to a <project>/<run> key, preferring private.
func (a *app) findRunKey(key string) string {
	for _, scope := range []string{"private", "public"} {
		if _, err := artifact.Resolve(a.root, scope+"/"+key); err == nil {
			return scope + "/" + key
		}
	}
	return "private/" + key
}
