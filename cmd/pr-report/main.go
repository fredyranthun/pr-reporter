// Command pr-report reports open pull requests across GitHub repositories.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
)

var version = "v0.1.1"

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

func executeReport(cfg config, stdout, stderr io.Writer) int {
	executor, err := locateGH()
	if err != nil {
		fmt.Fprintln(stderr, "pr-report: gh was not found in PATH")
		return 1
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return runReport(ctx, cfg, stdout, stderr, newClient(executor))
}
func runReport(ctx context.Context, cfg config, stdout, stderr io.Writer, c *client) int {

	if cfg.timeout > 0 {
		c.timeout = cfg.timeout
	}
	r := c.collect(ctx, cfg)
	if ctx.Err() != nil {
		return 130
	}
	for _, entries := range []list[diagnostic]{r.Errors, r.Warnings} {
		for _, d := range entries {
			repo := "?"
			if d.Repo != nil {
				repo = *d.Repo
			}
			number := "?"
			if d.PRNumber != nil {
				number = fmt.Sprint(*d.PRNumber)
			}
			fmt.Fprintf(stderr, "pr-report: %q PR %s %s (%s): %q\n", repo, number, d.Stage, d.Code, d.Message)
		}
	}
	render := func(w io.Writer, r report) error { return writeTableFields(w, r, cfg.fields) }
	if cfg.format == "json" {
		render = func(w io.Writer, r report) error { return writeJSONFields(w, r, cfg.fields) }
	}
	if err := render(stdout, r); err != nil {
		fmt.Fprintf(stderr, "pr-report: write output: %v\n", err)
		return 1
	}
	if !r.Complete {
		return 3
	}
	return 0
}
