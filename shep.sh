#!/bin/bash

# Shep - Laravel Worktree Manager
# A shell script for managing Git worktrees in Laravel projects

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
DIM='\033[2m'
NC='\033[0m' # No Color

# Print functions
error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

success() {
    echo -e "${GREEN}$1${NC}"
}

info() {
    echo -e "${CYAN}$1${NC}"
}

dim() {
    echo -e "${DIM}$1${NC}"
}

# Check if we're in a git repo
check_git_repo() {
    if ! git rev-parse --show-toplevel &>/dev/null; then
        error "Not in a git repository."
        return 1
    fi
}

# Get the repo root
get_repo_root() {
    git rev-parse --show-toplevel
}

# Get worktree path for a branch
get_worktree_path() {
    local branch="$1"
    echo "$(get_repo_root)/.worktrees/$branch"
}

# Check if branch exists
branch_exists() {
    local branch="$1"
    git show-ref --verify --quiet "refs/heads/$branch"
}

# Check if worktree exists
worktree_exists() {
    local branch="$1"
    local path
    path=$(get_worktree_path "$branch")
    [[ -d "$path" ]]
}

# Get Herd site name for a branch
get_herd_site_name() {
    local branch="$1"
    echo "pushsilver-$branch"
}

# Link worktree to Herd
link_to_herd() {
    local worktree_path="$1"
    local branch="$2"
    local site_name
    site_name=$(get_herd_site_name "$branch")

    if ! command -v herd &>/dev/null; then
        error "Herd CLI not found. Skipping Herd setup."
        return 1
    fi

    # Link the site (run from worktree directory)
    (cd "$worktree_path" && herd link "$site_name")
}

# Unlink worktree from Herd
unlink_from_herd() {
    local branch="$1"
    local site_name
    site_name=$(get_herd_site_name "$branch")

    if ! command -v herd &>/dev/null; then
        return 0
    fi

    herd unlink "$site_name" 2>/dev/null || true
}

# Setup environment for Laravel
setup_environment() {
    local worktree_path="$1"

    # Copy .env.example to .env if needed
    if [[ ! -f "$worktree_path/.env" && -f "$worktree_path/.env.example" ]]; then
        cp "$worktree_path/.env.example" "$worktree_path/.env"
    fi

    # Create database directory if needed
    mkdir -p "$worktree_path/database"

    # Create SQLite database
    local db_path="$worktree_path/database/database.sqlite"
    touch "$db_path"

    # Update .env if it exists
    if [[ -f "$worktree_path/.env" ]]; then
        # Update DB_CONNECTION
        if grep -q "^DB_CONNECTION=" "$worktree_path/.env"; then
            sed -i '' 's/^DB_CONNECTION=.*/DB_CONNECTION=sqlite/' "$worktree_path/.env"
        else
            echo "DB_CONNECTION=sqlite" >> "$worktree_path/.env"
        fi

        # Update DB_DATABASE
        if grep -q "^DB_DATABASE=" "$worktree_path/.env"; then
            sed -i '' "s|^DB_DATABASE=.*|DB_DATABASE=$db_path|" "$worktree_path/.env"
        else
            echo "DB_DATABASE=$db_path" >> "$worktree_path/.env"
        fi

        # Comment out unused DB settings
        sed -i '' 's/^DB_HOST=/#DB_HOST=/' "$worktree_path/.env"
        sed -i '' 's/^DB_PORT=/#DB_PORT=/' "$worktree_path/.env"
        sed -i '' 's/^DB_USERNAME=/#DB_USERNAME=/' "$worktree_path/.env"
        sed -i '' 's/^DB_PASSWORD=/#DB_PASSWORD=/' "$worktree_path/.env"
    fi
}

# Confirm prompt
confirm() {
    local prompt="$1"
    local default="${2:-n}"

    if [[ "$default" == "y" ]]; then
        prompt="$prompt [Y/n] "
    else
        prompt="$prompt [y/N] "
    fi

    read -r -p "$prompt" response
    response=${response:-$default}

    [[ "$response" =~ ^[Yy]$ ]]
}

# Command: new
cmd_new() {
    local branch="$1"

    if [[ -z "$branch" ]]; then
        error "Branch name required."
        echo "Usage: shep new <branch>"
        return 1
    fi

    check_git_repo || return 1

    if worktree_exists "$branch"; then
        error "Worktree for branch '$branch' already exists."
        return 1
    fi

    # Create branch if it doesn't exist
    if ! branch_exists "$branch"; then
        if confirm "Branch '$branch' does not exist. Create it?" "y"; then
            info "Creating branch '$branch'..."
            git branch "$branch"
        else
            echo "Aborted."
            return 0
        fi
    fi

    local worktree_path
    worktree_path=$(get_worktree_path "$branch")

    # Create worktree
    info "Creating worktree for '$branch'..."
    git worktree add "$worktree_path" "$branch"

    # Setup environment
    info "Setting up environment..."
    setup_environment "$worktree_path"

    success "Worktree created at: $worktree_path"

    # Output path for shell wrapper to cd into
    echo "$worktree_path"
}

# Command: remove
cmd_remove() {
    local branch="$1"

    if [[ -z "$branch" ]]; then
        error "Branch name required."
        echo "Usage: shep remove <branch>"
        return 1
    fi

    check_git_repo || return 1

    if ! worktree_exists "$branch"; then
        error "Worktree for branch '$branch' does not exist."
        return 1
    fi

    local worktree_path
    worktree_path=$(get_worktree_path "$branch")

    if confirm "Remove worktree at '$worktree_path'?" "n"; then
        info "Removing worktree '$branch'..."
        git worktree remove "$worktree_path" --force
        git worktree prune
        success "Worktree '$branch' removed."
    else
        echo "Aborted."
    fi
}

# Command: list (worktrees)
cmd_list() {
    check_git_repo || return 1

    # Get worktree list
    local worktrees
    worktrees=$(git worktree list)

    if [[ -z "$worktrees" ]]; then
        info "No worktrees found."
        return 0
    fi

    # Print header
    printf "\n"
    printf "${DIM}%-20s %-50s %s${NC}\n" "Branch" "Path" "HEAD"
    printf "${DIM}%-20s %-50s %s${NC}\n" "------" "----" "----"

    # Parse and print worktrees
    while IFS= read -r line; do
        local path branch head
        path=$(echo "$line" | awk '{print $1}')
        head=$(echo "$line" | awk '{print $2}')
        branch=$(echo "$line" | awk '{print $3}' | tr -d '[]')

        if [[ -z "$branch" ]]; then
            branch="(detached)"
        fi

        printf "%-20s %-50s %s\n" "$branch" "$path" "$head"
    done <<< "$worktrees"
    printf "\n"
}

# Command: help
cmd_help() {
    cat << 'EOF'
Shep - Laravel Worktree Manager

Usage: shep <command> [arguments]

Commands:
  new <branch>      Create a new worktree for a branch
  remove <branch>   Remove a worktree
  list              List all worktrees
  help              Show this help message

Examples:
  shep new feature-login    Create worktree for feature-login branch
  shep remove feature-login Remove the worktree
  shep list                 Show all worktrees

Installation:
  Add this to your ~/.zshrc or ~/.bashrc:

    source /path/to/shep.sh

  This enables automatic 'cd' into new worktrees.
EOF
}

# Main shep function (used when sourced)
shep() {
    local cmd="${1:-help}"
    shift 2>/dev/null || true

    case "$cmd" in
        new)
            local output
            output=$(cmd_new "$@")
            local exit_code=$?

            if [[ $exit_code -eq 0 ]]; then
                echo "$output"
                # Last line is the worktree path
                local worktree_path
                worktree_path=$(echo "$output" | tail -n 1)
                if [[ -d "$worktree_path" ]]; then
                    cd "$worktree_path" || return 1
                fi
            else
                echo "$output"
                return $exit_code
            fi
            ;;
        remove)
            cmd_remove "$@"
            ;;
        list|ls)
            cmd_list
            ;;
        help|--help|-h)
            cmd_help
            ;;
        *)
            error "Unknown command: $cmd"
            cmd_help
            return 1
            ;;
    esac
}

# If script is executed directly (not sourced), run the command
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cmd="${1:-help}"
    shift 2>/dev/null || true

    case "$cmd" in
        new)
            cmd_new "$@"
            ;;
        remove)
            cmd_remove "$@"
            ;;
        list|ls)
            cmd_list
            ;;
        help|--help|-h)
            cmd_help
            ;;
        *)
            error "Unknown command: $cmd"
            cmd_help
            exit 1
            ;;
    esac
fi
