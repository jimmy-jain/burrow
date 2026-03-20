#!/bin/bash
# Burrow - Quick Launchers command.
# Installs/updates Raycast and Alfred script commands.
# Auto-updates launcher scripts when Burrow version changes.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../lib/core/common.sh"

LAUNCHER_DIR="$HOME/Library/Application Support/Raycast/script-commands"
LAUNCHER_VERSION_FILE="$HOME/.config/burrow/launcher_version"
SETUP_SCRIPT="$SCRIPT_DIR/../scripts/setup-quick-launchers.sh"

# Get a hash of the setup script to detect changes.
_launcher_script_hash() {
    if [[ -f "$SETUP_SCRIPT" ]]; then
        /usr/bin/shasum -a 256 "$SETUP_SCRIPT" 2>/dev/null | awk '{print $1}' | head -c 12
    else
        echo "none"
    fi
}

# Check if launchers need updating.
_launchers_need_update() {
    # No launcher dir = never installed
    [[ ! -d "$LAUNCHER_DIR" ]] && return 0

    # No version file = never tracked
    [[ ! -f "$LAUNCHER_VERSION_FILE" ]] && return 0

    local current_hash
    current_hash=$(_launcher_script_hash)
    local saved_hash
    saved_hash=$(cat "$LAUNCHER_VERSION_FILE" 2>/dev/null || echo "")

    [[ "$current_hash" != "$saved_hash" ]]
}

# Save current version hash after install.
_save_launcher_version() {
    local hash
    hash=$(_launcher_script_hash)
    mkdir -p "$(dirname "$LAUNCHER_VERSION_FILE")"
    echo "$hash" > "$LAUNCHER_VERSION_FILE"
}

# Auto-update check (called from other commands silently).
# Returns 0 if updated, 1 if no update needed.
launcher_auto_update() {
    # Skip if no launchers installed
    [[ ! -d "$LAUNCHER_DIR" ]] && return 1

    # Skip if no setup script available
    [[ ! -f "$SETUP_SCRIPT" ]] && return 1

    if _launchers_need_update; then
        bash "$SETUP_SCRIPT" 2>/dev/null
        _save_launcher_version
        return 0
    fi
    return 1
}

show_help() {
    echo -e "${BOLD}Burrow Quick Launchers${NC}"
    echo ""
    echo -e "Usage: bw launchers [options]"
    echo ""
    echo -e "Options:"
    echo "  install       Install Raycast + Alfred commands (default)"
    echo "  update        Force update launcher scripts"
    echo "  status        Check if launchers are installed and current"
    echo "  --help        Show this help message"
}

show_status() {
    echo -e "${BOLD}Launcher Status${NC}"
    echo ""

    if [[ -d "$LAUNCHER_DIR" ]]; then
        local count
        count=$(ls "$LAUNCHER_DIR"/burrow-*.sh 2>/dev/null | wc -l | tr -d ' ')
        echo -e "  ${GREEN}${ICON_SUCCESS}${NC} $count Raycast commands installed"
        echo -e "  ${GRAY}${ICON_SUBLIST} $LAUNCHER_DIR${NC}"

        if [[ -f "$LAUNCHER_VERSION_FILE" ]]; then
            if _launchers_need_update; then
                echo -e "  ${YELLOW}${ICON_WARNING}${NC} Update available — run ${BOLD}bw launchers update${NC}"
            else
                echo -e "  ${GREEN}${ICON_SUCCESS}${NC} Up to date"
            fi
        fi
    else
        echo -e "  ${GRAY}${ICON_WARNING}${NC} Not installed — run ${BOLD}bw launchers${NC}"
    fi
}

do_install() {
    if [[ ! -f "$SETUP_SCRIPT" ]]; then
        echo -e "${RED}${ICON_ERROR}${NC} Setup script not found at $SETUP_SCRIPT"
        exit 1
    fi

    bash "$SETUP_SCRIPT"
    _save_launcher_version
}

main() {
    local action="${1:-install}"

    case "$action" in
        install)
            do_install
            ;;
        update)
            echo -e "${BOLD}Updating launcher scripts...${NC}"
            do_install
            echo -e "${GREEN}${ICON_SUCCESS}${NC} Launchers updated"
            ;;
        status)
            show_status
            ;;
        --help | -h | help)
            show_help
            ;;
        *)
            echo "Unknown option: $action"
            show_help
            exit 1
            ;;
    esac
}

if [[ "${BURROW_SKIP_MAIN:-0}" == "1" ]]; then
    if [[ "${BASH_SOURCE[0]}" != "$0" ]]; then
        return 0
    else
        exit 0
    fi
fi

main "$@"
