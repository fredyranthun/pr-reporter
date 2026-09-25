# pr-report v0.1.2

This release adds `--format chat` for copying a pull request report into a Slack or Microsoft Teams conversation or thread. It groups PRs by repository and shows their title, author, status, comment and unresolved-thread counts, review and merge state, and full URL. The layout uses plain text, spacing, and Unicode symbols, so it remains readable when a chat composer pastes text without applying Markdown. Partial collection and filtered reports are labeled clearly.

Use `--fields` with chat output to show only selected PR fields as labeled lines in the requested order. Table and JSON output remain available.

## Install or upgrade without Go

On Linux/amd64 (including WSL), run the installer. It downloads the latest release, verifies the binary against this release's `checksums.txt`, and replaces an existing `~/.local/bin/pr-report` installation:

```sh
curl -fsSL https://raw.githubusercontent.com/fredyranthun/pr-reporter/main/scripts/install.sh | sh
"$HOME/.local/bin/pr-report" --version
"$HOME/.local/bin/pr-report" --repo owner/name --format chat
```

To pin this release, use `PR_REPORT_VERSION=v0.1.2 sh` instead of `sh`. `INSTALL_DIR` can change the installation directory. The installer does not require Go. GitHub CLI `gh` is still required to query repositories.

The Linux/amd64 binary's SHA-256 is:

```text
2e437b4813ec6878485c29fa0be512ae594d647493aa971deba0e288c68228ed
```

You can also install from source with Go 1.25.0 or newer:

```sh
go install github.com/fredyranthun/pr-reporter/cmd/pr-report@v0.1.2
```
