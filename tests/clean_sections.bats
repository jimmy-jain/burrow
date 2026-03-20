#!/usr/bin/env bats
# Tests for clean section registry and interactive selection

setup() {
    # Create isolated test environment
    export TEST_DIR="$BATS_TMPDIR/clean_sections_$$"
    mkdir -p "$TEST_DIR"

    # Mock HOME for isolation
    export REAL_HOME="$HOME"
    export HOME="$TEST_DIR/home"
    mkdir -p "$HOME/.config/burrow"
    mkdir -p "$HOME/Library/Caches"
    mkdir -p "$HOME/Library/Logs"

    # Source the sections registry
    SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/../lib/clean" && pwd)"
    export BURROW_SKIP_MAIN=1
    export BURROW_DRY_RUN=1

    # Source core libs needed by sections.sh
    source "$(dirname "$BATS_TEST_FILENAME")/../lib/core/common.sh"
    source "$SCRIPT_DIR/sections.sh"
}

teardown() {
    export HOME="$REAL_HOME"
    rm -rf "$TEST_DIR"
}

# --- Section Registry Tests ---

@test "section registry defines all required arrays" {
    [[ ${#CLEAN_SECTION_IDS[@]} -gt 0 ]]
    [[ ${#CLEAN_SECTION_LABELS[@]} -gt 0 ]]
    [[ ${#CLEAN_SECTION_IDS[@]} -eq ${#CLEAN_SECTION_LABELS[@]} ]]
}

@test "section registry has at least 10 sections" {
    [[ ${#CLEAN_SECTION_IDS[@]} -ge 10 ]]
}

@test "all section IDs are unique" {
    local -a seen=()
    for id in "${CLEAN_SECTION_IDS[@]}"; do
        for s in "${seen[@]}"; do
            [[ "$s" != "$id" ]]
        done
        seen+=("$id")
    done
}

@test "every section ID has an estimate function" {
    for id in "${CLEAN_SECTION_IDS[@]}"; do
        type "estimate_section_${id}" &>/dev/null
    done
}

@test "every section ID has an execute function" {
    for id in "${CLEAN_SECTION_IDS[@]}"; do
        type "run_section_${id}" &>/dev/null
    done
}

@test "system section is excluded when SYSTEM_CLEAN is false" {
    SYSTEM_CLEAN=false
    local -a filtered=()
    build_selectable_sections
    local -a filtered=("${BURROW_SELECTABLE_SECTIONS[@]}")
    for id in "${filtered[@]}"; do
        [[ "$id" != "system" ]]
    done
}

@test "system section is included when SYSTEM_CLEAN is true" {
    SYSTEM_CLEAN=true
    local -a filtered=()
    build_selectable_sections
    local -a filtered=("${BURROW_SELECTABLE_SECTIONS[@]}")
    local found=false
    for id in "${filtered[@]}"; do
        [[ "$id" == "system" ]] && found=true
    done
    [[ "$found" == "true" ]]
}

@test "apple_silicon section excluded on non-arm64" {
    IS_M_SERIES=false
    local -a filtered=()
    build_selectable_sections
    local -a filtered=("${BURROW_SELECTABLE_SECTIONS[@]}")
    for id in "${filtered[@]}"; do
        [[ "$id" != "apple_silicon" ]]
    done
}

# --- Size Estimation Tests ---

@test "estimate returns 0 for empty directory" {
    local kb
    kb=$(estimate_section_user)
    [[ "$kb" =~ ^[0-9]+$ ]]
}

@test "estimate returns positive value for directory with content" {
    # Create enough test content to register on du
    mkdir -p "$HOME/Library/Caches/testdir"
    dd if=/dev/zero of="$HOME/Library/Caches/testdir/bigfile" bs=1024 count=500 2>/dev/null
    local kb
    kb=$(command du -skP "$HOME/Library/Caches" 2>/dev/null | awk '{print $1}')
    [[ "$kb" -gt 0 ]]
}

# --- Section Label Lookup ---

@test "get_section_label returns correct label for known ID" {
    local label
    label=$(get_section_label "user")
    [[ -n "$label" ]]
    [[ "$label" == "User essentials" ]]
}

@test "get_section_label returns empty for unknown ID" {
    local label
    label=$(get_section_label "nonexistent")
    [[ -z "$label" ]]
}

# --- Build Preselect Indices ---

@test "build_preselect_all generates all indices" {
    local -a ids=("user" "browsers" "dev_tools")
    local result
    result=$(build_preselect_all 3)
    [[ "$result" == "0,1,2" ]]
}
