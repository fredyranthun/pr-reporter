package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func prFixture(n int) map[string]any {
	return map[string]any{"id": fmt.Sprint("PR", n), "number": n, "title": "Title", "url": fmt.Sprintf("https://github.com/Org/Repo/pull/%d", n), "author": nil, "isDraft": false, "createdAt": "2026-01-01T00:00:00Z", "updatedAt": "2026-01-01T00:00:00Z", "comments": map[string]any{"totalCount": 0}, "reviewThreads": map[string]any{"totalCount": 0}, "reviewDecision": nil, "mergeable": "MERGEABLE", "mergeStateStatus": "CLEAN"}
}
func response(v any) processResult {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return processResult{Stdout: b}
}
func listFixture(nodes []any, next bool, cursor any) processResult {
	return response(map[string]any{"data": map[string]any{"repository": map[string]any{"nameWithOwner": "Org/Repo", "pullRequests": map[string]any{"nodes": nodes, "pageInfo": map[string]any{"hasNextPage": next, "endCursor": cursor}}}}})
}
func TestDecodeRequiredPRFields(t *testing.T) {
	for _, field := range []string{"id", "number", "title", "url", "isDraft", "createdAt", "updatedAt"} {
		for _, null := range []bool{false, true} {
			t.Run(fmt.Sprint(field, null), func(t *testing.T) {
				p := prFixture(1)
				if null {
					p[field] = nil
				} else {
					delete(p, field)
				}
				if _, e := decodeList(listFixture([]any{prFixture(2), p}, false, nil)); e == nil || e.code != "invalid_response" {
					t.Fatal(e)
				}
			})
		}
	}
	bad := map[string][]any{"id": {""}, "number": {0, -1, 1.5}, "url": {""}, "isDraft": {"false"}, "createdAt": {"yesterday"}, "updatedAt": {"bad"}, "author": {map[string]any{}, map[string]any{"login": nil}, map[string]any{"login": ""}, "bob"}, "comments": {map[string]any{"totalCount": -1}, map[string]any{"totalCount": 0.5}}, "reviewThreads": {map[string]any{"totalCount": "0"}}, "reviewDecision": {"", false}, "mergeable": {""}, "mergeStateStatus": {true}}
	for k, values := range bad {
		for _, v := range values {
			p := prFixture(1)
			p[k] = v
			if _, e := decodeList(listFixture([]any{p}, false, nil)); e == nil {
				t.Errorf("accepted %s=%v", k, v)
			}
		}
	}
}
func TestDecodeNullableDetails(t *testing.T) {
	for _, k := range []string{"author", "comments", "reviewThreads", "reviewDecision", "mergeable", "mergeStateStatus"} {
		for _, null := range []bool{false, true} {
			p := prFixture(1)
			if null {
				p[k] = nil
			} else {
				delete(p, k)
			}
			r, e := decodeList(listFixture([]any{p}, false, nil))
			if e != nil {
				t.Fatal(e)
			}
			missing := mainDetails(r.PRs.Value.Nodes.Value[0].Value)
			legitimate := null && (k == "author" || k == "reviewDecision")
			if (len(missing) == 0) != legitimate {
				t.Fatalf("%s null=%v: %v", k, null, missing)
			}
		}
	}
	for _, k := range []string{"comments", "reviewThreads"} {
		for _, v := range []any{map[string]any{}, map[string]any{"totalCount": nil}} {
			p := prFixture(1)
			p[k] = v
			r, e := decodeList(listFixture([]any{p}, false, nil))
			if e != nil || len(mainDetails(r.PRs.Value.Nodes.Value[0].Value)) != 1 {
				t.Fatal(e)
			}
		}
	}
	r, e := decodeList(listFixture([]any{prFixture(1)}, false, nil))
	if e != nil || len(mainDetails(r.PRs.Value.Nodes.Value[0].Value)) != 0 {
		t.Fatal(e)
	}
}
func TestDecodeEnvelopes(t *testing.T) {
	good := listFixture([]any{}, false, nil)
	for _, raw := range []string{`null`, `[]`, `{`, `{}`, `{"data":null}`, `{"data":{}}`, `{"data":{"repository":{}}}`, `{"errors":null}`, `{"errors":{}}`, `{"errors":"bad"}`} {
		if _, e := decodeList(processResult{Stdout: []byte(raw)}); e == nil {
			t.Fatal(raw)
		}
	}
	for _, errs := range []any{[]any{}, []any{map[string]any{"type": "NOT_FOUND"}}, []any{map[string]any{"message": "rate limit"}}} {
		var v map[string]any
		_ = json.Unmarshal(good.Stdout, &v)
		v["errors"] = errs
		_, e := decodeList(response(v))
		if (e == nil) != (len(errs.([]any)) == 0) {
			t.Fatal(e)
		}
	}
	r := good
	r.Err = errors.New("exit status 1")
	r.ExitCode = 1
	if _, e := decodeList(r); e == nil {
		t.Fatal("accepted failed process")
	}
	r.Stdout = []byte(`{"data":null,"errors":[{"type":"NOT_FOUND"}]}`)
	if _, e := decodeList(r); e == nil || e.code != "not_found_or_forbidden" {
		t.Fatal(e)
	}
	if _, e := decodeList(processResult{Stdout: []byte(`{"data":{"repository":null}}`)}); e == nil || e.code != "not_found_or_forbidden" {
		t.Fatal(e)
	}
}
func TestDecodeConnectionStructure(t *testing.T) {
	for _, path := range [][]string{{"data"}, {"data", "repository", "nameWithOwner"}, {"data", "repository", "pullRequests"}, {"data", "repository", "pullRequests", "nodes"}, {"data", "repository", "pullRequests", "pageInfo"}, {"data", "repository", "pullRequests", "pageInfo", "hasNextPage"}} {
		for _, null := range []bool{false, true} {
			r := listFixture([]any{}, false, nil)
			var v map[string]any
			_ = json.Unmarshal(r.Stdout, &v)
			m := v
			for _, k := range path[:len(path)-1] {
				m = m[k].(map[string]any)
			}
			k := path[len(path)-1]
			if null {
				m[k] = nil
			} else {
				delete(m, k)
			}
			if _, e := decodeList(response(v)); e == nil {
				t.Fatal(path, null)
			}
		}
	}
	for _, cursor := range []any{nil, ""} {
		if _, e := decodeList(listFixture([]any{}, true, cursor)); e == nil {
			t.Fatal("missing cursor")
		}
	}
	if _, e := decodeList(listFixture([]any{nil}, false, nil)); e == nil {
		t.Fatal("null node")
	}
}
