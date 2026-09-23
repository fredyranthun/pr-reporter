# pr-report v0.1.1

This release adds `--fields` to select pull request fields in table or JSON output. For example, `--fields repo,number,title,url` shows four table columns; with `--format json`, it selects those properties inside each `pull_requests` object while preserving report metadata, completeness, counters, errors, and warnings. Run `pr-report --help` for all field names. Output without `--fields` is unchanged.

## Install or upgrade without Go

On Linux/amd64 (including WSL), run the same installer used for v0.1.0. It downloads the latest release, verifies the binary against this release's `checksums.txt`, and replaces an existing `~/.local/bin/pr-report` installation:

```sh
curl -fsSL https://raw.githubusercontent.com/fredyranthun/pr-reporter/main/scripts/install.sh | sh
"$HOME/.local/bin/pr-report" --version
```

To pin this release, use `PR_REPORT_VERSION=v0.1.1 sh` instead of `sh`. `INSTALL_DIR` can change the installation directory. The installer does not require Go. GitHub CLI `gh` is still required to query repositories.

The Linux/amd64 binary's SHA-256 is:

```text
a31932cbec8f044c55e8edd5e81e0ea655b724c5f0635b1a7bdd82ad12bb109d
```

You can also install from source with Go 1.25.0 or newer:

```sh
go install github.com/fredyranthun/pr-reporter/cmd/pr-report@v0.1.1
```
