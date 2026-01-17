<?php

namespace Shep;

use RuntimeException;

class WorktreeManager
{
    public function getRepoRoot(): ?string
    {
        $result = $this->exec('git rev-parse --show-toplevel 2>/dev/null');

        return $result['exitCode'] === 0 ? trim($result['output']) : null;
    }

    public function isGitRepo(): bool
    {
        return $this->getRepoRoot() !== null;
    }

    public function branchExists(string $branch): bool
    {
        $result = $this->exec("git show-ref --verify --quiet refs/heads/{$branch}");

        return $result['exitCode'] === 0;
    }

    public function createBranch(string $branch): bool
    {
        $result = $this->exec("git branch {$branch}");

        return $result['exitCode'] === 0;
    }

    public function getWorktreePath(string $branch): string
    {
        $repoRoot = $this->getRepoRoot();

        if ($repoRoot === null) {
            throw new RuntimeException('Not in a git repository');
        }

        return $repoRoot . '/.worktrees/' . $branch;
    }

    public function worktreeExists(string $branch): bool
    {
        return is_dir($this->getWorktreePath($branch));
    }

    public function createWorktree(string $branch): bool
    {
        $path = $this->getWorktreePath($branch);
        $result = $this->exec("git worktree add \"{$path}\" {$branch}");

        return $result['exitCode'] === 0;
    }

    public function removeWorktree(string $branch): bool
    {
        $path = $this->getWorktreePath($branch);
        $result = $this->exec("git worktree remove \"{$path}\" --force");

        if ($result['exitCode'] === 0) {
            $this->exec('git worktree prune');

            return true;
        }

        return false;
    }

    public function setupEnvironment(string $branch): void
    {
        $worktreePath = $this->getWorktreePath($branch);

        // Copy .env.example to .env if needed
        $envPath = $worktreePath . '/.env';
        $envExamplePath = $worktreePath . '/.env.example';

        if (! file_exists($envPath) && file_exists($envExamplePath)) {
            copy($envExamplePath, $envPath);
        }

        // Create database directory if it doesn't exist
        $databaseDir = $worktreePath . '/database';
        if (! is_dir($databaseDir)) {
            mkdir($databaseDir, 0755, true);
        }

        // Create SQLite database file
        $databasePath = $databaseDir . '/database.sqlite';
        if (! file_exists($databasePath)) {
            touch($databasePath);
        }

        // Update .env with SQLite configuration
        if (file_exists($envPath)) {
            $this->updateEnvFile($envPath, $databasePath);
        }
    }

    private function updateEnvFile(string $envPath, string $databasePath): void
    {
        $content = file_get_contents($envPath);

        // Update DB_CONNECTION to sqlite
        $content = preg_replace(
            '/^DB_CONNECTION=.*$/m',
            'DB_CONNECTION=sqlite',
            $content
        );

        // If DB_CONNECTION doesn't exist, add it
        if (! preg_match('/^DB_CONNECTION=/m', $content)) {
            $content .= "\nDB_CONNECTION=sqlite";
        }

        // Update or add DB_DATABASE
        if (preg_match('/^DB_DATABASE=.*$/m', $content)) {
            $content = preg_replace(
                '/^DB_DATABASE=.*$/m',
                'DB_DATABASE=' . $databasePath,
                $content
            );
        } else {
            $content .= "\nDB_DATABASE=" . $databasePath;
        }

        // Comment out other DB settings that aren't needed for SQLite
        $dbSettings = ['DB_HOST', 'DB_PORT', 'DB_USERNAME', 'DB_PASSWORD'];
        foreach ($dbSettings as $setting) {
            $content = preg_replace(
                '/^(' . $setting . '=.*)$/m',
                '# $1',
                $content
            );
        }

        file_put_contents($envPath, $content);
    }

    public function listWorktrees(): array
    {
        $result = $this->exec('git worktree list --porcelain');

        if ($result['exitCode'] !== 0) {
            return [];
        }

        $worktrees = [];
        $current = [];

        foreach (explode("\n", $result['output']) as $line) {
            if (empty($line)) {
                if (! empty($current)) {
                    $worktrees[] = $current;
                    $current = [];
                }

                continue;
            }

            if (str_starts_with($line, 'worktree ')) {
                $current['path'] = substr($line, 9);
            } elseif (str_starts_with($line, 'HEAD ')) {
                $current['head'] = substr($line, 5);
            } elseif (str_starts_with($line, 'branch ')) {
                $current['branch'] = basename(substr($line, 7));
            } elseif ($line === 'bare') {
                $current['bare'] = true;
            } elseif ($line === 'detached') {
                $current['detached'] = true;
            }
        }

        if (! empty($current)) {
            $worktrees[] = $current;
        }

        return $worktrees;
    }

    private function exec(string $command): array
    {
        $output = '';
        $exitCode = 0;

        exec($command . ' 2>&1', $outputLines, $exitCode);
        $output = implode("\n", $outputLines);

        return [
            'output' => $output,
            'exitCode' => $exitCode,
        ];
    }
}
