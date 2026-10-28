#!/usr/bin/env bash
# Remove a jmark project created for verification. Execution workspaces clean themselves.
# Usage: cleanup-project.sh <id>
# Does not touch verification evidence under artifacts/.
set -euo pipefail

id=${1:-}
case "$id" in
  ''|*/*|.*) echo "usage: $0 <id>" >&2; exit 2 ;;
esac

root=$(cd "$(dirname "$0")/../../../.." && pwd)
cd "$root"

rm -rf "benches/$id"

echo "cleanup-project: removed $id"
