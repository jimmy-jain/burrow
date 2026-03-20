#!/bin/bash
# Burrow - Clean Section Registry
# Defines selectable cleanup sections with estimate and execute functions.
# Used by bin/clean.sh for interactive section selection.

set -euo pipefail

if [[ -n "${BURROW_SECTIONS_LOADED:-}" ]]; then
    return 0
fi
readonly BURROW_SECTIONS_LOADED=1

# ============================================================================
# Section Registry
# Parallel arrays (Bash 3.2 compatible — no associative arrays)
# ============================================================================

CLEAN_SECTION_IDS=(
    "system"
    "user"
    "app_caches"
    "browsers"
    "cloud_office"
    "dev_tools"
    "applications"
    "virtualization"
    "app_support"
    "orphaned"
    "apple_silicon"
    "device_backups"
    "time_machine"
    "large_files"
)

CLEAN_SECTION_LABELS=(
    "System (sudo)"
    "User essentials"
    "App caches"
    "Browsers"
    "Cloud & Office"
    "Developer tools"
    "Applications"
    "Virtualization"
    "Application Support"
    "Orphaned data"
    "Apple Silicon"
    "Device backups"
    "Time Machine"
    "Large files"
)

# ============================================================================
# Helpers
# ============================================================================

# Look up label by section ID.
get_section_label() {
    local target_id="$1"
    local i
    for ((i = 0; i < ${#CLEAN_SECTION_IDS[@]}; i++)); do
        if [[ "${CLEAN_SECTION_IDS[i]}" == "$target_id" ]]; then
            echo "${CLEAN_SECTION_LABELS[i]}"
            return 0
        fi
    done
    echo ""
}

# Build a filtered list of section IDs based on current context.
# Sets global BURROW_SELECTABLE_SECTIONS array.
build_selectable_sections() {
    BURROW_SELECTABLE_SECTIONS=()
    local i
    for ((i = 0; i < ${#CLEAN_SECTION_IDS[@]}; i++)); do
        local id="${CLEAN_SECTION_IDS[i]}"
        case "$id" in
            system)
                [[ "${SYSTEM_CLEAN:-false}" == "true" ]] && BURROW_SELECTABLE_SECTIONS+=("$id")
                ;;
            apple_silicon)
                [[ "${IS_M_SERIES:-false}" == "true" ]] && BURROW_SELECTABLE_SECTIONS+=("$id")
                ;;
            *)
                BURROW_SELECTABLE_SECTIONS+=("$id")
                ;;
        esac
    done
}

# Generate a comma-separated string of all indices "0,1,2,...,N-1".
build_preselect_all() {
    local count="$1"
    local result=""
    local i
    for ((i = 0; i < count; i++)); do
        [[ -n "$result" ]] && result+=","
        result+="$i"
    done
    echo "$result"
}

# ============================================================================
# Size Estimation Functions
# Each returns estimated reclaimable KB on stdout.
# Uses hint_get_path_size_kb_with_timeout for bounded scanning.
# ============================================================================

_estimate_paths() {
    local total_kb=0
    local path
    for path in "$@"; do
        [[ -e "$path" ]] || continue
        local kb=""
        if kb=$(hint_get_path_size_kb_with_timeout "$path" 0.5 2>/dev/null); then
            if [[ "$kb" =~ ^[0-9]+$ ]]; then
                total_kb=$((total_kb + kb))
            fi
        fi
    done
    echo "$total_kb"
}

estimate_section_system() {
    _estimate_paths \
        /Library/Caches \
        /Library/Logs \
        /private/var/log \
        /private/tmp \
        /private/var/tmp
}

estimate_section_user() {
    _estimate_paths \
        "$HOME/Library/Caches" \
        "$HOME/Library/Logs" \
        "$HOME/.Trash"
}

estimate_section_app_caches() {
    _estimate_paths \
        "$HOME/Library/Containers"
}

estimate_section_browsers() {
    _estimate_paths \
        "$HOME/Library/Caches/com.apple.Safari" \
        "$HOME/Library/Caches/Google" \
        "$HOME/Library/Caches/Firefox" \
        "$HOME/Library/Caches/com.microsoft.edgemac" \
        "$HOME/Library/Caches/BraveSoftware" \
        "$HOME/Library/Caches/com.operasoftware.Opera"
}

estimate_section_cloud_office() {
    _estimate_paths \
        "$HOME/Library/Caches/com.dropbox.DropboxMacUpdate" \
        "$HOME/Library/Caches/com.google.GoogleDrive" \
        "$HOME/Library/Caches/com.microsoft.OneDrive" \
        "$HOME/Library/Caches/com.microsoft.Word" \
        "$HOME/Library/Caches/com.microsoft.Excel" \
        "$HOME/Library/Caches/com.microsoft.Outlook"
}

estimate_section_dev_tools() {
    _estimate_paths \
        "$HOME/.npm" \
        "$HOME/.pnpm-store" \
        "$HOME/.yarn" \
        "$HOME/.cargo/registry" \
        "$HOME/.cache/pip" \
        "$HOME/Library/Developer/Xcode/DerivedData" \
        "$HOME/Library/Developer/Xcode/Archives" \
        "$HOME/Library/Developer/CoreSimulator"
}

estimate_section_applications() {
    _estimate_paths \
        "$HOME/Library/Caches/com.spotify.client" \
        "$HOME/Library/Application Support/Steam/appcache" \
        "$HOME/Library/Caches/com.apple.Music"
}

estimate_section_virtualization() {
    _estimate_paths \
        "$HOME/Library/Caches/com.vmware.fusion" \
        "$HOME/Library/Caches/com.parallels.desktop.console" \
        "$HOME/.docker"
}

estimate_section_app_support() {
    _estimate_paths \
        "$HOME/Library/Application Support/CrashReporter" \
        "$HOME/Library/Logs/DiagnosticReports"
}

estimate_section_orphaned() {
    _estimate_paths \
        "$HOME/Library/Saved Application State" \
        "$HOME/Library/Cookies"
}

estimate_section_apple_silicon() {
    _estimate_paths \
        "$HOME/Library/Developer/CoreSimulator/Caches"
}

estimate_section_device_backups() {
    _estimate_paths \
        "$HOME/Library/Application Support/MobileSync/Backup"
}

estimate_section_time_machine() {
    # Time Machine snapshots are hard to estimate; return 0
    echo "0"
}

estimate_section_large_files() {
    # Large file scan is discovery-based; return 0
    echo "0"
}

# ============================================================================
# Parallel Scan Orchestrator
# Runs all estimate functions concurrently with a global timeout.
# Results stored in CLEAN_SECTION_SIZES[] parallel to the section IDs.
# ============================================================================

CLEAN_SECTION_SIZES=()

scan_all_sections() {
    build_selectable_sections
    local -a section_ids=("${BURROW_SELECTABLE_SECTIONS[@]}")

    local scan_dir
    scan_dir=$(mktemp -d "${TMPDIR:-/tmp}/burrow-scan.XXXXXX")

    local id
    for id in "${section_ids[@]}"; do
        (
            local kb
            kb=$(estimate_section_"$id" 2>/dev/null) || kb=0
            echo "$kb" > "$scan_dir/$id"
        ) &
    done

    # Wait with global timeout (10s)
    local wait_start
    wait_start=$(date +%s)
    while [[ $(jobs -r -p | wc -l) -gt 0 ]]; do
        local now
        now=$(date +%s)
        if [[ $((now - wait_start)) -ge 10 ]]; then
            # Kill remaining jobs
            jobs -r -p | while read -r pid; do
                kill "$pid" 2>/dev/null || true
            done
            break
        fi
        sleep 0.2
    done
    wait 2>/dev/null || true

    # Collect results
    CLEAN_SECTION_SIZES=()
    for id in "${section_ids[@]}"; do
        local kb=0
        if [[ -f "$scan_dir/$id" ]]; then
            kb=$(cat "$scan_dir/$id" 2>/dev/null) || kb=0
            [[ "$kb" =~ ^[0-9]+$ ]] || kb=0
        fi
        CLEAN_SECTION_SIZES+=("$kb")
    done

    rm -rf "$scan_dir"
}

# ============================================================================
# Execute Functions
# Each wraps the existing cleanup calls with start_section/end_section.
# ============================================================================

run_section_system() {
    start_section "System"
    clean_deep_system
    clean_local_snapshots
    end_section
}

run_section_user() {
    start_section "User essentials"
    clean_user_essentials
    clean_finder_metadata
    scan_external_volumes
    end_section
}

run_section_app_caches() {
    start_section "App caches"
    clean_app_caches
    end_section
}

run_section_browsers() {
    start_section "Browsers"
    clean_browsers
    end_section
}

run_section_cloud_office() {
    start_section "Cloud & Office"
    clean_cloud_storage
    clean_office_applications
    end_section
}

run_section_dev_tools() {
    start_section "Developer tools"
    clean_developer_tools
    end_section
}

run_section_applications() {
    start_section "Applications"
    clean_user_gui_applications
    end_section
}

run_section_virtualization() {
    start_section "Virtualization"
    clean_virtualization_tools
    end_section
}

run_section_app_support() {
    start_section "Application Support"
    clean_application_support_logs
    end_section
}

run_section_orphaned() {
    start_section "Orphaned data"
    clean_orphaned_app_data
    clean_orphaned_system_services
    show_user_launch_agent_hint_notice
    end_section
}

run_section_apple_silicon() {
    clean_apple_silicon_caches
}

run_section_device_backups() {
    start_section "Device backups"
    check_ios_device_backups
    end_section
}

run_section_time_machine() {
    start_section "Time Machine"
    clean_time_machine_failed_backups
    end_section
}

run_section_large_files() {
    start_section "Large files"
    check_large_file_candidates
    end_section
}

# Run hint-only sections (always shown, not selectable).
run_hint_sections() {
    start_section "System Data clues"
    show_system_data_hint_notice
    end_section

    start_section "Project artifacts"
    show_project_artifact_hint_notice
    end_section
}

# Execute a list of section IDs in order.
execute_selected_sections() {
    local -a selected_ids=("$@")

    local had_errexit=0
    [[ $- == *e* ]] && had_errexit=1
    set +e

    local id
    for id in "${selected_ids[@]}"; do
        "run_section_${id}"
    done

    run_hint_sections

    [[ $had_errexit -eq 1 ]] && set -e
    return 0
}
