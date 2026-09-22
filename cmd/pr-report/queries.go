package main

import "strings"

const listQuery = `query($owner: String!, $name: String!, $pageSize: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    nameWithOwner
    pullRequests(states: OPEN, orderBy: {field: CREATED_AT, direction: ASC}, first: $pageSize, after: $cursor) {
      nodes {
        id number title url author { login } isDraft createdAt updatedAt
        comments { totalCount }
        reviewThreads { totalCount }
        reviewDecision mergeable mergeStateStatus
      }
      pageInfo { hasNextPage endCursor }
    }
  }
}`

func queryArgs(query, repo, cursor string) []string {
	owner, name, _ := strings.Cut(repo, "/")
	args := []string{"-f", "query=" + query, "-f", "owner=" + owner, "-f", "name=" + name, "-F", "pageSize=100"}
	if cursor != "" {
		args = append(args, "-f", "cursor="+cursor)
	}
	return args
}

const threadQuery = `query($owner: String!, $name: String!, $number: Int!, $pageSize: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: $pageSize, after: $cursor) {
        nodes { id isResolved }
        pageInfo { hasNextPage endCursor }
      }
    }
  }
}`
