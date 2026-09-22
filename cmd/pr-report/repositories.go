package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// loadRepositories validates every source before returning a unique list.
// File entries precede flags; the first spelling is retained for requested_repo.
func loadRepositories(file string, direct []string) ([]string, error) {
	var repos []string
	if file != "" {
		f, err := os.Open(file)
		if err != nil {
			return nil, fmt.Errorf("read repository file %q: %w", file, err)
		}
		defer f.Close()
		repos, err = parseRepositoryFile(f, file)
		if err != nil {
			return nil, err
		}
	}
	for i, value := range direct {
		repo, err := normalizeRepository(value)
		if err != nil {
			return nil, fmt.Errorf("--repo occurrence %d: invalid repository %q: %w", i+1, value, err)
		}
		repos = append(repos, repo)
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("repository list is empty; provide at least one owner/name")
	}

	unique := make([]string, 0, len(repos))
	seen := make(map[string]bool, len(repos))
	for _, repo := range repos {
		key := strings.ToLower(repo)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, repo)
		}
	}
	return unique, nil
}

func parseRepositoryFile(r io.Reader, source string) ([]string, error) {
	var repos []string
	reader := bufio.NewReader(r)
	for lineNumber := 1; ; lineNumber++ {
		// ReadString avoids Scanner's default 64 KiB line limit, including for comments.
		line, err := reader.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("read repository file %q at line %d: %w", source, lineNumber, err)
		}
		if !utf8.ValidString(line) {
			return nil, fmt.Errorf("%s:%d: invalid UTF-8 in %q", source, lineNumber, line)
		}
		if lineNumber == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
		}
		value := strings.TrimSpace(line)
		if value != "" && !strings.HasPrefix(value, "#") {
			repo, parseErr := normalizeRepository(value)
			if parseErr != nil {
				return nil, fmt.Errorf("%s:%d: invalid repository %q: %w", source, lineNumber, value, parseErr)
			}
			repos = append(repos, repo)
		}
		if errors.Is(err, io.EOF) {
			return repos, nil
		}
	}
}

func normalizeRepository(value string) (string, error) {
	if !utf8.ValidString(value) {
		return "", fmt.Errorf("expected UTF-8")
	}
	repo := strings.TrimSpace(value)
	owner, name, found := strings.Cut(repo, "/")
	if !found || !validRepositorySegment(owner, false) || !validRepositorySegment(name, true) || name == "." || name == ".." {
		return "", fmt.Errorf("expected owner/name using ASCII letters, digits and hyphens; name may also contain dots and underscores")
	}
	if strings.HasSuffix(strings.ToLower(name), ".git") {
		return "", fmt.Errorf(".git suffixes are not supported")
	}
	return repo, nil
}

func validRepositorySegment(value string, isName bool) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' {
			continue
		}
		if isName && (c == '.' || c == '_') {
			continue
		}
		return false
	}
	return true
}
