package main

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeRepository(t *testing.T) {
	for _, value := range []string{"Org-12/Repo_34.name-5", "a/.github", "a/a..b", "a/repo.git-tools", "-owner/repo", "123/456"} {
		t.Run(value, func(t *testing.T) {
			got, err := normalizeRepository(" \t\u2003" + value + "\r\n")
			if err != nil || got != value {
				t.Fatalf("normalize = %q, %v; want %q", got, err, value)
			}
		})
	}
}

func TestNormalizeRepositoryRejectsInvalidValues(t *testing.T) {
	values := []string{
		"", " \t", "owner", "owner/", "/repo", "owner/repo/", "owner/repo/extra",
		"https://github.com/owner/repo", "ssh://git@github.com/owner/repo", "git@github.com:owner/repo",
		"file:///owner/repo", "/owner/repo", "./repo", "../repo", "~/repo", `C:\owner\repo`, `owner\repo`,
		"owner/repo.git", "owner/repo.GIT", "owner/repo.Git", "owner/.git",
		"./name", "../name", "owner/.", "owner/..", "owner/../repo",
		"owner_name/repo", "owner.name/repo", "ownér/repo", "owner/répo", "owner/仓库",
		"owner/repo # comment", "owner/repo#comment", "# owner/repo", "owner/repo?query", "owner/repo@ref",
		"own er/repo", "owner/re po", "owner/\trepo", "owner/repo\nother/repo", "owner/re\x00po",
		"owner/re\xffpo", "\uFEFFowner/repo", "owner/\uFEFFrepo",
	}
	for _, value := range values {
		t.Run(value, func(t *testing.T) {
			if got, err := normalizeRepository(value); err == nil || got != "" {
				t.Fatalf("normalize(%q) = %q, %v; want rejection", value, got, err)
			}
		})
	}
}

func TestParseRepositoryFile(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"LF", "Org/one\norg/two\n", []string{"Org/one", "org/two"}},
		{"BOM CRLF whitespace comments", "\uFEFF  # Serviços\r\n \t\r\n  Org/one \t\r\n\t# ignored\r\norg/two\r\n", []string{"Org/one", "org/two"}},
		{"BOM before first repo", "\uFEFFOrg/one\n", []string{"Org/one"}},
		{"mixed endings and no final newline", "Org/one\r\norg/two\norg/three", []string{"Org/one", "org/two", "org/three"}},
		{"Unicode whitespace", "\u2003Org/one\u00a0\n", []string{"Org/one"}},
		{"empty", "", nil},
		{"BOM only", "\uFEFF", nil},
		{"comments only", "\n  # comentário\n\t#another\r\n \t", nil},
		{"long comment", "#" + strings.Repeat("a", 70000) + "\norg/repo", []string{"org/repo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRepositoryFile(strings.NewReader(tt.input), "repos.txt")
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parse = %#v, %v; want %#v", got, err, tt.want)
			}
		})
	}
}

func TestParseRepositoryFileErrorContext(t *testing.T) {
	tests := []struct {
		name  string
		input string
		line  string
		value string
	}{
		{"invalid after valid", "org/valid\r\n\r\n # comment\r\n https://github.com/org/repo \r\n", ":4:", "https://github.com/org/repo"},
		{"trailing comment", "org/repo # comment", ":1:", "org/repo # comment"},
		{"later BOM", "org/one\n\uFEFForg/two", ":2:", `\ufefforg/two`},
		{"double BOM", "\uFEFF\uFEFForg/repo", ":1:", `\ufefforg/repo`},
		{"BOM after whitespace", " \uFEFForg/repo", ":1:", `\ufefforg/repo`},
		{"invalid UTF-8 repo", "org/valid\norg/re\xffpo", ":2:", `org/re\xffpo`},
		{"invalid UTF-8 comment", "# bad \xff\norg/valid", ":1:", "invalid UTF-8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRepositoryFile(strings.NewReader(tt.input), "input.txt")
			if err == nil || got != nil {
				t.Fatalf("parse = %#v, %v; want error without partial results", got, err)
			}
			for _, part := range []string{"input.txt" + tt.line, tt.value} {
				if !strings.Contains(err.Error(), part) {
					t.Errorf("error %q lacks %q", err, part)
				}
			}
		})
	}
}

func TestParseRepositoryFileReadFailure(t *testing.T) {
	reader := io.MultiReader(strings.NewReader("org/valid\norg/partial"), brokenRepositoryReader{})
	got, err := parseRepositoryFile(reader, "broken.txt")
	if got != nil || !errors.Is(err, io.ErrUnexpectedEOF) || !strings.Contains(err.Error(), "broken.txt") || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("parse = %#v, %v; want contextual read error without partial results", got, err)
	}
}

type brokenRepositoryReader struct{}

func (brokenRepositoryReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestLoadRepositories(t *testing.T) {
	tests := []struct {
		name         string
		fileContents string
		direct       []string
		want         []string
	}{
		{"file dedup", "\uFEFF # list\r\n Team/Repo \r\nteam/repo\r\nTEAM/REPO\r\nTeam/Other\r\n", nil, []string{"Team/Repo", "Team/Other"}},
		{"combined dedup", " Team/Repo\nTeam/File\n", []string{"team/repo", " Team/Flag ", "TEAM/FLAG", "team/file"}, []string{"Team/Repo", "Team/File", "Team/Flag"}},
		{"empty file with flags", "", []string{"Team/Repo", "team/repo"}, []string{"Team/Repo"}},
		{"comments file with flags", "# comment\n", []string{"Team/Repo"}, []string{"Team/Repo"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := writeRepositoryFile(t, tt.fileContents)
			got, err := loadRepositories(file, tt.direct)
			if err != nil || !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("load = %#v, %v; want %#v", got, err, tt.want)
			}
		})
	}
	got, err := loadRepositories("", []string{" Org/Repo ", "org/repo", "Org/Other"})
	if err != nil || !reflect.DeepEqual(got, []string{"Org/Repo", "Org/Other"}) {
		t.Fatalf("direct load = %#v, %v", got, err)
	}
}

func TestLoadRepositoriesFailures(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	if got, err := loadRepositories(missing, []string{"org/repo"}); got != nil || !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), missing) {
		t.Fatalf("missing file = %#v, %v", got, err)
	}
	for _, contents := range []string{"", "\uFEFF", " \r\n# comment\r\n\t"} {
		if got, err := loadRepositories(writeRepositoryFile(t, contents), nil); got != nil || err == nil || !strings.Contains(err.Error(), "list is empty") {
			t.Fatalf("empty file = %#v, %v", got, err)
		}
	}
	if got, err := loadRepositories("", nil); got != nil || err == nil {
		t.Fatalf("empty sources = %#v, %v", got, err)
	}
	if got, err := loadRepositories("", []string{"org/repo", "# comment"}); got != nil || err == nil || !strings.Contains(err.Error(), "occurrence 2") {
		t.Fatalf("invalid flag = %#v, %v", got, err)
	}
}

func TestRepositoryInputValidatedBeforeExecution(t *testing.T) {
	mixed := writeRepositoryFile(t, "\uFEFF# Serviços\r\n Team/Repo \r\nteam/repo\r\nTeam/Other\r\n")
	var stdout, stderr bytes.Buffer
	calls := 0
	code := run([]string{"--repo=team/repo", "--repos=" + mixed, "--repo= Team/Third "}, &stdout, &stderr, func(cfg config, _, _ io.Writer) int {
		calls++
		if want := []string{"Team/Repo", "Team/Other", "Team/Third"}; !reflect.DeepEqual(cfg.repos, want) {
			t.Errorf("executor repositories = %#v, want %#v", cfg.repos, want)
		}
		return 0
	})
	if code != 0 || calls != 1 || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("exit = %d, calls = %d, stdout = %q, stderr = %q", code, calls, stdout.String(), stderr.String())
	}

	cases := [][]string{
		{"--repo=https://github.com/org/repo"},
		{"--repo=org/valid", "--repo=org/repo.git"},
		{"--repos=" + writeRepositoryFile(t, "org/valid\norg/repo # comment")},
		{"--repos=" + writeRepositoryFile(t, "org/valid\ninvalid"), "--repo=org/valid"},
		{"--repos=" + writeRepositoryFile(t, "\uFEFF # only comments\r\n")},
		{"--repos=" + filepath.Join(t.TempDir(), "missing.txt"), "--repo=org/valid"},
		{"--repos=" + t.TempDir()},
		{"--repos=" + mixed, "--repo=org/.."},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			stdout.Reset()
			stderr.Reset()
			calls = 0
			code := run(args, &stdout, &stderr, func(config, io.Writer, io.Writer) int { calls++; return 0 })
			if code != 2 || calls != 0 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("exit = %d, calls = %d, stdout = %q, stderr = %q", code, calls, stdout.String(), stderr.String())
			}
		})
	}
}

func writeRepositoryFile(t *testing.T, contents string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "repos.txt")
	if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
		t.Fatal(err)
	}
	return file
}
