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
    name: 'new',
    description: 'Create a new worktree for a branch',
)]
class NewCommand extends Command
{
    public function __construct(private WorktreeManager $worktreeManager)
    {
        parent::__construct();
    }

    protected function configure(): void
    {
        $this->addArgument('branch', InputArgument::REQUIRED, 'The branch name for the worktree');
    }

    protected function execute(InputInterface $input, OutputInterface $output): int
    {
        $branch = $input->getArgument('branch');

        // Validate we're in a git repo
        if (! $this->worktreeManager->isGitRepo()) {
            error('Not in a git repository.');

            return Command::FAILURE;
        }

        // Check if worktree already exists
        if ($this->worktreeManager->worktreeExists($branch)) {
            error("Worktree for branch '{$branch}' already exists.");

            return Command::FAILURE;
        }

        // Create branch if it doesn't exist
        if (! $this->worktreeManager->branchExists($branch)) {
            $create = confirm(
                label: "Branch '{$branch}' does not exist. Create it?",
                default: true
            );

            if (! $create) {
                info('Aborted.');

                return Command::SUCCESS;
            }

            $created = spin(
                fn () => $this->worktreeManager->createBranch($branch),
                "Creating branch '{$branch}'..."
            );

            if (! $created) {
                error("Failed to create branch '{$branch}'.");

                return Command::FAILURE;
            }
        }

        // Create worktree
        $success = spin(
            fn () => $this->worktreeManager->createWorktree($branch),
            "Creating worktree for '{$branch}'..."
        );

        if (! $success) {
            error("Failed to create worktree for '{$branch}'.");

            return Command::FAILURE;
        }

        // Setup environment
        spin(
            fn () => $this->worktreeManager->setupEnvironment($branch),
            'Setting up environment...'
        );

        $worktreePath = $this->worktreeManager->getWorktreePath($branch);

        info("Worktree created at: {$worktreePath}");

        // Output path only (for shell wrapper to capture and cd into)
        // The shell wrapper looks for the last line of output
        $output->writeln($worktreePath);

        return Command::SUCCESS;
    }
}
