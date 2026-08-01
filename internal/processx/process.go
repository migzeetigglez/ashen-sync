package processx

import (
	"os/exec"
	"runtime"
	"strings"
)

// Running reports whether any of the given process names appear to be running.
func Running(names []string) (bool, error) {
	if len(names) == 0 {
		return false, nil
	}
	switch runtime.GOOS {
	case "windows":
		return runningWindows(names)
	default:
		return runningUnix(names)
	}
}

func runningUnix(names []string) (bool, error) {
	out, err := exec.Command("ps", "-A", "-o", "comm=").Output()
	if err != nil {
		// fallback
		out, err = exec.Command("ps", "-ax", "-o", "comm=").Output()
		if err != nil {
			return false, err
		}
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		comm := strings.TrimSpace(line)
		base := comm
		if i := strings.LastIndex(comm, "/"); i >= 0 {
			base = comm[i+1:]
		}
		for _, n := range names {
			if strings.EqualFold(base, n) || strings.EqualFold(base, n+".exe") {
				return true, nil
			}
		}
	}
	return false, nil
}

func runningWindows(names []string) (bool, error) {
	out, err := exec.Command("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return false, err
	}
	lower := strings.ToLower(string(out))
	for _, n := range names {
		cand := strings.ToLower(n)
		if !strings.HasSuffix(cand, ".exe") {
			cand += ".exe"
		}
		if strings.Contains(lower, strings.ToLower(cand)) {
			return true, nil
		}
	}
	return false, nil
}
