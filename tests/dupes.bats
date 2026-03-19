#!/usr/bin/env bats

setup_file() {
	PROJECT_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"
	export PROJECT_ROOT

	ORIGINAL_HOME="${HOME:-}"
	export ORIGINAL_HOME

	ORIGINAL_GOCACHE="$(go env GOCACHE 2>/dev/null || true)"
	export ORIGINAL_GOCACHE

	HOME="$(mktemp -d "${BATS_TEST_DIRNAME}/tmp-dupes-home.XXXXXX")"
	export HOME

	mkdir -p "$HOME"

	# Build dupes binary from current source.
	if command -v go >/dev/null 2>&1; then
		DUPES_BIN="$(mktemp "${TMPDIR:-/tmp}/dupes-go.XXXXXX")"
		GOPATH="${ORIGINAL_HOME}/go" GOMODCACHE="${ORIGINAL_HOME}/go/pkg/mod" \
			GOCACHE="${ORIGINAL_GOCACHE}" \
			go build -o "$DUPES_BIN" "$PROJECT_ROOT/cmd/dupes" 2>/dev/null
		export DUPES_BIN
	fi
}

teardown_file() {
	rm -rf "$HOME"
	if [[ -n "${ORIGINAL_HOME:-}" ]]; then
		export HOME="$ORIGINAL_HOME"
	fi
	rm -f "${DUPES_BIN:-}"
}

setup() {
	if [[ ! -x "${DUPES_BIN:-}" ]]; then
		skip "dupes binary not available"
	fi

	TEST_DIR="$(mktemp -d "${TMPDIR:-/tmp}/dupes-test.XXXXXX")"
	export TEST_DIR
}

teardown() {
	rm -rf "${TEST_DIR:-}"
}

# Helper: create file with deterministic content.
create_file() {
	local path="$1"
	local size="${2:-1024}"
	local seed="${3:-0}"
	mkdir -p "$(dirname "$path")"
	python3 -c "
import sys
data = bytes([(i + $seed) % 256 for i in range($size)])
sys.stdout.buffer.write(data)
" >"$path"
}

# --- Report Mode ---

@test "dupes: no duplicates found in empty dir" {
	run "$DUPES_BIN" --min-size 0 "$TEST_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"No duplicates found"* ]]
}

@test "dupes: no duplicates with unique files" {
	create_file "$TEST_DIR/a.txt" 2048 0
	create_file "$TEST_DIR/b.txt" 2048 1

	run "$DUPES_BIN" --min-size 0 "$TEST_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"No duplicates found"* ]]
}

@test "dupes: finds duplicate pair" {
	create_file "$TEST_DIR/a.txt" 2048 0
	cp "$TEST_DIR/a.txt" "$TEST_DIR/b.txt"

	run "$DUPES_BIN" --min-size 0 "$TEST_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"2 copies"* ]]
	[[ "$output" == *"a.txt"* ]]
	[[ "$output" == *"b.txt"* ]]
}

@test "dupes: finds three copies" {
	create_file "$TEST_DIR/a.txt" 4096 0
	cp "$TEST_DIR/a.txt" "$TEST_DIR/b.txt"
	cp "$TEST_DIR/a.txt" "$TEST_DIR/c.txt"

	run "$DUPES_BIN" --min-size 0 "$TEST_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"3 copies"* ]]
}

@test "dupes: finds dupes across subdirectories" {
	create_file "$TEST_DIR/dir1/a.txt" 2048 0
	mkdir -p "$TEST_DIR/dir2"
	cp "$TEST_DIR/dir1/a.txt" "$TEST_DIR/dir2/a.txt"

	run "$DUPES_BIN" --min-size 0 "$TEST_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"2 copies"* ]]
}

@test "dupes: respects --min-size filter" {
	# Small dupes (below threshold).
	create_file "$TEST_DIR/small1.txt" 500 0
	cp "$TEST_DIR/small1.txt" "$TEST_DIR/small2.txt"

	# Large dupes (above threshold).
	create_file "$TEST_DIR/big1.txt" 2048 0
	cp "$TEST_DIR/big1.txt" "$TEST_DIR/big2.txt"

	run "$DUPES_BIN" --min-size 1KB "$TEST_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"2 copies"* ]]
	[[ "$output" == *"big"* ]]
}

@test "dupes: shows reclaimable space in summary" {
	create_file "$TEST_DIR/a.txt" 4096 0
	cp "$TEST_DIR/a.txt" "$TEST_DIR/b.txt"

	run "$DUPES_BIN" --min-size 0 "$TEST_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"reclaimable"* ]] || [[ "$output" == *"Reclaimable"* ]]
}

# --- JSON Mode ---

@test "dupes: --json outputs valid JSON with no dupes" {
	# Redirect stderr to avoid mixing with JSON stdout.
	local json_out
	json_out="$("$DUPES_BIN" --json --min-size 0 "$TEST_DIR" 2>/dev/null)"
	echo "$json_out" | python3 -c "import sys, json; d = json.load(sys.stdin); assert d['total_groups'] == 0"
}

@test "dupes: --json outputs valid JSON with dupes" {
	create_file "$TEST_DIR/a.txt" 2048 0
	cp "$TEST_DIR/a.txt" "$TEST_DIR/b.txt"

	local json_out
	json_out="$("$DUPES_BIN" --json --min-size 0 "$TEST_DIR" 2>/dev/null)"
	echo "$json_out" | python3 -c "
import sys, json
d = json.load(sys.stdin)
assert d['total_groups'] == 1
assert d['groups'][0]['copies'] == 2
assert d['reclaimable_bytes'] > 0
"
}

@test "dupes: --json includes file paths" {
	create_file "$TEST_DIR/x.dat" 2048 0
	cp "$TEST_DIR/x.dat" "$TEST_DIR/y.dat"

	local json_out
	json_out="$("$DUPES_BIN" --json --min-size 0 "$TEST_DIR" 2>/dev/null)"
	echo "$json_out" | python3 -c "
import sys, json
d = json.load(sys.stdin)
paths = [f['path'] for f in d['groups'][0]['files']]
assert any('x.dat' in p for p in paths)
assert any('y.dat' in p for p in paths)
"
}

# --- Conserve + Restore ---

@test "dupes: conserve moves duplicates preserving paths" {
	create_file "$TEST_DIR/src/a.txt" 2048 0
	cp "$TEST_DIR/src/a.txt" "$TEST_DIR/src/b.txt"

	CONSERVE_DIR="$TEST_DIR/conserve"

	run "$DUPES_BIN" --conserve "$CONSERVE_DIR" --min-size 0 "$TEST_DIR/src"
	[ "$status" -eq 0 ]
	[[ "$output" == *"conserved"* ]] || [[ "$output" == *"Conserved"* ]]

	# Keeper should still exist.
	[ -f "$TEST_DIR/src/a.txt" ] || [ -f "$TEST_DIR/src/b.txt" ]

	# Manifest should exist.
	[ -f "$CONSERVE_DIR/.mole-manifest.json" ]
}

@test "dupes: restore brings back conserved files" {
	create_file "$TEST_DIR/src/a.txt" 2048 0
	cp "$TEST_DIR/src/a.txt" "$TEST_DIR/src/b.txt"

	CONSERVE_DIR="$TEST_DIR/conserve"

	# Conserve first.
	"$DUPES_BIN" --conserve "$CONSERVE_DIR" --min-size 0 "$TEST_DIR/src" 2>/dev/null

	# One file should be gone.
	local missing=0
	[ -f "$TEST_DIR/src/a.txt" ] || missing=$((missing + 1))
	[ -f "$TEST_DIR/src/b.txt" ] || missing=$((missing + 1))
	[ "$missing" -eq 1 ]

	# Now restore.
	run "$DUPES_BIN" --restore "$CONSERVE_DIR"
	[ "$status" -eq 0 ]
	[[ "$output" == *"restored"* ]] || [[ "$output" == *"Restored"* ]]

	# Both should exist again.
	[ -f "$TEST_DIR/src/a.txt" ]
	[ -f "$TEST_DIR/src/b.txt" ]
}

@test "dupes: restore single file with --file" {
	create_file "$TEST_DIR/src/a.txt" 2048 0
	cp "$TEST_DIR/src/a.txt" "$TEST_DIR/src/b.txt"
	cp "$TEST_DIR/src/a.txt" "$TEST_DIR/src/c.txt"

	CONSERVE_DIR="$TEST_DIR/conserve"

	"$DUPES_BIN" --conserve "$CONSERVE_DIR" --min-size 0 "$TEST_DIR/src" 2>/dev/null

	# Verify manifest exists before trying to read it.
	[ -f "$CONSERVE_DIR/.mole-manifest.json" ]

	# Find one conserved file from manifest.
	local target
	target="$(python3 -c "
import json
m = json.load(open('$CONSERVE_DIR/.mole-manifest.json'))
print(m['entries'][0]['original_path'])
")"

	run "$DUPES_BIN" --restore "$CONSERVE_DIR" --file "$target"
	[ "$status" -eq 0 ]

	# Target should be restored.
	[ -f "$target" ]
}

# --- Flag Validation ---

@test "dupes: --delete and --conserve are mutually exclusive" {
	run "$DUPES_BIN" --delete --conserve /tmp/c "$TEST_DIR"
	[ "$status" -ne 0 ]
	[[ "$output" == *"mutually exclusive"* ]]
}

@test "dupes: --delete and --restore are mutually exclusive" {
	run "$DUPES_BIN" --delete --restore /tmp/r
	[ "$status" -ne 0 ]
	[[ "$output" == *"mutually exclusive"* ]]
}

@test "dupes: --help shows usage" {
	run "$DUPES_BIN" --help
	[ "$status" -eq 0 ]
	[[ "$output" == *"mo dupes"* ]]
	[[ "$output" == *"--conserve"* ]]
	[[ "$output" == *"--restore"* ]]
}

# --- CLI Integration ---

@test "dupes: mole dispatches to dupes" {
	run env HOME="$HOME" "$PROJECT_ROOT/mole" dupes --help
	[ "$status" -eq 0 ]
	[[ "$output" == *"mo dupes"* ]]
}

@test "dupes: mole help lists dupes command" {
	run env HOME="$HOME" "$PROJECT_ROOT/mole" --help
	[ "$status" -eq 0 ]
	[[ "$output" == *"dupes"* ]]
}
