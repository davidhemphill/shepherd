<?php

namespace Shep\Commands;

use Shep\WorktreeManager;
use Symfony\Component\Console\Attribute\AsCommand;
use Symfony\Component\Console\Command\Command;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;

use function Laravel\Prompts\error;
use function Laravel\Prompts\info;
use function Laravel\Prompts\table;

#[AsCommand(
    name: 'worktrees',
    description: 'List all worktrees',
)]
class ListCommand extends Command
{
    public function __construct(private WorktreeManager $worktreeManager)
    {
        parent::__construct();
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        // Validate we're in a git repo
        if (! $this->worktreeManager->isGitRepo()) {
            error('Not in a git repository.');

            return Command::FAILURE;
        }

        $worktrees = $this->worktreeManager->listWorktrees();

        if (empty($worktrees)) {
            info('No worktrees found.');

            return Command::SUCCESS;
        }

        $rows = [];
        foreach ($worktrees as $worktree) {
            $branch = $worktree['branch'] ?? ($worktree['detached'] ?? false ? '(detached)' : 'N/A');
            $rows[] = [
                $branch,
                $worktree['path'] ?? 'N/A',
                substr($worktree['head'] ?? '', 0, 8),
            ];
        }

        table(
            headers: ['Branch', 'Path', 'HEAD'],
            rows: $rows
        );

        return Command::SUCCESS;
    }
}
