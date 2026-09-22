# pr-report v0.1.0

First MVP release for Linux/amd64. It reports open GitHub pull requests, including drafts, general-comment counts, unresolved review threads, raw review/merge states, conservative blockage, and contextual partial failures. Output is a terminal table or schema 1 JSON. Input accepts repeated repositories or a UTF-8 file. Collection supports pagination, bounded concurrency, per-request timeouts, cancelable retries, and `Ctrl+C`.

## Install

Download `pr-report-linux-amd64` from this release, verify its SHA-256 checksum below, mark it executable, and run it. Alternatively, with Go 1.25.0 or newer:

```sh
go install github.com/fredyranthun/pr-reporter/cmd/pr-report@v0.1.0
pr-report --version
```

GitHub CLI `gh` must be installed and authenticated separately (`gh auth login --hostname github.com`). Development validated `gh` 2.100.0 and Go 1.25.1 on Linux/amd64.

SHA-256 `pr-report-linux-amd64`:

```text
f2cb17d9bbac166014540bfb3bec1b5b738b5185a31d5ec32b6b864d8cd29828
```

The [README](https://github.com/fredyranthun/pr-reporter/blob/v0.1.0/README.md) explains flags, input, output, exit codes, and optional manual verification.

## Known limits

Review-body-only text does not enter comment indicators, and thread counts are counts of discussions rather than individual inline messages. Merge classification does not evaluate every ruleset or merge queue. Results span several API observations, so counts can differ between principal and thread queries. Rate limits receive generic API failure handling; dedicated backoff is deferred. This release supports `github.com`, not GitHub Enterprise Server.
