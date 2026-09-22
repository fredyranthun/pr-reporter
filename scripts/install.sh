#!/bin/sh
set -eu

repo=fredyranthun/pr-reporter
asset=pr-report-linux-amd64

os=$(uname -s)
arch=$(uname -m)
case "$os:$arch" in
  Linux:x86_64|Linux:amd64) ;;
  *)
    printf 'Unsupported platform: %s/%s (available: Linux/amd64).\n' "$os" "$arch" >&2
    exit 1
    ;;
esac

for tool in curl awk mktemp mkdir install mv; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    printf 'Required command not found: %s\n' "$tool" >&2
    exit 1
  fi
done

if command -v sha256sum >/dev/null 2>&1; then
  checksum_tool=sha256sum
elif command -v shasum >/dev/null 2>&1; then
  checksum_tool=shasum
else
  printf 'Required command not found: sha256sum or shasum\n' >&2
  exit 1
fi

install_dir=${INSTALL_DIR:-"${HOME:?Set HOME or INSTALL_DIR}/.local/bin"}
if [ -n "${PR_REPORT_VERSION:-}" ]; then
  case "$PR_REPORT_VERSION" in
    *[!a-zA-Z0-9._-]*)
      printf 'Invalid PR_REPORT_VERSION: %s\n' "$PR_REPORT_VERSION" >&2
      exit 1
      ;;
  esac
  base_url="https://github.com/$repo/releases/download/$PR_REPORT_VERSION"
else
  base_url="https://github.com/$repo/releases/latest/download"
fi

tmpdir=$(mktemp -d)
staged=
cleanup() {
  if [ -n "$staged" ]; then
    rm -f "$staged"
  fi
  rm -rf "$tmpdir"
}
trap cleanup 0
trap 'exit 1' 1 2 3 15

printf 'Downloading checksums from %s\n' "$base_url"
curl -fsSL "$base_url/checksums.txt" -o "$tmpdir/checksums.txt"
if ! expected=$(awk -v name="$asset" '
  $2 == name && NF == 2 && length($1) == 64 && $1 !~ /[^[:xdigit:]]/ {
    count++
    checksum = $1
  }
  END {
    if (count != 1) exit 1
    print checksum
  }
' "$tmpdir/checksums.txt"); then
  printf 'No unique SHA-256 checksum for %s in release.\n' "$asset" >&2
  exit 1
fi

printf 'Downloading %s\n' "$asset"
curl -fsSL "$base_url/$asset" -o "$tmpdir/$asset"
if [ "$checksum_tool" = sha256sum ]; then
  actual=$(sha256sum "$tmpdir/$asset" | awk '{print $1}')
else
  actual=$(shasum -a 256 "$tmpdir/$asset" | awk '{print $1}')
fi
if [ "$expected" != "$actual" ]; then
  printf 'SHA-256 checksum mismatch for %s.\n' "$asset" >&2
  exit 1
fi

mkdir -p "$install_dir"
staged=$(mktemp "$install_dir/.pr-report.XXXXXX")
install -m 755 "$tmpdir/$asset" "$staged"
mv -f "$staged" "$install_dir/pr-report"
staged=

printf 'Installed %s\n' "$install_dir/pr-report"
"$install_dir/pr-report" --version
case ":$PATH:" in
  *":$install_dir:"*) ;;
  *) printf 'Add %s to your PATH to run pr-report from any directory.\n' "$install_dir" ;;
esac
