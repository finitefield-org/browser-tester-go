#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

changed_go_files=$(mktemp)
trap 'rm -f "$changed_go_files"' EXIT INT HUP TERM

{
  git diff --name-only --diff-filter=ACMRTUXB HEAD -- '*.go'
  git ls-files --others --exclude-standard -- '*.go'
} | sort -u >"$changed_go_files"

if [ -s "$changed_go_files" ]; then
  while IFS= read -r file; do
    [ -n "$file" ] || continue
    gofmt -w "$file"
  done <"$changed_go_files"
fi

go test ./... -count=1
