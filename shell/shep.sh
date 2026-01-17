#!/bin/bash

# Shep - Laravel Worktree Manager Shell Wrapper
# This wrapper enables the 'cd' functionality after creating a worktree
#
# Installation:
#   Add this to your ~/.zshrc or ~/.bashrc:
#   source ~/.composer/vendor/shep/shep/shell/shep.sh
#
# Or if installed locally:
#   source /path/to/shepherd/shell/shep.sh

shep() {
    if [[ "$1" == "new" && -n "$2" ]]; then
        # For 'new' command, capture output to get the worktree path
        local output
        output=$(command shep "$@")
        local exit_code=$?

        if [[ $exit_code -eq 0 ]]; then
            # Print the output (which includes info messages)
            echo "$output"

            # The last line of output is the worktree path
            local worktree_path
            worktree_path=$(echo "$output" | tail -n 1)

            # Only cd if the path exists and is a directory
            if [[ -d "$worktree_path" ]]; then
                cd "$worktree_path" || return 1
            fi
        else
            echo "$output"
            return $exit_code
        fi
    else
        # For all other commands, just pass through
        command shep "$@"
    fi
}
