// Command pr-report reports open pull requests across GitHub repositories.
package main

import (
	"fmt"
	"io"
	"os"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, executeReport))
}

// run validates flags and repository sources before the execution stage, which owns gh
// discovery and API calls when collection is implemented.
func run(args []string, stdout, stderr io.Writer, execute func(config, io.Writer, io.Writer) int) int {
	cfg, err := parseConfig(args)
	if err != nil {
		fmt.Fprintf(stderr, "pr-report: %v\n", err)
		return 2
	}
	// Help takes precedence when both informational flags are requested.
	switch {
	case cfg.help:
		err = writeHelp(stdout)
	case cfg.version:
		_, err = fmt.Fprintf(stdout, "pr-report %s\n", version)
	default:
		cfg.repos, err = loadRepositories(cfg.reposFile, cfg.repos)
		if err != nil {
			fmt.Fprintf(stderr, "pr-report: %v\n", err)
			return 2
		}
		return execute(cfg, stdout, stderr)
	}
	if err != nil {
		fmt.Fprintf(stderr, "pr-report: write output: %v\n", err)
		return 1
	}
	return 0
}

func executeReport(_ config, _ io.Writer, stderr io.Writer) int {
	fmt.Fprintln(stderr, "pr-report: report collection is not implemented yet")
	return 1
}
