#!/usr/bin/env bats

setup_file() {
    PROJECT_ROOT="$(cd "${BATS_TEST_DIRNAME}/.." && pwd)"
    export PROJECT_ROOT

    ORIGINAL_HOME="${HOME:-}"
    export ORIGINAL_HOME

    HOME="$(mktemp -d "${BATS_TEST_DIRNAME}/tmp-size.XXXXXX")"
    export HOME

    mkdir -p "$HOME"
}

teardown_file() {
    rm -rf "$HOME"
    if [[ -n "${ORIGINAL_HOME:-}" ]]; then
        export HOME="$ORIGINAL_HOME"
    fi
}

setup() {
    export TERM="xterm-256color"
    rm -rf "${HOME:?}"/*
    rm -rf "$HOME/Library" "$HOME/.config" "$HOME/.cache"
    mkdir -p "$HOME/Library/Caches" "$HOME/.config/burrow"
}

# ---------- basic invocation ----------

@test "bw size exits 0 and produces tabular output" {
    mkdir -p "$HOME/.npm"
    dd if=/dev/zero of="$HOME/.npm/dummy" bs=1024 count=10 2>/dev/null

    run env HOME="$HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh"

    [ "$status" -eq 0 ]
    [[ "$output" == *"Cache"* ]]
    [[ "$output" == *"Size"* ]]
    [[ "$output" == *"Path"* ]]
    [[ "$output" == *"npm"* ]]
}

# ---------- --json flag ----------

@test "bw size --json produces valid JSON" {
    mkdir -p "$HOME/.npm"
    dd if=/dev/zero of="$HOME/.npm/dummy" bs=1024 count=10 2>/dev/null

    run env HOME="$HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh" --json

    [ "$status" -eq 0 ]
    # Validate it is parseable JSON (python is available on macOS)
    echo "$output" | python3 -m json.tool >/dev/null 2>&1
    # Check structure
    echo "$output" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert 'caches' in data, 'missing caches key'
assert 'total_bytes' in data, 'missing total_bytes key'
assert isinstance(data['caches'], list), 'caches not a list'
assert len(data['caches']) > 0, 'caches is empty'
entry = data['caches'][0]
assert 'name' in entry, 'entry missing name'
assert 'size_bytes' in entry, 'entry missing size_bytes'
assert 'path' in entry, 'entry missing path'
"
}

# ---------- empty state ----------

@test "bw size shows no-caches message when nothing exists" {
    # Use a completely empty HOME with no pre-created directories
    local EMPTY_HOME
    EMPTY_HOME="$(mktemp -d "${BATS_TEST_DIRNAME}/tmp-size-empty.XXXXXX")"

    run env HOME="$EMPTY_HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh"

    rm -rf "$EMPTY_HOME"
    [ "$status" -eq 0 ]
    [[ "$output" == *"No developer caches found"* ]]
}

@test "bw size --json returns empty caches array when nothing exists" {
    local EMPTY_HOME
    EMPTY_HOME="$(mktemp -d "${BATS_TEST_DIRNAME}/tmp-size-empty.XXXXXX")"

    run env HOME="$EMPTY_HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh" --json

    rm -rf "$EMPTY_HOME"
    [ "$status" -eq 0 ]
    echo "$output" | python3 -c "
import sys, json
data = json.load(sys.stdin)
assert data['caches'] == [], 'expected empty caches'
assert data['total_bytes'] == 0, 'expected zero total'
"
}

# ---------- size calculation ----------

@test "bw size correctly calculates sizes for fake cache dirs" {
    mkdir -p "$HOME/.npm"
    mkdir -p "$HOME/.cargo"
    dd if=/dev/zero of="$HOME/.npm/data" bs=1024 count=100 2>/dev/null
    dd if=/dev/zero of="$HOME/.cargo/data" bs=1024 count=200 2>/dev/null

    run env HOME="$HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh"

    [ "$status" -eq 0 ]
    [[ "$output" == *"npm"* ]]
    [[ "$output" == *"Cargo"* ]]
}

@test "bw size sorts entries by size descending" {
    mkdir -p "$HOME/.npm"
    mkdir -p "$HOME/.cargo"
    # npm smaller than cargo
    dd if=/dev/zero of="$HOME/.npm/data" bs=1024 count=10 2>/dev/null
    dd if=/dev/zero of="$HOME/.cargo/data" bs=1024 count=500 2>/dev/null

    run env HOME="$HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh" --json

    [ "$status" -eq 0 ]
    echo "$output" | python3 -c "
import sys, json
data = json.load(sys.stdin)
sizes = [c['size_bytes'] for c in data['caches']]
assert sizes == sorted(sizes, reverse=True), f'not sorted descending: {sizes}'
"
}

# ---------- --help flag ----------

@test "bw size --help shows usage information" {
    run env HOME="$HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh" --help

    [ "$status" -eq 0 ]
    [[ "$output" == *"Usage"* ]]
    [[ "$output" == *"bw size"* ]]
    [[ "$output" == *"--json"* ]]
}

# ---------- total row ----------

@test "bw size shows total row in table output" {
    mkdir -p "$HOME/.npm"
    dd if=/dev/zero of="$HOME/.npm/data" bs=1024 count=50 2>/dev/null

    run env HOME="$HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh"

    [ "$status" -eq 0 ]
    [[ "$output" == *"Total"* ]]
}

@test "bw size --json total_bytes matches sum of individual sizes" {
    mkdir -p "$HOME/.npm"
    mkdir -p "$HOME/.cargo"
    dd if=/dev/zero of="$HOME/.npm/data" bs=1024 count=100 2>/dev/null
    dd if=/dev/zero of="$HOME/.cargo/data" bs=1024 count=200 2>/dev/null

    run env HOME="$HOME" TERM="xterm-256color" \
        bash --noprofile --norc "$PROJECT_ROOT/bin/size.sh" --json

    [ "$status" -eq 0 ]
    echo "$output" | python3 -c "
import sys, json
data = json.load(sys.stdin)
total = sum(c['size_bytes'] for c in data['caches'])
assert data['total_bytes'] == total, f'total mismatch: {data[\"total_bytes\"]} != {total}'
"
}
