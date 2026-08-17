package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Candidate struct {
	Name string
	Path string
}

type SystemPackageManager interface {
	Name() string
	EnsureReady() error
	Search(target string) ([]Candidate, error)
	Install(pkgName string) error
}

var (
	sysPM     SystemPackageManager
	sysPMErr  error
	sysPMOnce sync.Once
)

func getSystemPackageManager() (SystemPackageManager, error) {
	sysPMOnce.Do(func() {
		sysPM, sysPMErr = detectSystemPackageManager()
		if sysPMErr == nil {
			info("Detected system package manager: %s%s%s", bold, sysPM.Name(), reset)
		}
	})
	return sysPM, sysPMErr
}

func detectSystemPackageManager() (SystemPackageManager, error) {
	if isTermux() {
		return pkgManager{}, nil
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return dnfManager{}, nil
	}
	if _, err := exec.LookPath("apt-get"); err == nil {
		return aptManager{}, nil
	}
	return nil, fmt.Errorf("no supported system package manager detected (checked: Termux pkg, dnf, apt)")
}

func ensureSystemPackageManagerReady() error {
	pm, err := getSystemPackageManager()
	if err != nil {
		return err
	}
	step("Preparing %s...", pm.Name())
	if err := pm.EnsureReady(); err != nil {
		return err
	}
	success("%s is ready.", pm.Name())
	return nil
}

func resolveSystemPackage(target string) bool {
	pm, err := getSystemPackageManager()
	if err != nil {
		errlog("%v", err)
		return false
	}

	step("Searching for a package providing '%s' via %s...", target, pm.Name())
	candidates, err := pm.Search(target)
	if err != nil {
		errlog("%v", err)
		return false
	}
	if len(candidates) == 0 {
		errlog("%s found no candidates for '%s'.", pm.Name(), target)
		return false
	}
	if len(candidates) > 1 {
		info("Found %d candidates, picking the shortest match as a heuristic.", len(candidates))
	}

	best := pickBestCandidate(candidates)
	if best == "" {
		errlog("Could not determine the best candidate for '%s'.", target)
		return false
	}

	success("Candidate found: %s%s%s", bold, best, reset)
	step("Installing: %s install %s", pm.Name(), best)
	if err := pm.Install(best); err != nil {
		return false
	}
	success("Package '%s' installed. Retrying build...", best)
	return true
}

func pickBestCandidate(candidates []Candidate) string {
	var best string
	shortest := 1 << 30
	for _, c := range candidates {
		key := c.Path
		if key == "" {
			key = c.Name
		}
		if len(key) < shortest {
			shortest = len(key)
			best = c.Name
		}
	}
	return best
}

func privileged(argv ...string) []string {
	if os.Geteuid() == 0 {
		return argv
	}
	if _, err := exec.LookPath("sudo"); err == nil {
		return append([]string{"sudo"}, argv...)
	}
	return argv
}

func runArgv(argv []string) error      { return run(argv[0], argv[1:]...) }
func runQuietArgv(argv []string) error { return runQuiet(argv[0], argv[1:]...) }

func parseAptFileOutput(output string) []Candidate {
	var candidates []Candidate
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		candidates = append(candidates, Candidate{Name: parts[0], Path: parts[1]})
	}
	return candidates
}

/* pkg (Termux) */

type pkgManager struct{}

func (pkgManager) Name() string { return "pkg" }

func (pkgManager) EnsureReady() error {
	if _, err := exec.LookPath("apt-file"); err != nil {
		info("apt-file not found, installing it...")
		if err := runQuiet("pkg", "install", "-y", "apt-file"); err != nil {
			return fmt.Errorf("failed to install apt-file: %w", err)
		}
	}
	info("Updating apt-file index...")
	if err := runQuiet("apt-file", "update"); err != nil {
		return fmt.Errorf("apt-file update failed: %w", err)
	}
	return nil
}

func (pkgManager) Search(target string) ([]Candidate, error) {
	out, err := exec.Command("apt-file", "search", target).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return nil, nil
	}
	return parseAptFileOutput(string(out)), nil
}

func (pkgManager) Install(name string) error {
	return run("pkg", "install", "-y", name)
}

/* apt (Debian/Ubuntu) */

type aptManager struct{}

func (aptManager) Name() string { return "apt" }

func (aptManager) EnsureReady() error {
	if _, err := exec.LookPath("apt-file"); err != nil {
		info("apt-file not found, installing it...")
		if err := runQuietArgv(privileged("apt-get", "install", "-y", "apt-file")); err != nil {
			return fmt.Errorf("failed to install apt-file (are you root, or is sudo available? current euid=%d): %w", os.Geteuid(), err)
		}
	}
	info("Updating apt-file index...")
	if err := runQuietArgv(privileged("apt-file", "update")); err != nil {
		return fmt.Errorf("apt-file update failed: %w", err)
	}
	return nil
}

func (aptManager) Search(target string) ([]Candidate, error) {
	out, err := exec.Command("apt-file", "search", target).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return nil, nil
	}
	return parseAptFileOutput(string(out)), nil
}

func (aptManager) Install(name string) error {
	return runArgv(privileged("apt-get", "install", "-y", name))
}

/* ── dnf (Fedora/RHEL) */

type dnfManager struct{}

func (dnfManager) Name() string { return "dnf" }

func (dnfManager) EnsureReady() error {
	info("Refreshing dnf cache...")
	if err := runArgv(privileged("dnf", "makecache")); err != nil {
		warn("dnf makecache failed, search results may use a stale cache.")
	}
	return nil
}

func (dnfManager) Search(target string) ([]Candidate, error) {
	out, err := exec.Command("dnf", "repoquery", "--file", target, "--qf", "%{name}\n").Output()
	if err != nil {
		return nil, fmt.Errorf("dnf repoquery failed (is dnf-plugins-core installed?): %w", err)
	}
	seen := map[string]bool{}
	var candidates []Candidate
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		candidates = append(candidates, Candidate{Name: name})
	}
	return candidates, nil
}

func (dnfManager) Install(name string) error {
	return runArgv(privileged("dnf", "install", "-y", name))
}
