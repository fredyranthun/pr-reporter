#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd "$(dirname "$0")/.." && pwd)"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT
fixture_dir="$test_root/fixtures"
fake_dir="$test_root/fake-bin"
install_dir="$test_root/installed"
curl_log="$test_root/curl.log"
mkdir -p "$fixture_dir" "$fake_dir"

cat > "$fixture_dir/pr-report-linux-amd64" <<'SH'
#!/bin/sh
test "$1" = --version
printf 'pr-report v0.1.1\n'
SH
chmod +x "$fixture_dir/pr-report-linux-amd64"

write_checksums() {
  sha256sum "$fixture_dir/pr-report-linux-amd64" |
    awk '{print $1 "  pr-report-linux-amd64"}' > "$fixture_dir/checksums.txt"
}
write_checksums

cat > "$fake_dir/curl" <<'SH'
#!/bin/sh
set -eu
test "$1" = -fsSL
url=$2
test "$3" = -o
output=$4
case "$url" in
  https://github.com/fredyranthun/pr-reporter/releases/latest/download/checksums.txt|\
  https://github.com/fredyranthun/pr-reporter/releases/latest/download/pr-report-linux-amd64|\
  https://github.com/fredyranthun/pr-reporter/releases/download/v0.1.1/checksums.txt|\
  https://github.com/fredyranthun/pr-reporter/releases/download/v0.1.1/pr-report-linux-amd64) ;;
  *) exit 1 ;;
esac
printf '%s\n' "$url" >> "$CURL_LOG"
cp "$FIXTURE_DIR/$(basename "$url")" "$output"
SH
cat > "$fake_dir/uname" <<'SH'
#!/bin/sh
case "$1" in
  -s) printf '%s\n' "$TEST_OS" ;;
  -m) printf '%s\n' "$TEST_ARCH" ;;
  *) exit 1 ;;
esac
SH
chmod +x "$fake_dir/curl" "$fake_dir/uname"

test_os=Linux
test_arch=x86_64
version=
run_installer() {
  env PATH="$fake_dir:$PATH" INSTALL_DIR="$install_dir" FIXTURE_DIR="$fixture_dir" \
    CURL_LOG="$curl_log" TEST_OS="$test_os" TEST_ARCH="$test_arch" \
    PR_REPORT_VERSION="$version" sh "$repo_dir/scripts/install.sh"
}

run_installer > "$test_root/output"
test -x "$install_dir/pr-report"
cmp "$fixture_dir/pr-report-linux-amd64" "$install_dir/pr-report"
grep -Fq 'pr-report v0.1.1' "$test_root/output"
grep -Fq '/releases/latest/download/checksums.txt' "$curl_log"

# Installing latest again must replace an existing older binary.
cat > "$install_dir/pr-report" <<'SH'
#!/bin/sh
printf 'pr-report v0.1.0\n'
SH
chmod +x "$install_dir/pr-report"
test "$("$install_dir/pr-report" --version)" = 'pr-report v0.1.0'
run_installer > "$test_root/output"
grep -Fq 'pr-report v0.1.1' "$test_root/output"
cmp "$fixture_dir/pr-report-linux-amd64" "$install_dir/pr-report"

printf '%064d  pr-report-linux-amd64\n' 0 > "$fixture_dir/checksums.txt"
if run_installer > "$test_root/output" 2>&1; then
  printf 'Corrupt binary checksum was accepted.\n' >&2
  exit 1
fi
grep -Fq 'checksum mismatch' "$test_root/output"
cmp "$fixture_dir/pr-report-linux-amd64" "$install_dir/pr-report"

test_os=Darwin
: > "$curl_log"
if run_installer > "$test_root/output" 2>&1; then
  printf 'Unsupported platform was accepted.\n' >&2
  exit 1
fi
grep -Fq 'Unsupported platform' "$test_root/output"
test ! -s "$curl_log"

test_os=Linux
version=v0.1.1
write_checksums
run_installer > "$test_root/output"
grep -Fq '/releases/download/v0.1.1/checksums.txt' "$curl_log"
cmp "$fixture_dir/pr-report-linux-amd64" "$install_dir/pr-report"

printf 'Installer checks passed.\n'
