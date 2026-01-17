<?php

namespace Shep;

use Shep\Commands\ListCommand;
use Shep\Commands\NewCommand;
use Shep\Commands\RemoveCommand;
use Symfony\Component\Console\Application as ConsoleApplication;

class Application
{
    private ConsoleApplication $console;

    public function __construct()
    {
        $this->console = new ConsoleApplication('Shep - Laravel Worktree Manager', '1.0.0');

        $this->registerCommands();
    }

    private function registerCommands(): void
    {
        $worktreeManager = new WorktreeManager();

        $this->console->add(new NewCommand($worktreeManager));
        $this->console->add(new RemoveCommand($worktreeManager));
        $this->console->add(new ListCommand($worktreeManager));
    }

    public function run(): int
    {
        return $this->console->run();
    }
}
