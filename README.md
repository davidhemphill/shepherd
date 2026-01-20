# Shep

A CLI tool for managing Git worktrees in Laravel projects, with automatic Laravel Herd integration.

## Features

- Create and provision Git worktrees for feature branches
- Automatic Laravel Herd site linking with HTTPS
- Environment setup with SQLite database configuration
- Interactive prompts for common Laravel setup tasks

## Installation

### Using Go

```bash
go install github.com/davidhemphill/shepherd@latest
```

### Download Binary

Download the latest release from the [releases page](https://github.com/davidhemphill/shepherd/releases).

## Usage

```bash
# Create a new worktree for a branch
shep new feature-login

# Provision the current directory
shep init

# Provision an existing worktree
shep init feature-login

# Remove a worktree
shep remove feature-login

# List all worktrees
shep list
```

## What `shep new` Does

1. Creates the branch if it doesn't exist
2. Creates a Git worktree at `.worktrees/<branch>`
3. Copies `.env.example` to `.env`
4. Configures SQLite database
5. Runs `composer install`
6. Optionally generates application key
7. Optionally runs migrations with seeding
8. Links to Laravel Herd with HTTPS
9. Optionally starts `npm run dev`

## Worktree Structure

Worktrees are created in a `.worktrees` directory at your repo root:

```
my-project/
├── .worktrees/
│   ├── feature-login/
│   └── feature-dashboard/
├── app/
├── ...
```

## Herd Site Naming

Sites are named based on your project folder:

| Project Folder | Branch | Herd Site |
|----------------|--------|-----------|
| `myapp.dev` | `feature` | `myapp-feature.dev.test` |
| `myapp` | `feature` | `myapp-feature.test` |

## Requirements

- Git
- Go 1.21+ (for installation from source)
- [Laravel Herd](https://herd.laravel.com/) (optional, for automatic site linking)
- Composer (for Laravel dependency installation)

## License

MIT
