package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// runBuild runs the given argv (e.g. ["pip", "install", "pandas"]),
// streaming output live to the terminal while also capturing it
// (stdout+stderr combined) for pattern detection if the build fails.
func runBuild(argv []string) (string, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	var combined bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &combined)
	cmd.Stderr = io.MultiWriter(os.Stderr, &combined)
	err := cmd.Run()
	return combined.String(), err
}

// run executes a command with output streamed directly to the terminal.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// runQuiet executes a command without streaming its output.
func runQuiet(name string, args ...string) error {
	return exec.Command(name, args...).Run()
}
