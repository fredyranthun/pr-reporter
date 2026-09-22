package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type requestFailure struct {
	code, message string
	transient     bool
}

func (e *requestFailure) Error() string { return e.message }
func invalid(message string) *requestFailure {
	return &requestFailure{code: "invalid_response", message: message}
}
func (e *requestFailure) diagnostic(repo *string, number *int, stage string) diagnostic {
	return diagnostic{repo, number, stage, e.code, e.message}
}

// Classify only recognizable transport/API failures. Unclassified API errors,
// including rate limits, are not transient. Never publish raw process stderr.
func processFailure(r processResult) *requestFailure {
	if errors.Is(r.Err, context.DeadlineExceeded) {
		return &requestFailure{"timeout", "request timed out", true}
	}
	s := strings.ToLower(string(r.Stderr))
	switch {
	case strings.Contains(s, "http 401"), strings.Contains(s, "gh auth login"), strings.Contains(s, "bad credentials"):
		return &requestFailure{"auth", "GitHub authentication failed; authenticate gh manually", false}
	case strings.Contains(s, "http 403"), strings.Contains(s, "http 404"):
		return &requestFailure{"not_found_or_forbidden", "repository or pull request does not exist or is inaccessible", false}
	case strings.Contains(s, "http 500"), strings.Contains(s, "http 502"), strings.Contains(s, "http 503"), strings.Contains(s, "http 504"):
		return &requestFailure{"api", "GitHub server temporarily unavailable", true}
	case strings.Contains(s, "no such host"), strings.Contains(s, "connection refused"), strings.Contains(s, "connection reset"), strings.Contains(s, "network is unreachable"), strings.Contains(s, "tls handshake timeout"), strings.Contains(s, "unexpected eof"), strings.Contains(s, "i/o timeout"):
		return &requestFailure{"network", "network request failed", true}
	}
	return &requestFailure{"api", "GitHub API request failed", false}
}
func decodeEnvelope(r processResult) (apiRepository, *requestFailure) {
	var raw map[string]json.RawMessage
	err := json.Unmarshal(r.Stdout, &raw)
	if err != nil || raw == nil {
		if r.Err != nil || r.ExitCode != 0 {
			return apiRepository{}, processFailure(r)
		}
		return apiRepository{}, invalid("expected a JSON response object")
	}
	if b, ok := raw["errors"]; ok {
		var errs []json.RawMessage
		if err := json.Unmarshal(b, &errs); err != nil || errs == nil {
			return apiRepository{}, invalid("errors must be an array")
		}
		if len(errs) > 0 {
			// GitHub supplies structured types for inaccessible objects and auth errors.
			code := "api"
			message := "GitHub returned GraphQL errors"
			for _, b := range errs {
				var e struct {
					Type       string `json:"type"`
					Extensions struct {
						Code string `json:"code"`
					} `json:"extensions"`
				}
				_ = json.Unmarshal(b, &e)
				switch strings.ToUpper(e.Type + " " + e.Extensions.Code) {
				case "NOT_FOUND ", "FORBIDDEN ", " NOT_FOUND", " FORBIDDEN":
					code = "not_found_or_forbidden"
					message = "repository or pull request does not exist or is inaccessible"
				case "UNAUTHORIZED ", " UNAUTHENTICATED":
					code = "auth"
					message = "GitHub authentication failed; authenticate gh manually"
				}
			}
			return apiRepository{}, &requestFailure{code, message, false}
		}
	}
	if r.Err != nil || r.ExitCode != 0 {
		return apiRepository{}, processFailure(r)
	}
	var env apiEnvelope
	if err := json.Unmarshal(r.Stdout, &env); err != nil {
		return apiRepository{}, invalid("invalid API field type: " + err.Error())
	}
	if !env.Data.known() {
		return apiRepository{}, invalid("missing data object")
	}
	if env.Data.Value.Repository.Present && env.Data.Value.Repository.Null {
		return apiRepository{}, &requestFailure{"not_found_or_forbidden", "repository or pull request does not exist or is inaccessible", false}
	}
	if !env.Data.Value.Repository.known() {
		return apiRepository{}, invalid("missing repository object")
	}
	return env.Data.Value.Repository.Value, nil
}
func validateConnection[T any](f apiField[apiConnection[T]]) *requestFailure {
	if !f.known() || !f.Value.Nodes.known() || !f.Value.PageInfo.known() {
		return invalid("missing connection, nodes, or pageInfo")
	}
	p := f.Value.PageInfo.Value
	if !p.HasNext.known() {
		return invalid("missing pageInfo.hasNextPage")
	}
	if p.HasNext.Value && (!p.Cursor.known() || p.Cursor.Value == "") {
		return invalid("next page has no cursor")
	}
	for _, n := range f.Value.Nodes.Value {
		if !n.known() {
			return invalid("null connection node")
		}
	}
	return nil
}
func decodeList(r processResult) (apiRepository, *requestFailure) {
	repo, e := decodeEnvelope(r)
	if e != nil {
		return repo, e
	}
	if !repo.Name.known() {
		return repo, invalid("missing canonical repository name")
	}
	name, err := normalizeRepository(repo.Name.Value)
	if err != nil || name != repo.Name.Value {
		return repo, invalid("invalid canonical repository name")
	}
	if e = validateConnection(repo.PRs); e != nil {
		return repo, e
	}
	for _, n := range repo.PRs.Value.Nodes.Value {
		if e = validatePR(n.Value); e != nil {
			return repo, e
		}
	}
	return repo, nil
}
func validatePR(p apiPR) *requestFailure {
	if !p.ID.known() || p.ID.Value == "" || !p.Number.known() || p.Number.Value <= 0 || !p.Title.known() || !p.URL.known() || p.URL.Value == "" || !p.Draft.known() {
		return invalid("missing or invalid PR identity, title, url, or isDraft")
	}
	for _, f := range []apiField[string]{p.Created, p.Updated} {
		if !f.known() {
			return invalid("missing PR timestamp")
		}
		if _, e := time.Parse(time.RFC3339, f.Value); e != nil {
			return invalid("invalid PR timestamp")
		}
	}
	if p.Author.known() && (!p.Author.Value.Login.known() || p.Author.Value.Login.Value == "") {
		return invalid("invalid author login")
	}
	for _, f := range []apiField[apiCount]{p.Comments, p.Threads} {
		if f.known() && f.Value.Total.known() && f.Value.Total.Value < 0 {
			return invalid("negative totalCount")
		}
	}
	for _, f := range []apiField[string]{p.Review, p.Mergeable, p.MergeState} {
		if f.known() && f.Value == "" {
			return invalid("empty enum value")
		}
	}
	return nil
}
func countValue(f apiField[apiCount]) *int {
	if !f.known() {
		return nil
	}
	return f.Value.Total.pointer()
}
func mainDetails(p apiPR) []string {
	var missing []string
	if !p.Author.Present {
		missing = append(missing, "author")
	}
	if countValue(p.Comments) == nil {
		missing = append(missing, "comments.totalCount")
	}
	if countValue(p.Threads) == nil {
		missing = append(missing, "reviewThreads.totalCount")
	}
	if !p.Review.Present {
		missing = append(missing, "reviewDecision")
	}
	if !p.Mergeable.known() {
		missing = append(missing, "mergeable")
	}
	if !p.MergeState.known() {
		missing = append(missing, "mergeStateStatus")
	}
	return missing
}
func detailFailure(fields []string) *requestFailure {
	return invalid(fmt.Sprintf("unavailable PR details: %s", strings.Join(fields, ", ")))
}
