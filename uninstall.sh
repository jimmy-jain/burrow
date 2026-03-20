#!/bin/bash
# Burrow - Standalone uninstaller.
# Works independently of the main burrow installation, so it can clean up
# broken installs where "bw remove" fails due to missing libraries.

set -euo pipefail

GREEN='\033[38;5;108m'
YELLOW='\033[38;5;214m'
RED='\033[38;5;167m'
NC='\033[0m'

ICON_SUCCESS="✓"
ICON_ERROR="✗"
ICON_LIST="›"
ICON_ARROW="›"

DRY_RUN=false
FORCE=false

usage() {
    echo "Usage: uninstall.sh [--dry-run] [--force] [--help]"
    echo ""
    echo "Options:"
    echo "  --dry-run   Preview what would be removed without deleting anything"
    echo "  --force     Skip confirmation prompt"
    echo "  --help      Show this help message"
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --force)
            FORCE=true
            shift
            ;;
        --help | -h)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac
done

maybe_sudo_rm() {
    local path="$1"
    if [[ ! -e "$path" && ! -L "$path" ]]; then
        return 0
    fi
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "  ${YELLOW}${ICON_LIST}${NC} Would remove: $path"
        return 0
    fi
    if [[ ! -w "$(dirname "$path")" ]]; then
        sudo rm -f "$path" 2> /dev/null
    else
        rm -f "$path" 2> /dev/null
    fi
}

maybe_sudo_rm_rf() {
    local path="$1"
    if [[ ! -e "$path" ]]; then
        return 0
    fi
    if [[ "$DRY_RUN" == "true" ]]; then
        echo -e "  ${YELLOW}${ICON_LIST}${NC} Would remove: $path"
        return 0
    fi
    rm -rf "$path" 2> /dev/null || true
}

# Detect installations
echo "Detecting Burrow installations..."

is_homebrew=false
brew_cmd=""
if command -v brew > /dev/null 2>&1; then
    brew_cmd="brew"
elif [[ -x "/opt/homebrew/bin/brew" ]]; then
    brew_cmd="/opt/homebrew/bin/brew"
elif [[ -x "/usr/local/bin/brew" ]]; then
    brew_cmd="/usr/local/bin/brew"
fi

if [[ -n "$brew_cmd" ]] && "$brew_cmd" list burrow > /dev/null 2>&1; then
    is_homebrew=true
fi

# Find manual installs
declare -a manual_installs=()
declare -a alias_installs=()

for path in "/usr/local/bin/burrow" "$HOME/.local/bin/burrow" "/opt/local/bin/burrow"; do
    if [[ -f "$path" ]]; then
        if [[ ! -L "$path" ]] || ! readlink "$path" | grep -q "Cellar/burrow"; then
            manual_installs+=("$path")
        fi
    fi
done

# Also check PATH
found_burrow=$(command -v burrow 2> /dev/null || true)
if [[ -n "$found_burrow" && -f "$found_burrow" ]]; then
    if [[ ! -L "$found_burrow" ]] || ! readlink "$found_burrow" | grep -q "Cellar/burrow"; then
        local_exists=false
        for existing in "${manual_installs[@]+"${manual_installs[@]}"}"; do
            [[ "$existing" == "$found_burrow" ]] && local_exists=true
        done
        [[ "$local_exists" == "false" ]] && manual_installs+=("$found_burrow")
    fi
fi

for path in "/usr/local/bin/bw" "$HOME/.local/bin/bw" "/opt/local/bin/bw"; do
    if [[ -f "$path" ]]; then
        if [[ ! -L "$path" ]] || ! readlink "$path" | grep -q "Cellar/burrow"; then
            alias_installs+=("$path")
        fi
    fi
done

found_mo=$(command -v bw 2> /dev/null || true)
if [[ -n "$found_mo" && -f "$found_mo" ]]; then
    if [[ ! -L "$found_mo" ]] || ! readlink "$found_mo" | grep -q "Cellar/burrow"; then
        local_exists=false
        for existing in "${alias_installs[@]+"${alias_installs[@]}"}"; do
            [[ "$existing" == "$found_mo" ]] && local_exists=true
        done
        [[ "$local_exists" == "false" ]] && alias_installs+=("$found_mo")
    fi
fi

manual_count=${#manual_installs[@]}
alias_count=${#alias_installs[@]}
has_config=false
has_cache=false
[[ -d "$HOME/.config/burrow" ]] && has_config=true
[[ -d "$HOME/.cache/burrow" ]] && has_cache=true

if [[ "$is_homebrew" == "false" && $manual_count -eq 0 && $alias_count -eq 0 && "$has_config" == "false" && "$has_cache" == "false" ]]; then
    echo -e "${YELLOW}No Burrow installation detected${NC}"
    exit 0
fi

# Show what will be removed
echo ""
if [[ "$DRY_RUN" == "true" ]]; then
    echo -e "${YELLOW}DRY RUN${NC} — no files will be removed"
    echo ""
fi

echo -e "The following will be removed:"
[[ "$is_homebrew" == "true" ]] && echo -e "  ${ICON_LIST} Burrow via Homebrew"
for install in ${manual_installs[@]+"${manual_installs[@]}"}; do
    echo -e "  ${ICON_LIST} $install"
done
for alias in ${alias_installs[@]+"${alias_installs[@]}"}; do
    echo -e "  ${ICON_LIST} $alias"
done
[[ "$has_config" == "true" ]] && echo -e "  ${ICON_LIST} ~/.config/burrow"
[[ "$has_cache" == "true" ]] && echo -e "  ${ICON_LIST} ~/.cache/burrow"

if [[ "$DRY_RUN" == "true" ]]; then
    echo ""
    echo -e "${GREEN}${ICON_SUCCESS}${NC} Dry run complete, no changes made"
    exit 0
fi

# Confirm unless --force
if [[ "$FORCE" != "true" ]]; then
    echo ""
    echo -ne "${ICON_ARROW} Press Enter to confirm, Ctrl+C to cancel: "
    read -r
fi

# Perform removal
has_error=false

if [[ "$is_homebrew" == "true" ]]; then
    if [[ -n "$brew_cmd" ]]; then
        echo "Removing Homebrew installation..."
        if ! "$brew_cmd" uninstall --force burrow 2>&1; then
            has_error=true
            echo -e "${RED}Homebrew uninstall failed. Run manually: brew uninstall --force burrow${NC}"
        else
            echo -e "${GREEN}${ICON_SUCCESS}${NC} Removed via Homebrew"
        fi
    fi
fi

for install in ${manual_installs[@]+"${manual_installs[@]}"}; do
    if maybe_sudo_rm "$install"; then
        echo -e "${GREEN}${ICON_SUCCESS}${NC} Removed $install"
    else
        has_error=true
        echo -e "${RED}${ICON_ERROR}${NC} Failed to remove $install"
    fi
done

for alias in ${alias_installs[@]+"${alias_installs[@]}"}; do
    if maybe_sudo_rm "$alias"; then
        echo -e "${GREEN}${ICON_SUCCESS}${NC} Removed $alias"
    else
        has_error=true
        echo -e "${RED}${ICON_ERROR}${NC} Failed to remove $alias"
    fi
done

maybe_sudo_rm_rf "$HOME/.cache/burrow"
maybe_sudo_rm_rf "$HOME/.config/burrow"

echo ""
if [[ "$has_error" == "true" ]]; then
    echo -e "${YELLOW}${ICON_ERROR} Burrow uninstalled with some errors${NC}"
    exit 1
else
    echo -e "${GREEN}${ICON_SUCCESS} Burrow uninstalled successfully${NC}"
fi
