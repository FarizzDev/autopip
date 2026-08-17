package main

import (
	"fmt"
	"regexp"
)

// Kind describes how a detected missing dependency should be resolved.
type Kind int

const (
	KindSystem    Kind = iota // resolve via the system package manager
	KindPip                   // resolve via pip install
	KindPkgDirect             // target is already a known package name, install directly
)

type ErrorPattern struct {
	Name     string
	Regex    *regexp.Regexp
	FormatTo string
	Kind     Kind
}

var patterns = []ErrorPattern{
	{
		Name:     "pkg-config",
		Regex:    regexp.MustCompile(`No package '(.*?)' found`),
		FormatTo: "%s.pc",
		Kind:     KindSystem,
	},
	{
		Name:     "pkg-config (custom check)",
		Regex:    regexp.MustCompile(`Package '(.*?)' was not found`),
		FormatTo: "%s.pc",
		Kind:     KindSystem,
	},
	{
		Name:     "pkg-config (subprocess argv)",
		Regex:    regexp.MustCompile(`Command '\['pkg-config'(?:, '[^']*')*, '([a-zA-Z0-9_+-]+)'\]' returned non-zero`),
		FormatTo: "%s.pc",
		Kind:     KindSystem,
	},
	{
		Name:     "missing header",
		Regex:    regexp.MustCompile(`fatal error: (.*?\.h[a-z]*): No such file`),
		FormatTo: "%s",
		Kind:     KindSystem,
	},
	{
		Name:     "linker",
		Regex:    regexp.MustCompile(`cannot find -l([a-zA-Z0-9_]+)`),
		FormatTo: "lib%s.so",
		Kind:     KindSystem,
	},
	{
		Name:     "missing tool",
		Regex:    regexp.MustCompile(`([a-zA-Z0-9_]+): command not found`),
		FormatTo: "bin/%s",
		Kind:     KindSystem,
	},
	{
		Name:     "missing python module",
		Regex:    regexp.MustCompile(`ModuleNotFoundError: No module named '([a-zA-Z0-9_.]+)'`),
		FormatTo: "%s",
		Kind:     KindPip,
	},
	{
		Name:     "rust toolchain missing",
		Regex:    regexp.MustCompile(`Rust not found, installing into a temporary directory`),
		FormatTo: "rust",
		Kind:     KindPkgDirect,
	},
}

type Match struct {
	PatternName string
	Target      string
	Kind        Kind
}

func detectAll(output string) []Match {
	seenTargets := map[string]bool{}
	var matches []Match

	for _, pat := range patterns {
		for _, m := range pat.Regex.FindAllStringSubmatch(output, -1) {
			var target string
			if len(m) > 1 {
				target = fmt.Sprintf(pat.FormatTo, m[1])
			} else {
				target = pat.FormatTo
			}
			if seenTargets[target] {
				continue
			}
			seenTargets[target] = true
			matches = append(matches, Match{PatternName: pat.Name, Target: target, Kind: pat.Kind})
		}
	}
	return matches
}
