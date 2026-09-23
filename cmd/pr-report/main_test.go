package main

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInvalidInputNeverExecutes(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		message string
	}{
		{"missing source", nil, "repository source"},
		{"flags without source", []string{"--format=json"}, "repository source"},
		{"empty file", []string{"--repos="}, "file must not be empty"},
		{"blank file", []string{"--repos", " \t"}, "file must not be empty"},
		{"empty repo", []string{"--repo="}, "repository must not be empty"},
		{"blank repo", []string{"--repo", " \t"}, "repository must not be empty"},
		{"empty repeated repo", []string{"--repo=org/repo", "--repo="}, "repository must not be empty"},
		{"positional only", []string{"org/repo"}, "positional"},
		{"positional after flags", []string{"--repo=org/repo", "extra"}, "positional"},
		{"flags after positional", []string{"extra", "--repo=org/repo"}, "positional"},
		{"positional after terminator", []string{"--repo=org/repo", "--", "extra"}, "positional"},
		{"bare dash", []string{"--repo=org/repo", "-"}, "positional"},
		{"unknown flag", []string{"--repo=org/repo", "--unknown"}, "flag provided but not defined"},
		{"malformed flag", []string{"---repo=org/repo"}, "bad flag syntax"},
		{"missing repo value", []string{"--repo"}, "flag needs an argument"},
		{"missing file value", []string{"--repos"}, "flag needs an argument"},
		{"missing format value", []string{"--format"}, "flag needs an argument"},
		{"missing concurrency value", []string{"--concurrency"}, "flag needs an argument"},
		{"missing timeout value", []string{"--timeout"}, "flag needs an argument"},
		{"unknown format", []string{"--repo=org/repo", "--format=yaml"}, "--format"},
		{"uppercase format", []string{"--repo=org/repo", "--format=JSON"}, "--format"},
		{"empty format", []string{"--repo=org/repo", "--format="}, "--format"},
		{"concurrency zero", []string{"--repo=org/repo", "--concurrency=0"}, "between 1 and 8"},
		{"concurrency negative", []string{"--repo=org/repo", "--concurrency=-1"}, "between 1 and 8"},
		{"concurrency too large", []string{"--repo=org/repo", "--concurrency=9"}, "between 1 and 8"},
		{"concurrency fraction", []string{"--repo=org/repo", "--concurrency=1.5"}, "invalid value"},
		{"concurrency text", []string{"--repo=org/repo", "--concurrency=many"}, "invalid value"},
		{"concurrency overflow", []string{"--repo=org/repo", "--concurrency=999999999999999999999999999"}, "invalid value"},
		{"timeout zero", []string{"--repo=org/repo", "--timeout=0s"}, "must be positive"},
		{"timeout negative", []string{"--repo=org/repo", "--timeout=-1s"}, "must be positive"},
		{"timeout missing unit", []string{"--repo=org/repo", "--timeout=30"}, "invalid value"},
		{"timeout text", []string{"--repo=org/repo", "--timeout=later"}, "invalid value"},
		{"timeout overflow", []string{"--repo=org/repo", "--timeout=999999999999999999h"}, "invalid value"},
		{"invalid filter boolean", []string{"--repo=org/repo", "--only-unresolved=maybe"}, "invalid boolean"},
		{"separate filter boolean", []string{"--repo=org/repo", "--only-unresolved", "false"}, "positional"},
		{"invalid help boolean", []string{"--help=maybe"}, "invalid boolean"},
		{"invalid version boolean", []string{"--version=maybe"}, "invalid boolean"},
		{"help false needs source", []string{"--help=false"}, "repository source"},
		{"version false needs source", []string{"--version=false"}, "repository source"},
		{"help with invalid range", []string{"--help", "--concurrency=0"}, "between 1 and 8"},
		{"help with unknown flag", []string{"--help", "--unknown"}, "flag provided but not defined"},
		{"unknown flag before help", []string{"--unknown", "--help"}, "flag provided but not defined"},
		{"help with positional", []string{"--help", "extra"}, "positional"},
		{"version with invalid duration", []string{"--version", "--timeout=0s"}, "must be positive"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			calls := 0
			execute := func(config, io.Writer, io.Writer) int { calls++; return 0 }
			if code := run(tt.args, &stdout, &stderr, execute); code != 2 {
				t.Errorf("exit = %d, want 2", code)
			}
			if calls != 0 {
				t.Errorf("executor called %d times for invalid input", calls)
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), tt.message) {
				t.Errorf("stdout = %q, stderr = %q; want only a diagnostic containing %q", stdout.String(), stderr.String(), tt.message)
			}
		})
	}
}

func TestInformationalFlagsWithoutGH(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, args := range [][]string{{"--help"}, {"-h"}, {"--version"}, {"--version", "--help"}, {"--help", "--repos", "does-not-exist.txt"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			calls := 0
			code := run(args, &stdout, &stderr, func(config, io.Writer, io.Writer) int { calls++; return 1 })
			if code != 0 || calls != 0 || stderr.Len() != 0 {
				t.Fatalf("exit = %d, executor calls = %d, stderr = %q", code, calls, stderr.String())
			}
			if len(args) == 1 && args[0] == "--version" {
				if want := "pr-report " + version + "\n"; stdout.String() != want {
					t.Errorf("version = %q, want %q", stdout.String(), want)
				}
				return
			}
			for _, text := range []string{"Usage: pr-report", "-repos", "-repo", "-format", "-fields", "-only-unresolved", "-concurrency", "-timeout", "-help", "-version", "default \"table\"", "default false", "default 1", "default 30s", "review body", "inline messages"} {
				if !strings.Contains(stdout.String(), text) {
					t.Errorf("help missing %q", text)
				}
			}
		})
	}
}

func TestValidConfigReachesExecutor(t *testing.T) {
	var stdout, stderr bytes.Buffer
	calls := 0
	file := writeRepositoryFile(t, "org/one\n")
	args := []string{"--repos=" + file, "--repo=org/one", "--repo=org/two", "--format=json", "--only-unresolved", "--concurrency=8", "--timeout=45s"}
	want := config{reposFile: file, repos: []string{"org/one", "org/two"}, format: "json", onlyUnresolved: true, concurrency: 8, timeout: 45 * time.Second}
	code := run(args, &stdout, &stderr, func(got config, out, diagnostics io.Writer) int {
		calls++
		if !reflect.DeepEqual(got, want) {
			t.Errorf("executor config = %#v, want %#v", got, want)
		}
		if out != &stdout || diagnostics != &stderr {
			t.Error("executor did not receive the command's output streams")
		}
		return 3
	})
	if code != 3 || calls != 1 {
		t.Errorf("exit = %d, calls = %d; want 3 and 1", code, calls)
	}
}

func TestInformationalOutputFailure(t *testing.T) {
	for _, arg := range []string{"--help", "--version"} {
		t.Run(arg, func(t *testing.T) {
			var stderr bytes.Buffer
			code := run([]string{arg}, failingWriter{}, &stderr, func(config, io.Writer, io.Writer) int {
				t.Fatal("executor called for informational output")
				return 0
			})
			if code != 1 || !strings.Contains(stderr.String(), "write output") {
				t.Errorf("exit = %d, stderr = %q", code, stderr.String())
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("output unavailable")
}
