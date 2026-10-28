#!/usr/bin/env bash
# Capture a command's cmd/stdout/stderr/exit under a path prefix.
# Usage: capture.sh <prefix> <command…>
set -euo pipefail

if [ "$#" -lt 2 ]; then
  echo "usage: $0 <prefix> <command…>" >&2
  exit 2
fi

prefix=$1
shift
mkdir -p "$(dirname "$prefix")"
printf '%s\n' "$*" >"${prefix}.cmd.txt"

set +e
"$@" >"${prefix}.stdout.txt" 2>"${prefix}.stderr.txt"
ec=$?
set -e
printf '%s\n' "$ec" >"${prefix}.exit.txt"

if [ "$ec" -eq 0 ] && grep -q 'just bench' "${prefix}.cmd.txt" 2>/dev/null; then
  cp "${prefix}.stdout.txt" "${prefix}.result.json"
elif [ "$ec" -eq 0 ] && head -1 "${prefix}.stdout.txt" 2>/dev/null | grep -q '^{'; then
  if grep -q '"metric"' "${prefix}.stdout.txt" 2>/dev/null; then
    cp "${prefix}.stdout.txt" "${prefix}.result.json"
  fi
fi

exit "$ec"
