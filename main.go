package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const version = "0.6.3"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "-version", "--version":
		fmt.Printf("autopip %s\n", version)
		os.Exit(0)
	case "-h", "--help", "help":
		printUsage()
		os.Exit(0)
	}

	action := Action(os.Args[1])
	if !action.Valid() {
		fmt.Fprintf(os.Stderr, "%s%serror:%s unknown command %s'%s'%s\n\n", red, bold, reset, yellow, os.Args[1], reset)
		printUsage()
		os.Exit(2)
	}

	defaultAllowBinary := detectDefaultAllowBinary()

	fs := flag.NewFlagSet("autopip "+string(action), flag.ExitOnError)
	maxRetries := fs.Int("max-retries", 15, "maximum number of dependency-resolution attempts before giving up")
	backendName := fs.String("backend", "pip", "package manager backend to use")
	allowBinary := fs.Bool("allow-binary", defaultAllowBinary, "allow using a prebuilt wheel instead of building from source")

	flagArgs, positional := splitFlags(os.Args[2:])
	fs.Parse(flagArgs)
	positional = append(positional, fs.Args()...)

	explicitAllowBinary := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "allow-binary" {
			explicitAllowBinary = true
		}
	})

	if len(positional) < 1 {
		fmt.Fprintf(os.Stderr, "%s%serror:%s missing <package> argument\n", red, bold, reset)
		fmt.Fprintf(os.Stderr, "%s%sUsage: %s%sautopip %s %s[flags] <package>%s\n", green, bold, cyan, bold, action, cyan, reset)
		os.Exit(2)
	}
	targetPkg := positional[0]

	backend, err := getBackend(*backendName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s%serror:%s %v\n", red, bold, reset, err)
		os.Exit(2)
	}

	argv := backend.Command(action, targetPkg, !*allowBinary)
	runLoop(runConfig{
		action:              action,
		backend:             backend,
		argv:                argv,
		targetPkg:           targetPkg,
		maxRetries:          *maxRetries,
		allowBinary:         *allowBinary,
		explicitAllowBinary: explicitAllowBinary,
	})
}

type runConfig struct {
	action              Action
	backend             Backend
	argv                []string
	targetPkg           string
	maxRetries          int
	allowBinary         bool
	explicitAllowBinary bool
}

func runLoop(cfg runConfig) {
	start := time.Now()
	banner(fmt.Sprintf("%s %s via %s", cfg.action, cfg.targetPkg, cfg.backend.Name()))

	if !cfg.explicitAllowBinary {
		if cfg.allowBinary {
			info("Detected a glibc environment, prebuilt wheels are allowed by default.")
		} else {
			info("Detected Termux/bionic (or an unrecognized) environment, building from source by default.")
		}
	}

	if err := ensureSystemPackageManagerReady(); err != nil {
		fmt.Println()
		errlog("Could not prepare a system package manager: %v", err)
		errlog("Automatic dependency resolution needs this to work, so stopping now")
		errlog("Fix the issue above (often: needs sudo, or run as root) and try again")
		os.Exit(1)
	}

	seen := map[string]int{}

	for attempt := 1; attempt <= cfg.maxRetries; attempt++ {
		section(fmt.Sprintf("Attempt %d/%d", attempt, cfg.maxRetries))
		info("Running: %s%s%s", bold, strings.Join(cfg.argv, " "), reset)

		output, err := runBuild(cfg.argv)
		if err == nil {
			elapsed := time.Since(start).Round(time.Second)
			fmt.Println()
			success("%s completed in %s.", capitalize(string(cfg.action)), elapsed)
			os.Exit(0)
		}

		matches := detectAll(output)
		if len(matches) == 0 {
			fmt.Println()
			errlog("No known pattern matched the error below:")
			printRawLog(output)
			os.Exit(1)
		}
		if len(matches) > 1 {
			info("Detected %d missing dependencies in this attempt — resolving all before retrying.", len(matches))
		}

		resolvedAny := false
		stuck := false
		for _, m := range matches {
			warn("Detected [%s]: %s%s%s", m.PatternName, bold, m.Target, reset)

			seen[m.Target]++
			if seen[m.Target] > 2 {
				errlog("Target '%s' was retried %d times but the same error keeps recurring.", m.Target, seen[m.Target])
				stuck = true
				continue
			}

			if resolve(m.Target, m.Kind) {
				resolvedAny = true
			} else {
				errlog("Failed to resolve dependency '%s'.", m.Target)
			}
		}

		if stuck || !resolvedAny {
			fmt.Println()
			errlog("Stopping: unable to make further progress.")
			printRawLog(output)
			os.Exit(1)
		}
	}

	fmt.Println()
	errlog("Reached the maximum of %d attempts. Stopping.", cfg.maxRetries)
	os.Exit(1)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var boolFlags = map[string]bool{
	"allow-binary": true,
}

func splitFlags(args []string) (flags, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}

		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") {
			continue
		}
		if boolFlags[name] {
			continue
		}
		if i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return flags, positional
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "A pip wrapper that installs its own missing dependencies.\n\n")
	fmt.Fprintf(os.Stderr, "%s%sUsage: %s%sautopip %s<action> [flags] <package>%s\n\n", green, bold, cyan, bold, cyan, reset)
	fmt.Fprintf(os.Stderr, "%s%sCommands:%s\n", green, bold, reset)
	for _, a := range []Action{ActionWheel, ActionInstall, ActionUpgrade} {
		fmt.Fprintf(os.Stderr, "  %s%s%-10s %s%s\n", cyan, bold, a, reset, a.Description())
	}
	fmt.Fprintf(os.Stderr, "\n%s%sExamples:%s\n", green, bold, reset)
	fmt.Fprintf(os.Stderr, "  autopip wheel manim\n")
	fmt.Fprintf(os.Stderr, "  autopip install pandas\n")
	fmt.Fprintf(os.Stderr, "  autopip upgrade --max-retries 5 numpy\n\n")
	fmt.Fprintf(os.Stderr, "%s%sCommand options:%s\n", green, bold, reset)
	fmt.Fprintf(os.Stderr, "      %s%s--max-retries %s<N>%s\n          maximum resolution attempts (default 15)\n", cyan, bold, cyan, reset)
	fmt.Fprintf(os.Stderr, "      %s%s--backend %s<NAME>%s\n          package manager backend (default: pip) [possible values: %s]\n", cyan, bold, cyan, reset, strings.Join(availableBackends, ", "))
	fmt.Fprintf(os.Stderr, "      %s%s--allow-binary%s\n", cyan, bold, reset)
	fmt.Fprintf(os.Stderr, "          Allow using a prebuilt wheel instead of building from source\n")
	fmt.Fprintf(os.Stderr, "          [default: auto-detected — off on Termux/bionic, on if glibc is detected]\n")
	fmt.Fprintf(os.Stderr, "          PyPI's manylinux/glibc wheels aren't ABI-compatible with bionic, so\n")
	fmt.Fprintf(os.Stderr, "          building from source is the safe default there. Override explicitly if\n")
	fmt.Fprintf(os.Stderr, "          the auto-detection guesses wrong for your environment.\n\n")
	fmt.Fprintf(os.Stderr, "%s%sGlobal options:%s\n", green, bold, reset)
	fmt.Fprintf(os.Stderr, "      %s%s--version%s\n          print version and exit\n", cyan, bold, reset)
	fmt.Fprintf(os.Stderr, "      %s%s--help%s\n          show this help\n", cyan, bold, reset)
}
