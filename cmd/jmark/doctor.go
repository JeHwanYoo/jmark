package main

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/JeHwanYoo/jmark/internal/measure"
)

func (a *app) cmdDoctor() error {
	ok := true
	for _, name := range []string{"just", "bash", "hyperfine", "mise"} {
		line, found := commandVersion(name)
		if !found {
			a.warn("missing host command: %s", name)
			ok = false
			continue
		}
		fmt.Fprintf(a.stdout, "doctor: found %s (%s)\n", name, line)
	}
	if !ok {
		return fmt.Errorf("doctor: FAILED")
	}
	if err := a.cat.CheckRegistry(); err != nil {
		a.writeErr(err)
		ok = false
	} else {
		fmt.Fprintln(a.stdout, "doctor: registry ok")
	}
	ids, err := a.cat.ListProjects()
	if err != nil {
		a.writeErr(err)
		ok = false
	} else {
		for _, id := range ids {
			if err := a.cat.CheckProject(id); err != nil {
				a.writeErr(err)
				ok = false
				continue
			}
			runtimes, err := a.cat.ProjectRuntimes(id)
			if err != nil {
				a.writeErr(err)
				ok = false
				continue
			}
			fmt.Fprintf(a.stdout, "doctor: project ok %s (runtimes: %s)\n", id, strings.Join(runtimes, " "))
		}
	}
	runtimes, err := a.cat.ListRuntimes()
	if err != nil {
		a.writeErr(err)
		ok = false
	} else {
		m := &measure.Measure{Root: a.root, Stderr: a.stderr}
		for _, id := range runtimes {
			runtime, err := a.cat.Runtime(id)
			if err != nil {
				a.writeErr(err)
				ok = false
				continue
			}
			missing, err := a.cat.RuntimeMissing(id)
			if err != nil {
				a.writeErr(err)
				ok = false
				continue
			}
			if len(missing) > 0 {
				fmt.Fprintf(a.stdout, "doctor: runtime %s (%s) missing: %s\n", id, runtime.Family, strings.Join(missing, " "))
				continue
			}
			fmt.Fprintf(a.stdout, "doctor: runtime %s (%s) found\n", id, runtime.Family)
			if runtime.Probes == "" {
				continue
			}
			plan, err := a.cat.SelftestPlan(id)
			if err != nil {
				a.writeErr(err)
				a.warn("probe self-test failed for %s", id)
				ok = false
				continue
			}
			if err := m.ProbeSelftest(plan); err != nil {
				a.writeErr(err)
				a.warn("probe self-test failed for %s", id)
				ok = false
				continue
			}
			fmt.Fprintf(a.stdout, "doctor: probe self-test ok %s\n", id)
		}
	}
	if !ok {
		return fmt.Errorf("doctor: FAILED")
	}
	fmt.Fprintln(a.stdout, "doctor: OK")
	return nil
}

func commandVersion(name string) (string, bool) {
	if _, err := exec.LookPath(name); err != nil {
		return "", false
	}
	out, _ := exec.Command(name, "--version").CombinedOutput()
	return firstLine(string(out)), true
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
