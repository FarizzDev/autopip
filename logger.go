package main

import (
	"fmt"
	"strings"
	"time"
)

// ANSI styling.
const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	purple = "\033[35m"
	cyan   = "\033[0;36m"
	gray   = "\033[90m"
)

func now() string {
	return time.Now().Format("15:04:05")
}

func logf(color, tag, format string, args ...any) {
	fmt.Printf("%s[%s]%s %s%s%-7s%s %s\n",
		gray, now(), reset,
		color, bold, tag, reset,
		fmt.Sprintf(format, args...))
}

func info(format string, a ...any)    { logf(blue, "INFO", format, a...) }
func warn(format string, a ...any)    { logf(yellow, "WARN", format, a...) }
func errlog(format string, a ...any)  { logf(red, "ERROR", format, a...) }
func success(format string, a ...any) { logf(green, "OK", format, a...) }
func step(format string, a ...any)    { logf(purple, "STEP", format, a...) }

func banner(pkg string) {
	line := strings.Repeat("─", 58)
	fmt.Printf("\n%s%s┌%s┐%s\n", cyan, bold, line, reset)
	fmt.Printf("%s%s│%s  %sAUTOPIP%s — resolving build for %s%s%s\n",
		cyan, bold, reset, cyan, reset, bold, pkg, reset)
	fmt.Printf("%s%s└%s┘%s\n\n", cyan, bold, line, reset)
}

func section(title string) {
	pad := max(50-len(title), 0)
	fmt.Printf("\n%s%s— %s %s%s\n", dim, bold, title, strings.Repeat("─", pad), reset)
}

func printRawLog(output string) {
	sep := strings.Repeat("─", 58)
	fmt.Printf("\n%s%s%s\n", gray, sep, reset)
	fmt.Printf("%sRAW ERROR LOG%s\n", dim, reset)
	fmt.Printf("%s%s%s\n", gray, sep, reset)
	fmt.Println(output)
}
