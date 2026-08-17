package main

import "testing"

// To add a new pattern: add its ErrorPattern to patterns.go, then add a
// case here with a real (or representative) error snippet and the
// target you expect autopip to resolve.
var detectionCases = []struct {
	name       string
	output     string
	wantTarget string
	wantKind   Kind
}{
	{
		name: "pkg-config CLI format",
		output: `Package pangocairo was not found in the pkg-config search path.
No package 'pangocairo' found`,
		wantTarget: "pangocairo.pc",
		wantKind:   KindSystem,
	},
	{
		name: "pkg-config custom check (manimpango-style)",
		output: `      Package 'pangocairo' was not found
      Traceback (most recent call last):
        File "<string>", line 137, in check_min_version`,
		wantTarget: "pangocairo.pc",
		wantKind:   KindSystem,
	},
	{
		name:       "pkg-config via subprocess argv",
		output:     `subprocess.CalledProcessError: Command '['pkg-config', '--print-errors', '--atleast-version', '1.30.0', 'pangocairo']' returned non-zero exit status 1.`,
		wantTarget: "pangocairo.pc",
		wantKind:   KindSystem,
	},
	{
		name:       "missing header",
		output:     `cairo.c:10:10: fatal error: cairo.h: No such file or directory`,
		wantTarget: "cairo.h",
		wantKind:   KindSystem,
	},
	{
		name:       "linker missing library",
		output:     `/usr/bin/ld: cannot find -lffi: No such file or directory`,
		wantTarget: "libffi.so",
		wantKind:   KindSystem,
	},
	{
		name:       "missing shell tool",
		output:     `sh: 1: rustc: command not found`,
		wantTarget: "bin/rustc",
		wantKind:   KindSystem,
	},
	{
		name:       "missing python module",
		output:     `ModuleNotFoundError: No module named 'Cython'`,
		wantTarget: "Cython",
		wantKind:   KindPip,
	},
	{
		name:       "rust toolchain missing (maturin)",
		output:     `      Rust not found, installing into a temporary directory`,
		wantTarget: "rust",
		wantKind:   KindPkgDirect,
	},
}

func TestDetectAll(t *testing.T) {
	for _, tc := range detectionCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := detectAll(tc.output)
			if len(matches) == 0 {
				t.Fatalf("no pattern matched; expected target %q", tc.wantTarget)
			}

			var found bool
			for _, m := range matches {
				if m.Target == tc.wantTarget {
					found = true
					if m.Kind != tc.wantKind {
						t.Errorf("target %q matched with Kind=%v, want %v", tc.wantTarget, m.Kind, tc.wantKind)
					}
				}
			}
			if !found {
				t.Errorf("expected target %q not found among matches: %+v", tc.wantTarget, matches)
			}
		})
	}
}

// TestDetectAllNoFalsePositives guards against overly broad patterns
// matching output that isn't actually a build error, which would cause
// autopip to "resolve" something unrelated and mask the real problem.
func TestDetectAllNoFalsePositives(t *testing.T) {
	clean := `Collecting numpy
  Using cached numpy-2.1.0-cp312-cp312-manylinux_2_17_x86_64.whl
Successfully installed numpy-2.1.0`

	if matches := detectAll(clean); len(matches) != 0 {
		t.Errorf("expected no matches on clean install output, got: %+v", matches)
	}
}
