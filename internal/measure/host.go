package measure

import (
	"os/exec"
	"strconv"
	"strings"
)

type hostSnap struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Kernel    string `json:"kernel"`
	CPU       string `json:"cpu"`
	Hyperfine string `json:"hyperfine"`
	NCPU      *int   `json:"ncpu,omitempty"`
}

// HostSnapshot returns one JSON object describing this machine at bench time.
func HostSnapshot() ([]byte, error) {
	osName, err := output("uname", "-s")
	if err != nil {
		return nil, err
	}
	arch, err := output("uname", "-m")
	if err != nil {
		return nil, err
	}
	kernel, err := output("uname", "-r")
	if err != nil {
		return nil, err
	}
	var cpu, ncpu string
	if osName == "Darwin" {
		cpu = outputIgnore("sysctl", "-n", "machdep.cpu.brand_string")
		ncpu = outputIgnore("sysctl", "-n", "hw.ncpu")
	} else {
		cpu = cpuinfoModel(outputIgnore("cat", "/proc/cpuinfo"))
		ncpu = outputIgnore("nproc")
	}
	return encodeHost(osName, arch, kernel, cpu, hyperfineVersion(), ncpu)
}

// RuntimeVersion captures the first line of the registry's version command.
func RuntimeVersion(command string) (string, error) {
	if command == "" {
		return "", nil
	}
	args := splitSpaces(command)
	if len(args) == 0 {
		return "", nil
	}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return "", jerr("runtime version command failed: %s: %v", command, err)
	}
	return strings.TrimSpace(firstLine(string(out))), nil
}

func encodeHost(osName, arch, kernel, cpu, hf, ncpu string) ([]byte, error) {
	snap := hostSnap{
		OS: osName, Arch: arch, Kernel: kernel, CPU: cpu, Hyperfine: hf,
	}
	if ncpu != "" {
		n, err := strconv.Atoi(strings.TrimSpace(ncpu))
		if err != nil {
			return nil, jerr("ncpu: %v", err)
		}
		snap.NCPU = &n
	}
	return marshalCompact(snap)
}

func cpuinfoModel(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if !strings.Contains(line, "model name") {
			continue
		}
		_, rest, ok := strings.Cut(line, ": ")
		if ok {
			return rest
		}
		return ""
	}
	return ""
}

func hyperfineVersion() string {
	b, err := exec.Command("hyperfine", "--version").CombinedOutput()
	if len(b) > 0 {
		return firstLine(string(b))
	}
	if err != nil {
		return firstLine(err.Error())
	}
	return ""
}

func output(name string, args ...string) (string, error) {
	b, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", err
	}
	return stripTrailingNewlines(string(b)), nil
}

func outputIgnore(name string, args ...string) string {
	s, err := output(name, args...)
	if err != nil {
		return ""
	}
	return s
}

func firstLine(s string) string {
	if i := strings.IndexAny(s, "\r\n"); i >= 0 {
		return s[:i]
	}
	return s
}
