#!/bin/bash
# Burrow - Report module
# Combines system info into a single JSON document.

set -euo pipefail

# Prevent multiple sourcing
if [[ -n "${BURROW_REPORT_LOADED:-}" ]]; then
    return 0
fi
readonly BURROW_REPORT_LOADED=1

# Escape a string for safe JSON embedding.
# Handles backslashes, double quotes, newlines, tabs, and carriage returns.
_json_escape() {
    local str="$1"
    str="${str//\\/\\\\}"
    str="${str//\"/\\\"}"
    str="${str//$'\n'/\\n}"
    str="${str//$'\t'/\\t}"
    str="${str//$'\r'/\\r}"
    printf '%s' "$str"
}

# Collect the macOS marketing version (e.g. "15.3").
_get_os_version() {
    local ver=""
    if command -v sw_vers > /dev/null 2>&1; then
        ver=$(sw_vers -productVersion 2> /dev/null || true)
    fi
    if [[ -z "$ver" ]]; then
        ver="unknown"
    fi
    printf '%s' "$ver"
}

# Collect the hardware architecture (arm64 / x86_64).
_get_architecture() {
    uname -m 2> /dev/null || printf 'unknown'
}

# Collect free disk space as a human-readable string (e.g. "150GB").
_get_report_free_space() {
    if type get_free_space > /dev/null 2>&1; then
        get_free_space
    else
        local target="/"
        if [[ -d "/System/Volumes/Data" ]]; then
            target="/System/Volumes/Data"
        fi
        df -h "$target" | awk 'NR==2 {print $4}'
    fi
}

_get_report_cache_dir() {
    printf '%s\n' "${XDG_CACHE_HOME:-$HOME/.cache}/burrow"
}

_get_cleanup_potential_cache_file() {
    local cache_dir
    cache_dir=$(_get_report_cache_dir)
    printf '%s\n' "$cache_dir/report_cleanup_potential_v1"
}

_get_cleanup_potential_cache_ttl() {
    local ttl="${BW_REPORT_CLEANUP_CACHE_TTL:-300}"
    if [[ ! "$ttl" =~ ^[0-9]+$ ]]; then
        ttl=300
    fi
    printf '%s\n' "$ttl"
}

_read_cached_cleanup_potential() {
    if [[ "${BW_REPORT_DISABLE_CACHE:-0}" == "1" ]]; then
        return 1
    fi

    local ttl
    ttl=$(_get_cleanup_potential_cache_ttl)
    if [[ "$ttl" -eq 0 ]]; then
        return 1
    fi

    local cache_file
    cache_file=$(_get_cleanup_potential_cache_file)
    if [[ ! -r "$cache_file" ]]; then
        return 1
    fi

    local now mtime age
    now=$(get_epoch_seconds)
    mtime=$(get_file_mtime "$cache_file")
    if [[ ! "$now" =~ ^[0-9]+$ ]] || [[ ! "$mtime" =~ ^[0-9]+$ ]]; then
        return 1
    fi
    age=$((now - mtime))
    if [[ "$age" -lt 0 ]] || [[ "$age" -gt "$ttl" ]]; then
        return 1
    fi

    local cached_bytes="" cached_human=""
    IFS=$'\t' read -r cached_bytes cached_human < "$cache_file" || true
    if [[ ! "$cached_bytes" =~ ^[0-9]+$ ]] || [[ -z "$cached_human" ]]; then
        return 1
    fi

    printf '%s %s' "$cached_bytes" "$cached_human"
}

_write_cached_cleanup_potential() {
    if [[ "${BW_REPORT_DISABLE_CACHE:-0}" == "1" ]]; then
        return 0
    fi

    local bytes="$1"
    local human="$2"
    if [[ ! "$bytes" =~ ^[0-9]+$ ]] || [[ -z "$human" ]]; then
        return 0
    fi

    local cache_dir cache_file tmp_file
    cache_dir=$(_get_report_cache_dir)
    cache_file=$(_get_cleanup_potential_cache_file)
    tmp_file="${cache_file}.$$"

    ensure_user_dir "$cache_dir"

    if printf '%s\t%s\n' "$bytes" "$human" > "$tmp_file" 2> /dev/null; then
        mv "$tmp_file" "$cache_file" 2> /dev/null || cp "$tmp_file" "$cache_file" 2> /dev/null || true
        rm -f "$tmp_file" 2> /dev/null || true
    fi
}

# Try to determine total reclaimable bytes from `bw clean --dry-run`.
# Returns the byte count on stdout, or 0 if unavailable.
_get_cleanup_potential() {
    local potential_bytes=0
    local potential_human="0B"
    local cached=""

    cached=$(_read_cached_cleanup_potential || true)
    if [[ -n "$cached" ]]; then
        printf '%s' "$cached"
        return 0
    fi

    # Look for a 'bw' or 'burrow' binary relative to SCRIPT_DIR or on PATH.
    local bw_bin=""
    if [[ -n "${SCRIPT_DIR:-}" && -x "$SCRIPT_DIR/../burrow" ]]; then
        bw_bin="$SCRIPT_DIR/../burrow"
    elif [[ -n "${SCRIPT_DIR:-}" && -x "$SCRIPT_DIR/../bw" ]]; then
        bw_bin="$SCRIPT_DIR/../bw"
    elif command -v bw > /dev/null 2>&1; then
        bw_bin="bw"
    elif command -v burrow > /dev/null 2>&1; then
        bw_bin="burrow"
    fi

    if [[ -n "$bw_bin" ]]; then
        local dry_output=""
        dry_output=$("$bw_bin" clean --dry-run 2> /dev/null || true)

        # Try to extract "Potential space: X.XXGB" from the summary
        local match=""
        match=$(printf '%s\n' "$dry_output" | grep -o '[0-9][0-9]*\.[0-9]*GB' | tail -1 || true)
        if [[ -n "$match" ]]; then
            potential_human="$match"
            # Convert to approximate bytes (base-10 GB)
            local whole="${match%%.*}"
            local frac="${match#*.}"
            frac="${frac%GB}"
            # Pad or truncate fractional part to 2 digits
            while [[ ${#frac} -lt 2 ]]; do frac="${frac}0"; done
            frac="${frac:0:2}"
            potential_bytes=$(((whole * 100 + ${frac#0}) * 10000000))
        fi
    fi

    _write_cached_cleanup_potential "$potential_bytes" "$potential_human"
    printf '%s %s' "$potential_bytes" "$potential_human"
}

# Print usage information.
_report_usage() {
    printf '%s\n' "Usage: bw report [OPTIONS]"
    printf '\n'
    printf '%s\n' "Generate a JSON system report combining status, check, and cleanup data."
    printf '\n'
    printf '%s\n' "Options:"
    printf '%s\n' "  --out <file>    Write JSON to <file> instead of stdout"
    printf '%s\n' "  --help, -h      Show this help message"
}

# Main entry point.
report_main() {
    local out_file=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help | -h)
                _report_usage
                return 0
                ;;
            --out)
                if [[ $# -lt 2 ]]; then
                    printf 'Error: --out requires a file path argument\n' >&2
                    return 1
                fi
                out_file="$2"
                shift 2
                ;;
            *)
                printf 'Unknown option: %s\n' "$1" >&2
                _report_usage >&2
                return 1
                ;;
        esac
    done

    # Gather data
    local timestamp
    timestamp=$(date -u '+%Y-%m-%dT%H:%M:%SZ')

    local version="2.0.0"

    local hostname_val
    hostname_val=$(hostname -s 2> /dev/null || hostname 2> /dev/null || printf 'unknown')

    local os_version
    os_version=$(_get_os_version)

    local architecture
    architecture=$(_get_architecture)

    local free_space
    free_space=$(_get_report_free_space)

    local cleanup_potential
    cleanup_potential=$(_get_cleanup_potential)

    local cleanup_bytes="${cleanup_potential%% *}"
    local cleanup_human="${cleanup_potential#* }"

    # Build JSON manually (no jq dependency, Bash 3.2 compatible)
    local json=""
    json=$(printf '{\n')
    json+=$(printf '  "timestamp": "%s",\n' "$(_json_escape "$timestamp")")
    json+=$(printf '  "version": "%s",\n' "$(_json_escape "$version")")
    json+=$(printf '  "hostname": "%s",\n' "$(_json_escape "$hostname_val")")
    json+=$(printf '  "os_version": "%s",\n' "$(_json_escape "$os_version")")
    json+=$(printf '  "architecture": "%s",\n' "$(_json_escape "$architecture")")
    json+=$(printf '  "free_space": "%s",\n' "$(_json_escape "$free_space")")
    json+=$(printf '  "cleanup_potential_bytes": %s,\n' "$cleanup_bytes")
    json+=$(printf '  "cleanup_potential_human": "%s"\n' "$(_json_escape "$cleanup_human")")
    json+=$(printf '}\n')

    if [[ -n "$out_file" ]]; then
        printf '%s' "$json" > "$out_file"
    else
        printf '%s' "$json"
    fi
}
