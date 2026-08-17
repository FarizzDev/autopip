package main

import (
	"os"
	"os/exec"
	"strings"
)

func isTermux() bool {
	return strings.Contains(os.Getenv("PREFIX"), "com.termux")
}

func isGlibc() bool {
	out, err := exec.Command("python3", "-c", "import platform; print(platform.libc_ver()[0])").Output()
	if err != nil {
		// Fall back to `python` if `python3` isn't on PATH.
		out, err = exec.Command("python", "-c", "import platform; print(platform.libc_ver()[0])").Output()
		if err != nil {
			return false
		}
	}
	return strings.TrimSpace(string(out)) == "glibc"
}

func detectDefaultAllowBinary() bool {
	if isTermux() {
		return false
	}
	return isGlibc()
}
