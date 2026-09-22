package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseConfigValid(t *testing.T) {
	defaults := config{format: "table", concurrency: 1, timeout: 30 * time.Second}
	tests := []struct {
		name string
		args []string
		want config
	}{
		{"direct defaults", []string{"--repo", "org/repo"}, withConfig(defaults, func(c *config) { c.repos = []string{"org/repo"} })},
		{"file defaults", []string{"--repos", "repos.txt"}, withConfig(defaults, func(c *config) { c.reposFile = "repos.txt" })},
		{"combined sources and flags", []string{"--repos", "repos.txt", "--repo", "org/one", "--repo=org/two", "--format=json", "--only-unresolved", "--concurrency", "8", "--timeout", "1m30s"}, config{
			reposFile: "repos.txt", repos: []string{"org/one", "org/two"}, format: "json", onlyUnresolved: true, concurrency: 8, timeout: 90 * time.Second,
		}},
		{"explicit defaults", []string{"--repo=org/repo", "--format=table", "--only-unresolved=false", "--concurrency=1", "--timeout=30s"}, withConfig(defaults, func(c *config) { c.repos = []string{"org/repo"} })},
		{"fractional duration", []string{"--repo", "org/repo", "--timeout", "1.5s", "--concurrency", "3"}, withConfig(defaults, func(c *config) {
			c.repos = []string{"org/repo"}
			c.timeout = 1500 * time.Millisecond
			c.concurrency = 3
		})},
		{"small positive duration", []string{"--repo", "org/repo", "--timeout=1ns"}, withConfig(defaults, func(c *config) { c.repos = []string{"org/repo"}; c.timeout = time.Nanosecond })},
		{"help without source", []string{"--help"}, withConfig(defaults, func(c *config) { c.help = true })},
		{"help alias", []string{"-h"}, withConfig(defaults, func(c *config) { c.help = true })},
		{"version without source", []string{"--version"}, withConfig(defaults, func(c *config) { c.version = true })},
		{"both informational flags", []string{"--version", "--help"}, withConfig(defaults, func(c *config) { c.help = true; c.version = true })},
		{"single dash flags", []string{"-repo", "org/repo", "-format", "json"}, withConfig(defaults, func(c *config) { c.repos = []string{"org/repo"}; c.format = "json" })},
		{"empty argument terminator", []string{"--repo", "org/repo", "--"}, withConfig(defaults, func(c *config) { c.repos = []string{"org/repo"} })},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseConfig(tt.args)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("config = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func withConfig(base config, set func(*config)) config {
	set(&base)
	return base
}

func TestParseConfigDoesNotRetainPreviousSources(t *testing.T) {
	if _, err := parseConfig([]string{"--repo", "org/one"}); err != nil {
		t.Fatal(err)
	}
	if _, err := parseConfig(nil); err == nil {
		t.Fatal("second parse inherited a repository from the first parse")
	}
}
