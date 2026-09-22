# MVP acceptance coverage

This maps each acceptance scenario in §11 of `pr-report-spec.md` to an offline test. The suite uses controlled API responses and never needs a GitHub token or network access.

| Scenario | Test |
|---|---|
| CRLF, comments, blanks, duplicate input | `TestParseRepositoryFile`, `TestLoadRepositories` |
| Invalid/empty input; exit 2 before execution | `TestInvalidInputNeverExecutes`, `TestRepositoryInputValidatedBeforeExecution` |
| 205 PRs on three pages | `TestList205PRs` |
| 130 threads on two pages | `TestThreads130AndDuplicates` |
| Outdated unresolved thread | `TestThreads130AndDuplicates` |
| Review-body-only text excluded | `TestReviewBodyOnlyAndResolvedThreads` |
| BLOCKED, DIRTY, draft, CLEAN, UNKNOWN | `TestMergeDecisionTable` |
| Contradictory merge data | `TestMergeDecisionTable`, `TestSignalsAndCompatibilityWarnings` |
| New API enum | `TestSignalsAndCompatibilityWarnings`, `TestJSONGoldenReports` |
| Accessible plus inaccessible repositories | `TestCollectMultipleRepositories`, `TestConcurrentPartialFailureAndCancellation` |
| Failure on second PR page | `TestListRejectsWholeConflictingPage` |
| Failure during thread pagination | `TestThreadFailurePreservesPRWithoutSubtotal` |
| GraphQL data together with errors | `TestDecodeEnvelopes`, `TestThreadFailurePreservesPRWithoutSubtotal` |
| Successfully queried empty repository | `TestCompletenessAndCounters`, `TestJSONGoldenReports` |
| Filter removes all records | `TestSortingAndPostCollectionFilter`, `TestJSONGoldenReports` |
| Bounded, cancelable timeout retries | `TestPerAttemptTimeout`, `TestRetryBudgetAndBackoff`, `TestCancelDuringRetryWait` |
| At most three active subprocesses | `TestConcurrentCollectionStableOutput`, `TestConcurrencyBoundsRetriesAndThreads` |
| Terminal escapes/newlines sanitized; JSON escaped | `TestTerminalSanitization`, `TestJSONGoldenReports` |

The CI workflow checks formatting, ordinary and race-enabled tests, and the command build on Linux. Authenticated integration against GitHub remains a manual check.
