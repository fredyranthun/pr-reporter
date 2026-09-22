package main

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestListQueryArguments(t *testing.T) {
	f := &fakeExecutor{replies: []processResult{{}, {}}}
	for _, cursor := range []string{"", "opaque+/="} {
		args := queryArgs(listQuery, "Some-Org/true", cursor)
		f.Execute(context.Background(), args...)
		want := []string{"-f", "query=" + listQuery, "-f", "owner=Some-Org", "-f", "name=true", "-F", "pageSize=100"}
		if cursor != "" {
			want = append(want, "-f", "cursor="+cursor)
		}
		if !reflect.DeepEqual(args, want) {
			t.Fatal(args)
		}
	}
	if len(f.calls) != 2 {
		t.Fatal(f.calls)
	}
	for _, s := range []string{"states: OPEN", "CREATED_AT", "direction: ASC", "id number title url", "author { login }", "isDraft createdAt updatedAt", "comments { totalCount }", "reviewThreads { totalCount }", "reviewDecision mergeable mergeStateStatus", "pageInfo { hasNextPage endCursor }", "nameWithOwner"} {
		if !strings.Contains(listQuery, s) {
			t.Fatal("missing", s)
		}
	}
	for _, s := range []string{"body", "reviews {", "--paginate", "Some-Org"} {
		if strings.Contains(listQuery, s) {
			t.Fatal("unexpected", s)
		}
	}
}
