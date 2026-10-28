package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (a *app) cmdRuntimeMise(args []string) error {
	if args[0] != "lock" && args[0] != "install" {
		return fmt.Errorf("usage: runtimes [lock|install [<runtime>...]]")
	}
	command := []string{args[0]}
	if args[0] == "install" {
		command = append(command, "--locked")
	}
	for _, id := range args[1:] {
		if !a.cat.RuntimeExists(id) {
			return a.unknownRuntime(id)
		}
		if id == "py" {
			id = "python"
		}
		command = append(command, id)
	}
	return a.runMise(command)
}

func (a *app) runMise(args []string) error {
	cmd := exec.Command(filepath.Join(a.root, "scripts", "mise"), args...)
	cmd.Dir = a.root
	cmd.Stdin, cmd.Stdout, cmd.Stderr = a.in, a.stdout, a.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mise: %w", err)
	}
	return nil
}

func managedBench(args []string, root string) int {
	data, err := os.MkdirTemp("", "jmark-mise-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer os.RemoveAll(data)

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	cmd := exec.Command(filepath.Join(root, "scripts", "mise"), append([]string{"exec", "--locked", "--", exe}, args...)...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "JMARK_MISE_ACTIVE=1", "JMARK_MISE_DATA_DIR="+data)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
