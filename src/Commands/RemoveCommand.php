<?php

namespace Shep\Commands;

use Shep\WorktreeManager;
use Symfony\Component\Console\Attribute\AsCommand;
use Symfony\Component\Console\Command\Command;
use Symfony\Component\Console\Input\InputArgument;
use Symfony\Component\Console\Input\InputInterface;
use Symfony\Component\Console\Output\OutputInterface;

use function Laravel\Prompts\confirm;
use function Laravel\Prompts\error;
use function Laravel\Prompts\info;
use function Laravel\Prompts\spin;

#[AsCommand(
    name: 'remove',
    description: 'Remove a worktree',
)]
class RemoveCommand extends Command
{
    public function __construct(private WorktreeManager $worktreeManager)
    {
        parent::__construct();
    }

    protected function configure(): void
    {
        $this->addArgument('branch', InputArgument::REQUIRED, 'The branch name of the worktree to remove');
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        $branch = $input->getArgument('branch');

        // Validate we're in a git repo
        if (! $this->worktreeManager->isGitRepo()) {
            error('Not in a git repository.');

            return Command::FAILURE;
        }

        // Check if worktree exists
        if (! $this->worktreeManager->worktreeExists($branch)) {
            error("Worktree for branch '{$branch}' does not exist.");

            return Command::FAILURE;
        }

        $worktreePath = $this->worktreeManager->getWorktreePath($branch);

        // Confirm removal
        $confirmed = confirm(
            label: "Remove worktree at '{$worktreePath}'?",
            default: false,
            hint: 'This will remove the worktree directory and its contents.'
        );

        if (! $confirmed) {
            info('Aborted.');

            return Command::SUCCESS;
        }

        // Remove worktree
        $success = spin(
            fn () => $this->worktreeManager->removeWorktree($branch),
            "Removing worktree '{$branch}'..."
        );

        if (! $success) {
            error("Failed to remove worktree '{$branch}'.");

            return Command::FAILURE;
        }

        info("Worktree '{$branch}' removed successfully.");

        return Command::SUCCESS;
    }
}
