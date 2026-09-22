# pr-report implementation tasks

Based on [pr-report-spec.md](pr-report-spec.md), version 1.2, dated 2026-09-22.

**32 tasks to implement, validate, document, and publish the MVP. 5 additional tasks are deferred until after the MVP.** Checked tasks record completed work; unchecked tasks remain planned. Tasks are reviewable work units, not equal-duration estimates. Mark a task complete when its implementation and applicable validation are finished.

The executable is `pr-report`, the command lives in `cmd/pr-report` with `package main`, and the module is `github.com/fredyranthun/pr-reporter`. Use the standard library, target Linux initially, and keep the code portable. Runtime authentication belongs to `gh`; collection is read-only and restricted to `github.com`.

## Phase 1 — Contracts and foundation (5 tasks)

- [x] **T01 — Resolve remaining contract edge cases.** Document which absent required fields invalidate a page versus leave a nullable field incomplete; which merge fields are required for classification; precedence of unfamiliar enums and otherwise decisive merge signals; which observation wins for duplicate IDs; and how thread counts observed at different times are represented. Define repository `details_complete` when listing fails before any PR is collected. Keep these decisions consistent with the spec and add corresponding fixtures as those features are implemented. References: §§4–8.

  Completed 2026-09-22: [normative contract decisions and fixture coverage matrix](docs/contract-edge-cases.md), synchronized with spec 1.2. Reviewed all six T01 decisions against §§4–9. Fixture implementation is assigned to the corresponding feature tasks; no Go implementation is claimed.

- [x] **T02 — Set up the repository and Go command.** Inspect any existing remote history before initializing or connecting Git to `https://github.com/fredyranthun/pr-reporter.git`. Create `go.mod` with the specified module path and an explicit minimum Go version, the `cmd/pr-report` source layout, and a `.gitignore` for generated binaries. Keep the existing specification and task document. Verify that `go build -o pr-report ./cmd/pr-report` produces the expected executable. References: §10.

  Completed 2026-09-22: inspected remote history and preserved the existing T01 commit and specifications. Added `go.mod` (module `github.com/fredyranthun/pr-reporter`, minimum Go 1.25.0), `cmd/pr-report/main.go` with `package main`, and generated-binary ignore rules. Verified `go build -o pr-report ./cmd/pr-report` with Go 1.25.1 on Linux/amd64, executable startup, binary exclusion from Git, formatting, and offline `go test ./...` (no test files yet). The entry point reports unimplemented collection on stderr and exits 1; CLI implementation remains T03.

- [x] **T03 — Implement CLI flags and validation.** Use `flag` for all flags in §3 with their exact defaults; support repeated `--repo`, concurrency 1–8, positive duration, and only `table`/`json` formats. Reject positional arguments, unknown or malformed flags, and missing sources before any API call. Implement `--help` and `--version` without requiring `gh` or repository input; include the review-body counting limitation in help. Test valid and invalid combinations and prove invalid input invokes no executor. References: §§3, 11.

  Completed 2026-09-22: implemented all §3 flags with `flag`, exact defaults, repeatable repository sources, range/duration/format checks, and exit 2 for invalid flags, positional arguments, or missing sources. Help/version work without `gh` or sources; help documents review-body exclusion. All flags are validated before informational output; help wins when both help and version are requested. Added offline tests for valid combinations and 44 invalid-input cases proving zero executor calls, plus help/version, output failures, and execution handoff. Verified `go test ./...`, formatting, the command build, and six built-command scenarios with an empty PATH. Repository parsing/normalization remains T04; report collection remains unimplemented.

- [x] **T04 — Implement repository file parsing and normalization.** Combine `--repos` and repeated `--repo` values; handle UTF-8, initial BOM, LF/CRLF, whitespace, blank lines, and full-line comments. Enforce the owner/name character rules and reject URLs, paths, `.git` suffixes, dot segments, and trailing comments. Report invalid file values with line numbers, handle file read failures, deduplicate case-insensitively, and reject an effectively empty list. Cover these cases in parser tests. References: §§3, 11.

  Completed 2026-09-22: added UTF-8 file parsing with initial BOM, LF/CRLF, whitespace, blank lines, and full-line comments; validated ASCII owner/name rules and rejected URLs, paths, `.git` suffixes, dot segments, and trailing comments. File entries precede repeated flags; case-insensitive deduplication retains the first spelling. Invalid file values include filename, line number, and value; read failures and effectively empty combined lists exit 2 before execution. Added parser and command tests, including read failures after partial input and zero executor calls for invalid sources. Verified offline `go test ./...`, formatting, the command build, and eight built-command scenarios. Help/version still skip file reads; collection remains unimplemented.

- [x] **T05 — Define API DTOs and public report models.** Model every PR field from §4 and every report, repository, error, and warning field from §9, including IDs needed internally for deduplication. Separate DTOs from output types. Preserve null versus zero/false, nullable authors and review decisions, raw enum strings, explicit null properties, and empty arrays as `[]`. Include `schema_version: 1`, requested/canonical repository names, and timestamps. Verify JSON field names and nullable serialization. References: §§4, 8–10.

  Completed 2026-09-22: Added separate presence-aware API DTOs and schema 1 public models. Tests verify explicit nulls, zero/false, raw enums, exact PR field names, empty arrays, and UTC timestamps. Verified offline `go test ./...`, formatting, and command build.

## Phase 2 — GitHub execution and PR listing (6 tasks)

Depends on the foundation; deliver paginated collection and basic JSON before adding thread details.

- [x] **T06 — Build the subprocess executor and test double.** Locate `gh` after input validation and invoke it through a small interface backed by `exec.CommandContext`, with separate arguments and no shell. Fix the host to `github.com`, preserve the authentication environment, and capture stdout, stderr, and process outcome separately. Never start login, extract tokens, or log credentials. Provide a fake executor with queued responses, invocation tracking, and cancellation support so tests need no network or token. References: §§3, 7, 10.

  Completed 2026-09-22: Added a shell-free CommandContext executor fixed to github.com, with separate streams and process status. Offline helper-process tests verify literal arguments, inherited environment, missing gh, and cancellation; a queued fake records calls. Verified offline `go test ./...`, formatting, and command build.

- [x] **T07 — Define the PR-list GraphQL query.** Use constant query text with separately passed variables for owner, name, page size, and cursor. Request open PRs including drafts, creation order ascending, pages of up to 100, canonical repository name, IDs, main PR fields, `comments.totalCount`, `reviewThreads.totalCount`, and page information. Fetch no comment/review bodies or large nested connections. Validate query arguments with executor fixtures. References: §§4, 5, 7.

  Completed 2026-09-22: Added the constant open-PR query with ascending creation order and 100-item pages. Executor fixtures verify separate literal variables, optional opaque cursors, all requested fields, and absence of body collection. Verified offline `go test ./...`, formatting, and command build.

- [x] **T08 — Decode and validate API responses.** Parse structured responses even when process execution fails; do not infer success from exit code alone. Reject malformed responses, missing required data according to T01, and any page containing GraphQL `errors`, even if `data` is present. Preserve inaccessible/nonexistent repository ambiguity. Introduce the spec's structured error context and codes, with generic API handling for rate-limit responses. Test mixed `data`/`errors`, invalid JSON, null repository, and process failures. References: §§7–9.

  Completed 2026-09-22: Added structured response validation and contextual error codes, including mixed data/errors and process failures. Fixtures cover required fields, malformed types, nullable details, inaccessible repositories, and pagination structure. Verified offline `go test ./...`, formatting, and command build.

- [x] **T09 — Implement explicit PR pagination.** Invoke `gh` once per page attempt and advance a repository's cursor sequentially until completion. Detect repeated cursors, absent next cursors, and invalid pages; impose no arbitrary total-PR cap. Deduplicate PRs by ID, retain only previously valid pages after a failure, and preserve identity by repository plus PR number. Test 205 unique PRs across three pages, duplicate IDs, cursor failures, and failure on page two. References: §§4, 7, 11.

  Completed 2026-09-22: Implemented explicit sequential PR pagination with whole-page validation and first-observation deduplication. Tests collect 205 PRs, preserve accepted pages after failure, and reject cursor, identity, canonical-name, and malformed-duplicate conflicts. Verified offline `go test ./...`, formatting, and command build.

- [x] **T10 — Collect multiple repositories sequentially.** Process all validated inputs with default concurrency one, continue after a repository fails, and preserve requested names alongside canonical API names. Track per-repository listing completion and record UTC run start/end and main-data observation times. Distinguish an inaccessible repository from a successfully queried empty one. Test mixed accessible/inaccessible repositories and identical PR numbers in different repositories. References: §§4, 7–9, 11.

  Completed 2026-09-22: Added sequential multi-repository collection with independent outcomes and UTC run/observation timestamps. Tests verify continuation after inaccessible repositories, canonical versus requested names, empty success, and identical PR numbers in different repositories. Verified offline `go test ./...`, formatting, and command build.

- [x] **T11 — Emit a basic JSON report for the first collection milestone.** Connect validated input, executor, listing, and models to a usable command. Emit one JSON object to stdout and diagnostics to stderr. Until thread collection is implemented, leave unknown details null and mark their incompleteness truthfully. Exercise a complete no-thread example and a partial-listing example through the command entry point. This is an intermediate milestone; T19 completes the JSON contract. References: §§9, 12.

  Completed 2026-09-22: Connected validated command input to gh collection and indented JSON output with isolated diagnostics. Offline command tests cover complete no-thread and partial-listing reports; unavailable thread details remain null and incomplete until T12. Verified offline `go test ./...`, formatting, and command build.

## Phase 3 — Thread details and report semantics (8 tasks)

Depends on reliable PR listing. Keep normalization separate from collection and test it as pure logic.

- [x] **T12 — Query and paginate review threads.** Add a constant thread query with independent cursor state per PR, pages up to 100, thread IDs, and `isResolved`. Skip enumeration when the known thread count is zero. Deduplicate IDs and count all unresolved threads, including outdated ones; fetch no bodies. Test 130 threads across two pages, resolved/outdated threads, duplicates, and independent cursors across PRs. References: §§5, 7, 11.

  Completed 2026-09-22: Added independent thread pagination, first-ID deduplication, and zero-total skipping without fetching bodies. Tests cover 130 threads, outdated unresolved entries, duplicate resolution changes, independent PR cursors, unknown totals, and non-atomic count differences. Verified offline `go test ./...`, formatting, and command build.

- [x] **T13 — Preserve PRs when detail collection fails.** Apply response and cursor validation to thread pages. If enumeration fails, retain the PR, set `unresolved_threads_count` to null, mark its details incomplete, and attach a `review_threads` error with repository/PR context. Preserve independently known fields and never expose a partial unresolved subtotal as final. Test failure after a valid first page and a PR that becomes unavailable between requests. References: §§5, 7–9, 11.

  Completed 2026-09-22: Validated detail-failure recovery through the full collector. Fixtures prove unavailable PRs, malformed threads, mixed GraphQL errors, failed processes, and second-page cursor failures retain independent PR fields while discarding unresolved subtotals and emitting contextual errors. Verified offline `go test ./...`, formatting, and command build.

- [x] **T14 — Derive comment indicators.** Implement all three states of `has_comments`: true from either positive count, false only when both counts are known zero, otherwise null. Include bot/author comments and resolved threads; exclude text present only in a review body. Keep thread counts distinct from inline-message counts. Test the known/unknown count combinations and the review-body-only acceptance case. References: §§5, 11.

  Completed 2026-09-22: Implemented the three-state comment indicator from principal counts only. Tests cover all known/unknown combinations, resolved threads, review-body-only text, and later threads with an unavailable principal count; corrected a mixed-error thread fixture. Verified offline `go test ./...`, formatting, and command build.

- [x] **T15 — Implement conservative merge classification.** Apply §6 precedence for incomplete/contradictory data, drafts, conflicts, `BLOCKED`, `CLEAN` plus `MERGEABLE`, and unknown combinations. Detect the specified `CLEAN` contradictions, keep unresolved threads from independently establishing blockage, and preserve raw API values. Test the full decision table, legitimate null review decisions, and contradictory combinations. References: §§6, 11.

  Completed 2026-09-22: Implemented conservative merge classification with required-field, unfamiliar-enum, and contradiction precedence. Decision-table tests cover all supported states, legitimate null review decisions, unrelated detail failures, and unresolved threads that do not independently block merge. Verified offline `go test ./...`, formatting, and command build.

- [x] **T16 — Derive signals and compatibility warnings.** Generate all supported signals only when corresponding fields justify them, and sort signals alphabetically. Preserve unfamiliar enum values, apply the interpretation defined in T01, and attach contextual compatibility warnings. Keep a `BLOCKED` PR blocked even without an identified cause. Test new enums, all supported signals, and warnings that do not automatically make collection incomplete. References: §§6, 9, 11.

  Completed 2026-09-22: Added alphabetically sorted independent signals and per-field unknown-enum warnings. Tests verify raw unfamiliar values override classification without making collection incomplete, known signals survive, and contradictions produce the specified signal set. Verified offline `go test ./...`, formatting, and command build.

- [x] **T17 — Finalize completeness and collection counters.** Compute PR details completion, each repository's separate listing/details markers, and report `complete` from collection outcomes. Successful empty repositories have both markers true; unknown business states alone do not make collection incomplete. Set unresolved repository names to null while preserving requested names. Compute pre-filter counts from unique collected records. Test complete empty output, partial listing, partial details, all repositories failing, and `UNKNOWN` merge state with complete collection. References: §§7–9, 11.

  Completed 2026-09-22: Locked down completeness and unique pre-filter counters with collector fixtures. Covered empty success, all-repository failure, partial listing/details, legitimate nulls, UNKNOWN and contradictory business states, and an incomplete first duplicate that later data must not repair. Verified offline `go test ./...`, formatting, and command build.

- [x] **T18 — Implement sorting and post-collection filtering.** Sort PRs by repository case-insensitively and number ascending. Apply `--only-unresolved` only after collection, including known positive unresolved counts and excluding zero or unknown counts. Preserve errors and completion markers, report the active filter, and distinguish `prs_collected` from `prs_returned`. Test mixed counts, filtered partial results, and a filter eliminating every record. References: §§3, 9, 11.

  Completed 2026-09-22: Added stable case-insensitive repository/number sorting and post-collection unresolved filtering. Tests verify known positive counts only, all-filtered output, preserved errors/completeness, and separate collected/returned counters. Verified offline `go test ./...`, formatting, and command build.

- [x] **T19 — Complete and lock down JSON output.** Render exactly one object with two-space indentation and the full §9 contract, including timestamps, repository outcomes, counters, PRs, errors, and warnings. Preserve full titles, original enum values, explicit nulls, and empty arrays; send every operational diagnostic to stderr. Test complete, empty, filtered, and incomplete reports against fixtures and verify stdout parses as one JSON value with no trailing diagnostics. References: §§4, 9, 11.

  Completed 2026-09-22: Locked schema 1 JSON output with reviewed golden fixtures for complete, empty, filtered, incomplete, and unfamiliar-enum reports. Tests enforce two-space indentation, one JSON value, full escaped titles, explicit nulls/arrays, raw enums, and stderr-only operational diagnostics. Verified offline `go test ./...`, formatting, and command build.

## Phase 4 — Table output and operational behavior (7 tasks)

Depends on the report model and collection behavior above.

- [x] **T20 — Render the terminal table.** Use `text/tabwriter`, no colors, and the exact columns from §9. Display only general comments under `COMENT.`, preserve the raw merge state, render blockage as `sim`/`não`/`?`, and show `?` for unavailable fields. Test representative complete and nullable rows. References: §9.

  Completed 2026-09-22: Added default terminal rendering with text/tabwriter and the exact specified columns. Tests verify general-comment counts, raw merge state, Portuguese blocked labels, nullable cells, and color-free output. Verified offline `go test ./...`, formatting, and command build.

- [x] **T21 — Sanitize terminal fields and truncate titles.** Remove terminal escape/control sequences, tabs, and newlines from every API-derived table field. If truncating titles, use the spec's 60-rune limit with an ellipsis and avoid splitting UTF-8. Preserve original values in JSON. Test escape sequences, line breaks, multibyte titles, and JSON escaping. References: §§9, 11.

  Completed 2026-09-22: Sanitized all API-derived table cells, including ANSI control strings, tabs/newlines, and format controls; titles use a 60-rune inclusive ellipsis limit. Tests cover CSI/OSC/C1 sequences, Unicode boundaries, every displayed string field, and unchanged escaped JSON. Verified offline `go test ./...`, formatting, and command build.

- [x] **T22 — Add table summaries and empty-result messages.** Show complete/incomplete repository totals and PRs collected/displayed. Distinguish no open PRs, no filter matches, and no recovered PRs with incomplete collection. Ensure partial reports remain visibly incomplete even when filtering hides affected records. Verify table output and stderr diagnostics independently. References: §§8, 9, 11.

  Completed 2026-09-22: Added table summaries for repository completeness and collected/displayed PR counts, plus distinct empty-success, no-filter-match, and incomplete-recovery messages. Tests verify filtered partial output remains visibly incomplete. Verified offline `go test ./...`, formatting, and command build.

- [x] **T23 — Implement request timeouts and user cancellation.** Apply `--timeout` per request attempt using derived contexts. Handle `Ctrl+C` through a shared cancelable context, terminate subprocesses, and stop pending work and waits. Distinguish request timeout from user cancellation; cancellation exits 130 without requiring a report. Test a timed-out request, cancellation during execution, and cancellation of queued work. References: §§3, 8, 10, 11.

  Completed 2026-09-22: Added per-attempt timeout contexts and shared Ctrl+C cancellation through subprocess execution and pending work. Offline tests distinguish timeout report errors from exit 130 cancellation and prove later repositories never start after cancellation. Verified offline `go test ./...`, formatting, and command build.

- [x] **T24 — Implement bounded retries for transient failures.** Retry timeout, network, and temporary server failures at most twice after the initial attempt, using 1s/2s waits plus small jitter. Make waits cancelable and testable without real delays. Do not retry authentication, permission, invalid queries, schema incompatibility, or unclassified API errors. Keep dedicated rate-limit detection/backoff outside the MVP. Test eventual success, exhaustion, non-retryable errors, and cancellation during waits. References: §§8, 11.

  Completed 2026-09-22: Added at most two retries for recognizable timeout/network/temporary-server failures, with cancelable 1s/2s waits and up to 100ms jitter. Deterministic tests cover eventual success, exhaustion, non-retryable auth/permission/query/schema/rate-limit responses, and cancellation during waits. Verified offline `go test ./...`, formatting, and command build.

- [x] **T25 — Finalize diagnostics and exit codes.** Return 0 for emitted complete reports, 1 for operational failure preventing report emission, 2 for invalid input/flags, 3 for emitted incomplete reports including all-repository failure, and 130 for user cancellation. Handle missing `gh` and output-write failures consistently, and preserve the specified structured error stages/codes. Test each exit path and ensure blocked PRs do not cause a nonzero exit in a complete report. References: §§3, 8, 9.

  Completed 2026-09-22: Validated all report exit paths for table and JSON: complete/blocked/empty 0, missing gh or output failure 1, input errors 2, all-repository failure 3, and cancellation 130. Structured context and stderr diagnostics remain separate from report output. Verified offline `go test ./...`, formatting, and command build.

- [x] **T26 — Complete the sequential command flow.** Wire validation, context setup, collection, normalization, filtering, rendering, and exit-code selection together. Ensure all input is validated before API work and errors from one repository do not stop independent repositories. Add offline end-to-end scenarios for default table output, JSON, multiple repositories, partial results, help/version, and filtered reports using the fake executor. References: §§3, 7–12.

  Completed 2026-09-22: Validated the complete sequential CLI using offline end-to-end scenarios for default table and JSON, combined/deduplicated sources, multiple repositories, thread details, partial results, filtering, and informational/invalid-input bypass. Verified offline `go test ./...`, formatting, and command build.

## Phase 5 — Concurrency, acceptance, and delivery (6 tasks)

Introduce concurrency after the sequential flow works. The flag is optional to use, but its implementation is part of the MVP.

- [x] **T27 — Add bounded global concurrency.** Honor `--concurrency` for independent repositories/detail requests while retaining sequential cursors within each connection. Share one global subprocess limit; do not multiply nested pools. Keep collection state race-free, scheduling cancelable, and report ordering stable. Retain sequential behavior at the default value of one. References: §§3, 7, 10.

  Completed 2026-09-22: Implemented bounded listing/detail worker phases with one global subprocess semaphore, cancelable scheduling, and deterministic result assembly. Tests demonstrate parallel requests without exceeding three active processes and stable repository/PR ordering; default execution remains sequential. Verified offline `go test ./...`, formatting, and command build.

- [x] **T28 — Validate concurrency and record development timings.** Use an instrumented fake executor to prove concurrency three never exceeds three active subprocesses, including retries and thread queries. Test one/eight bounds, concurrent partial failures, and cancellation without deadlock or leaked work. Run `go test -race ./...` in a compatible environment. Measure duration per repository during development without promising fixed latency or adding timing fields to the public schema. References: §§7, 10, 11.

  Completed 2026-09-22: Instrumented the fake executor to verify concurrency one/three/eight across retry and thread requests, concurrent partial failures, cancellation, and deterministic ordering. Measured per-run development timings in tests without public timing fields; `go test -race ./...` passed. Verified offline `go test ./...`, formatting, and command build.

- [x] **T29 — Audit acceptance coverage and automate checks.** Confirm all 18 acceptance scenarios below are covered by tests. Run formatting checks, `go test ./...`, the race suite after concurrency, and a command build; fix failures. Add a minimal GitHub Actions workflow for build/test checks, including race testing on a compatible Linux runner. The default suite must require neither network access to GitHub nor an authentication token. A live authenticated integration check remains optional/manual. References: §§10, 11.

  Completed 2026-09-22: Audited all 18 acceptance scenarios against named offline tests and added Linux GitHub Actions checks for formatting, tests, race detection, and build. Verified the audit references, local ordinary/race suites, and command build without GitHub authentication. Verified offline `go test ./...`, formatting, and command build.

- [x] **T30 — Write the README and usage documentation.** Explain build/install commands, minimum Go version, validated `gh` version, manual `gh auth login`, input formats, all flags/defaults, examples, output/schema semantics, exit codes, filtering, and partial results. Explain the review-body exclusion, thread-versus-comment counts, conservative blockage classification, non-atomic observation, and deferred rate-limit handling. Document development tests and optional manual integration steps. A new user must be able to install, authenticate, and report without reading the code. References: §§2–10, 12, 13.

  Completed 2026-09-22: Wrote a Portuguese user guide covering local and published installation, validated Go/gh versions, authentication, inputs, every flag/default, output/schema, exit codes, filtering, partial results, merge and thread limits, retries, tests, and optional manual integration. Verified offline `go test ./...`, formatting, and command build.

- [x] **T31 — Prepare and validate the first release artifact.** Build the Linux `pr-report` binary with a version reported by `--version`; choose and document the initial supported Linux architecture(s). Verify the documented local build/install procedure, help/version behavior, and release instructions. Prepare release notes including known limitations. Keep generated binaries out of source commits. References: §§3, 10, 12.

  Completed 2026-09-22: Prepared a static Linux/amd64 v0.1.0 binary, verified help/version, local build and install, and recorded SHA-256 and known limitations in release notes. Source installs report v0.1.0; the generated binary remains ignored by Git. Verified offline `go test ./...`, formatting, and command build.

- [x] **T32 — Publish the repository and first release.** Review the final changes, commit source/tests/docs/workflow, and push to `https://github.com/fredyranthun/pr-reporter.git` without overwriting existing remote work. Verify GitHub checks, create the release tag and release with the prepared binary, and verify the documented `go install github.com/fredyranthun/pr-reporter/cmd/pr-report@latest` command against the published version. Record completion in this checklist. This is a delivery task; writing this plan does not perform publication. References: §§10, 12 and the requested GitHub destination.

  Completed 2026-09-22: reviewed the clean release source and passing GitHub Actions workflow; pushed the annotated `v0.1.0` tag and [first GitHub release](https://github.com/fredyranthun/pr-reporter/releases/tag/v0.1.0) with the Linux/amd64 binary. GitHub's uploaded asset digest matched `f2cb17d9bbac166014540bfb3bec1b5b738b5185a31d5ec32b6b864d8cd29828`. Public `go install github.com/fredyranthun/pr-reporter/cmd/pr-report@latest` downloaded v0.1.0 and produced a working command reporting `pr-report v0.1.0`; help output matched the release artifact. Updated published installation guidance.

## Acceptance coverage map

This map is a cross-reference, not an additional set of tasks. Each row corresponds to one scenario in spec §11.

| Acceptance scenario | Tasks responsible |
|---|---|
| CRLF, comments, blank lines, duplicate input | T04 |
| Invalid/empty input; exit 2 and no API call | T03, T04, T25, T26 |
| 205 PRs over three pages | T09 |
| 130 threads over two pages | T12 |
| Outdated unresolved thread counted | T12 |
| Review-body-only text excluded | T14 |
| BLOCKED, DIRTY, draft, CLEAN, UNKNOWN | T15, T17 |
| Contradictory merge data | T15, T16 |
| New API enum | T16 |
| Accessible and inaccessible repositories together | T10, T17, T25, T26 |
| Failure on second PR page | T09, T17 |
| Failure during thread pagination | T13, T17 |
| GraphQL data together with errors | T08, T13 |
| Successfully queried empty repository | T17, T19, T25 |
| Filter eliminates every record | T18, T19, T22 |
| Bounded, cancelable timeout retries | T23, T24 |
| At most three active subprocesses | T27, T28 |
| Terminal escapes/newlines sanitized; JSON escaped | T19, T21 |

## Deferred — Rate-limit handling (5 tasks, outside the MVP)

Based on spec §13. These tasks do not block T01–T32 or MVP completion. Until implemented, affected requests follow generic API failure handling and retain the spec's partial-result behavior.

- [ ] **F01 — Capture HTTP metadata.** Extend the executor response format to separate status/headers from the JSON body, with parsing tests.
- [ ] **F02 — Classify rate-limit failures.** Identify primary and secondary limits and introduce the contextual `rate_limit` report code without losing original diagnostics.
- [ ] **F03 — Define and implement bounded wait policy.** Handle `Retry-After`, reset times, absent timing information, and a maximum permitted wait; document the selected behavior.
- [ ] **F04 — Coordinate global pauses.** Pause pending requests across workers and make all waits cancelable while preserving already collected results.
- [ ] **F05 — Bound and test recovery.** Define a rate-limit retry budget, test recovery/exhaustion/concurrency/cancellation, and update help/README as needed.

## Completion rule

The implementation is ready when T01–T31 are complete, all spec acceptance cases pass, and the report never silently treats failed collection as complete. Delivery is complete when T32 is also complete. Deferred tasks F01–F05 are tracked separately. No database, service, scheduler, notifications, web/TUI, PR mutations, Enterprise Server support, or full ruleset/merge-queue evaluation is included.
