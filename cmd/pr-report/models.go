package main

import (
	"bytes"
	"encoding/json"
	"time"
)

// apiField preserves absence, explicit null, and zero values independently.
// These transport-only types never become the public JSON report.
type apiField[T any] struct {
	Value         T
	Present, Null bool
}

func (f *apiField[T]) UnmarshalJSON(b []byte) error {
	*f = apiField[T]{Present: true, Null: bytes.Equal(bytes.TrimSpace(b), []byte("null"))}
	if f.Null {
		return nil
	}
	return json.Unmarshal(b, &f.Value)
}
func (f apiField[T]) known() bool { return f.Present && !f.Null }
func (f apiField[T]) pointer() *T {
	if !f.known() {
		return nil
	}
	v := f.Value
	return &v
}
func ptr[T any](v T) *T { return &v }

type apiEnvelope struct {
	Data   apiField[apiData]           `json:"data"`
	Errors apiField[[]json.RawMessage] `json:"errors"`
}
type apiData struct {
	Repository apiField[apiRepository] `json:"repository"`
}
type apiRepository struct {
	Name apiField[string]               `json:"nameWithOwner"`
	PRs  apiField[apiConnection[apiPR]] `json:"pullRequests"`
	PR   apiField[apiThreadPR]          `json:"pullRequest"`
}
type apiConnection[T any] struct {
	Nodes    apiField[[]apiField[T]] `json:"nodes"`
	PageInfo apiField[apiPageInfo]   `json:"pageInfo"`
}
type apiPageInfo struct {
	HasNext apiField[bool]   `json:"hasNextPage"`
	Cursor  apiField[string] `json:"endCursor"`
}
type apiCount struct {
	Total apiField[int] `json:"totalCount"`
}
type apiAuthor struct {
	Login apiField[string] `json:"login"`
}
type apiPR struct {
	ID         apiField[string]    `json:"id"`
	Number     apiField[int]       `json:"number"`
	Title      apiField[string]    `json:"title"`
	URL        apiField[string]    `json:"url"`
	Author     apiField[apiAuthor] `json:"author"`
	Draft      apiField[bool]      `json:"isDraft"`
	Created    apiField[string]    `json:"createdAt"`
	Updated    apiField[string]    `json:"updatedAt"`
	Comments   apiField[apiCount]  `json:"comments"`
	Threads    apiField[apiCount]  `json:"reviewThreads"`
	Review     apiField[string]    `json:"reviewDecision"`
	Mergeable  apiField[string]    `json:"mergeable"`
	MergeState apiField[string]    `json:"mergeStateStatus"`
}
type apiThreadPR struct {
	Threads apiField[apiConnection[apiThread]] `json:"reviewThreads"`
}
type apiThread struct {
	ID       apiField[string] `json:"id"`
	Resolved apiField[bool]   `json:"isResolved"`
}

// list always emits an array, even for its zero value.
type list[T any] []T

func (v list[T]) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]T(v))
}

type report struct {
	SchemaVersion int                    `json:"schema_version"`
	StartedAt     time.Time              `json:"started_at"`
	FinishedAt    time.Time              `json:"finished_at"`
	Complete      bool                   `json:"complete"`
	Filters       reportFilters          `json:"filters"`
	Repositories  list[repositoryResult] `json:"repositories"`
	PRsCollected  int                    `json:"prs_collected"`
	PRsReturned   int                    `json:"prs_returned"`
	PullRequests  list[pullRequest]      `json:"pull_requests"`
	Errors        list[diagnostic]       `json:"errors"`
	Warnings      list[diagnostic]       `json:"warnings"`
}
type reportFilters struct {
	OnlyUnresolved bool `json:"only_unresolved"`
}
type repositoryResult struct {
	RequestedRepo   string  `json:"requested_repo"`
	Repo            *string `json:"repo"`
	ListingComplete bool    `json:"listing_complete"`
	DetailsComplete bool    `json:"details_complete"`
	PRsCollected    int     `json:"prs_collected"`
}
type pullRequest struct {
	Repo                      string       `json:"repo"`
	Number                    int          `json:"number"`
	Title                     string       `json:"title"`
	URL                       string       `json:"url"`
	Author                    *string      `json:"author"`
	IsDraft                   bool         `json:"is_draft"`
	CreatedAt                 string       `json:"created_at"`
	UpdatedAt                 string       `json:"updated_at"`
	ObservedAt                time.Time    `json:"observed_at"`
	ConversationCommentsCount *int         `json:"conversation_comments_count"`
	ReviewThreadsCount        *int         `json:"review_threads_count"`
	UnresolvedThreadsCount    *int         `json:"unresolved_threads_count"`
	HasComments               *bool        `json:"has_comments"`
	ReviewDecision            *string      `json:"review_decision"`
	Mergeable                 *string      `json:"mergeable"`
	MergeStateStatus          *string      `json:"merge_state_status"`
	Blocked                   *bool        `json:"blocked"`
	Signals                   list[string] `json:"signals"`
	DetailsComplete           bool         `json:"details_complete"`
}
type diagnostic struct {
	Repo     *string `json:"repo"`
	PRNumber *int    `json:"pr_number"`
	Stage    string  `json:"stage"`
	Code     string  `json:"code"`
	Message  string  `json:"message"`
}

func newReport(start time.Time) report { return report{SchemaVersion: 1, StartedAt: start.UTC()} }
