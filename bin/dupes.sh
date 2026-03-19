#!/bin/bash
# Burrow - Dupes command.
# Runs the Go duplicate file finder.
# Finds and manages true content-duplicate files using xxhash.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_BIN="$SCRIPT_DIR/dupes-go"
if [[ -x "$GO_BIN" ]]; then
    exec "$GO_BIN" "$@"
fi

echo "Bundled dupes binary not found. Please reinstall Burrow or run bw update to restore it." >&2
exit 1
