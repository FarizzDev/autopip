package main

import (
	"fmt"
	"strings"
)

type Action string

const (
	ActionWheel   Action = "wheel"   // build a wheel without installing it
	ActionInstall Action = "install" // install a package
	ActionUpgrade Action = "upgrade" // upgrade an already-installed package
)

func (a Action) Valid() bool {
	switch a {
	case ActionWheel, ActionInstall, ActionUpgrade:
		return true
	}
	return false
}

func (a Action) Description() string {
	switch a {
	case ActionWheel:
		return "build a wheel without installing it"
	case ActionInstall:
		return "install a package"
	case ActionUpgrade:
		return "upgrade an already-installed package"
	}
	return ""
}

type Backend interface {
	Name() string
	// Command returns the full argv (including the executable name) to run for the given action/package.
	Command(action Action, pkg string, buildFromSource bool) []string
}

var availableBackends = []string{"pip"}

func getBackend(name string) (Backend, error) {
	switch name {
	case "pip":
		return pipBackend{}, nil
	case "uv":
		return nil, fmt.Errorf("backend 'uv' is not implemented yet — planned for a future release")
	default:
		return nil, fmt.Errorf("unknown backend %q [possible values: %s]", name, strings.Join(availableBackends, ", "))
	}
}

/* pip */

type pipBackend struct{}

func (pipBackend) Name() string { return "pip" }

func (pipBackend) Command(action Action, pkg string, buildFromSource bool) []string {
	var args []string
	switch action {
	case ActionWheel:
		args = []string{"pip", "wheel"}
	case ActionInstall:
		args = []string{"pip", "install"}
	case ActionUpgrade:
		args = []string{"pip", "install", "--upgrade"}
	}
	if buildFromSource {
		// Forces building from source instead of pulling a prebuilt
		// manylinux/glibc wheel from PyPI, which is not ABI-compatible
		// with Termux's bionic libc.
		args = append(args, "--no-binary", ":all:")
	}
	args = append(args, pkg)
	return args
}
