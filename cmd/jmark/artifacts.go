package main

import (
	"fmt"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/artifact"
)

func (a *app) cmdVisibility(target string, args []string) error {
	usage := "usage: " + target + " [<project>/<run> | <visibility>/<project>/<run>]"
	if len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		fmt.Fprintln(a.stdout, usage)
		return nil
	}
	if len(args) > 1 {
		return fmt.Errorf("%s", usage)
	}
	source := "private"
	if target == "private" {
		source = "public"
	}
	var picks []string
	if len(args) == 1 {
		key := strings.TrimPrefix(args[0], "artifacts/")
		if len(strings.Split(key, "/")) == 2 {
			key = source + "/" + key
		}
		picks = []string{key}
	} else {
		runs, err := artifact.List(a.root, source, "")
		if err != nil {
			return err
		}
		if len(runs) == 0 {
			return fmt.Errorf("no %s runs", source)
		}
		key, selected, err := a.runFzf(linesInput(runs), "-m", "--expect=q", "--prompt=jmark "+target+"> ", "--height=40%", "--reverse", "--border", "--header=Tab: toggle · Enter: move to "+target+" · q: quit")
		if err != nil {
			return err
		}
		if key == "q" || len(selected) == 0 {
			return errSoftQuit
		}
		allowed := map[string]bool{}
		for _, run := range runs {
			allowed[run] = true
		}
		for _, run := range selected {
			if !allowed[run] {
				return fmt.Errorf("invalid pick: %s", run)
			}
		}
		picks = selected
	}
	for _, key := range picks {
		moved, err := artifact.Move(a.root, key, target)
		if err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, "artifacts/"+moved)
	}
	return nil
}

func (a *app) cmdSite(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: site")
	}
	path, err := artifact.BuildSite(a.root)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, path)
	return nil
}
