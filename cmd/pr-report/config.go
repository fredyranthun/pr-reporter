package main

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

type config struct {
	reposFile      string
	repos          []string
	format         string
	fields         []string
	onlyUnresolved bool
	concurrency    int
	timeout        time.Duration
	help           bool
	version        bool
}

func newFlagSet(cfg *config) *flag.FlagSet {
	fs := flag.NewFlagSet("pr-report", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	fs.Func("repos", "Read repositories from a UTF-8 text `file` (may be combined with --repo)", func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("repository file must not be empty")
		}
		cfg.reposFile = value
		return nil
	})
	fs.Func("repo", "Include `owner/name` (repeatable; may be combined with --repos)", func(value string) error {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("repository must not be empty")
		}
		cfg.repos = append(cfg.repos, value)
		return nil
	})
	fs.StringVar(&cfg.format, "format", "table", "Output `format`: table or json")
	fs.Func("fields", "Comma-separated pull request `fields` in display order (see list below)", func(value string) error {
		if cfg.fields != nil {
			return fmt.Errorf("--fields may only be specified once")
		}
		fields, err := parseFields(value)
		if err != nil {
			return err
		}
		cfg.fields = fields
		return nil
	})
	fs.BoolVar(&cfg.onlyUnresolved, "only-unresolved", false, "Show only PRs with confirmed unresolved threads (default false)")
	fs.IntVar(&cfg.concurrency, "concurrency", 1, "Global gh subprocess limit, from 1 to 8")
	fs.DurationVar(&cfg.timeout, "timeout", 30*time.Second, "Positive request timeout per attempt")
	fs.BoolVar(&cfg.help, "help", false, "Show help without requiring gh or repositories")
	fs.BoolVar(&cfg.help, "h", false, "Alias for --help")
	fs.BoolVar(&cfg.version, "version", false, "Show version without requiring gh or repositories")
	return fs
}

func parseConfig(args []string) (config, error) {
	var cfg config
	fs := newFlagSet(&cfg)
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() != 0 {
		return config{}, fmt.Errorf("positional arguments are not supported: %q", fs.Arg(0))
	}
	if cfg.format != "table" && cfg.format != "json" {
		return config{}, fmt.Errorf("--format must be table or json")
	}
	if cfg.concurrency < 1 || cfg.concurrency > 8 {
		return config{}, fmt.Errorf("--concurrency must be between 1 and 8")
	}
	if cfg.timeout <= 0 {
		return config{}, fmt.Errorf("--timeout must be positive")
	}
	if !cfg.help && !cfg.version && cfg.reposFile == "" && len(cfg.repos) == 0 {
		return config{}, fmt.Errorf("at least one repository source is required: use --repos or --repo")
	}
	return cfg, nil
}

func writeHelp(w io.Writer) error {
	var help strings.Builder
	help.WriteString("Usage: pr-report [flags]\n\nReport open GitHub pull requests, including drafts.\n")
	help.WriteString("Supply --repos <file>, repeated --repo <owner/name>, or both.\n")
	help.WriteString("Flags accept one or two leading dashes; positional arguments are not supported.\n\nFlags:\n")
	fs := newFlagSet(&config{})
	fs.SetOutput(&help)
	fs.PrintDefaults()
	help.WriteString("\nAvailable --fields names: " + fieldNames() + "\n")
	help.WriteString("--fields selects PR columns/properties in the given order; report metadata and diagnostics remain.\n")
	help.WriteString("\nExamples:\n  pr-report --repo my-org/backend --repo my-org/frontend\n  pr-report --repo my-org/backend --fields repo,number,title,url\n  pr-report --repos repos.txt --format json --fields repo,number,title --concurrency 3 --timeout 45s\n")
	help.WriteString("\nRepository files use UTF-8 (optional initial BOM), LF or CRLF, and one owner/name per line.\n")
	help.WriteString("Blank lines and full-line # comments are ignored; surrounding whitespace is trimmed.\n")
	help.WriteString("Duplicates are combined case-insensitively. URLs, paths, .git suffixes, and trailing comments are rejected.\n")
	help.WriteString("\nComment counts exclude text present only in a review body.\nReview thread counts are not counts of individual inline messages.\n")
	_, err := io.WriteString(w, help.String())
	return err
}
