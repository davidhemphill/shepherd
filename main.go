package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Colors
const (
	Red    = "\033[0;31m"
	Green  = "\033[0;32m"
	Yellow = "\033[0;33m"
	Cyan   = "\033[0;36m"
	Dim    = "\033[2m"
	NC     = "\033[0m" // No Color
)

// Print functions
func printError(msg string) {
	fmt.Fprintf(os.Stderr, "%sError: %s%s\n", Red, msg, NC)
}

func printSuccess(msg string) {
	fmt.Printf("%s%s%s\n", Green, msg, NC)
}

func printInfo(msg string) {
	fmt.Printf("%s%s%s\n", Cyan, msg, NC)
}

func printDim(msg string) {
	fmt.Printf("%s%s%s\n", Dim, msg, NC)
}

// Git operations
func isGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	return cmd.Run() == nil
}

func checkGitRepo() bool {
	if !isGitRepo() {
		printError("Not in a git repository.")
		return false
	}
	return true
}

func getRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getMainRepoRoot() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	out, err := cmd.Output()
	if err != nil {
		return getRepoRoot()
	}

	gitCommonDir := strings.TrimSpace(string(out))
	if gitCommonDir == ".git" {
		return getRepoRoot()
	}

	return filepath.Dir(gitCommonDir), nil
}

func getWorktreePath(branch string) (string, error) {
	root, err := getRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".worktrees", branch), nil
}

func branchExists(branch string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch)
	return cmd.Run() == nil
}

func worktreeExists(branch string) bool {
	path, err := getWorktreePath(branch)
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func getCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// Herd operations
func getHerdSiteName(branch string) (string, error) {
	repoRoot, err := getMainRepoRoot()
	if err != nil {
		return "", err
	}

	repoName := filepath.Base(repoRoot)

	// Split repo name on first dot: "pushsilver.dev" -> "pushsilver" + "dev"
	parts := strings.SplitN(repoName, ".", 2)
	base := parts[0]

	if len(parts) > 1 {
		// Has a suffix (e.g., pushsilver.dev -> pushsilver-branch.dev)
		return fmt.Sprintf("%s-%s.%s", base, branch, parts[1]), nil
	}
	// No suffix (e.g., myapp -> myapp-branch)
	return fmt.Sprintf("%s-%s", repoName, branch), nil
}

func herdExists() bool {
	_, err := exec.LookPath("herd")
	return err == nil
}

func linkToHerd(worktreePath, branch string) error {
	siteName, err := getHerdSiteName(branch)
	if err != nil {
		return err
	}

	if !herdExists() {
		printError("Herd CLI not found. Skipping Herd setup.")
		return fmt.Errorf("herd not found")
	}

	// Link the site (run from worktree directory)
	cmd := exec.Command("herd", "link", siteName)
	cmd.Dir = worktreePath
	if err := cmd.Run(); err != nil {
		return err
	}

	// Secure the site with HTTPS
	cmd = exec.Command("herd", "secure", siteName)
	cmd.Run() // Ignore errors

	// Restart Herd to refresh the GUI
	cmd = exec.Command("herd", "restart")
	cmd.Run() // Ignore errors

	return nil
}

func unlinkFromHerd(branch string) {
	if !herdExists() {
		return
	}

	siteName, err := getHerdSiteName(branch)
	if err != nil {
		return
	}

	cmd := exec.Command("herd", "unlink", siteName)
	cmd.Run() // Ignore errors
}

// Environment setup
func setupEnvironment(worktreePath, branch string) error {
	envPath := filepath.Join(worktreePath, ".env")
	envExamplePath := filepath.Join(worktreePath, ".env.example")

	// Copy .env.example to .env if needed
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		if _, err := os.Stat(envExamplePath); err == nil {
			input, err := os.ReadFile(envExamplePath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(envPath, input, 0644); err != nil {
				return err
			}
		}
	}

	// Create database directory if needed
	dbDir := filepath.Join(worktreePath, "database")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return err
	}

	// Create SQLite database
	dbPath := filepath.Join(dbDir, "database.sqlite")
	f, err := os.OpenFile(dbPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	f.Close()

	// Get the site URL for this worktree
	siteName, err := getHerdSiteName(branch)
	if err != nil {
		return err
	}
	appURL := fmt.Sprintf("https://%s.test", siteName)

	// Update .env if it exists
	if _, err := os.Stat(envPath); err == nil {
		content, err := os.ReadFile(envPath)
		if err != nil {
			return err
		}
		envContent := string(content)

		// Update APP_URL
		appURLRegex := regexp.MustCompile(`(?m)^APP_URL=.*$`)
		if appURLRegex.MatchString(envContent) {
			envContent = appURLRegex.ReplaceAllString(envContent, "APP_URL="+appURL)
		} else {
			envContent += "\nAPP_URL=" + appURL
		}

		// Update DB_CONNECTION
		dbConnRegex := regexp.MustCompile(`(?m)^DB_CONNECTION=.*$`)
		if dbConnRegex.MatchString(envContent) {
			envContent = dbConnRegex.ReplaceAllString(envContent, "DB_CONNECTION=sqlite")
		} else {
			envContent += "\nDB_CONNECTION=sqlite"
		}

		// Update DB_DATABASE
		dbDatabaseRegex := regexp.MustCompile(`(?m)^DB_DATABASE=.*$`)
		if dbDatabaseRegex.MatchString(envContent) {
			envContent = dbDatabaseRegex.ReplaceAllString(envContent, "DB_DATABASE="+dbPath)
		} else {
			envContent += "\nDB_DATABASE=" + dbPath
		}

		// Comment out unused DB settings
		dbHostRegex := regexp.MustCompile(`(?m)^DB_HOST=`)
		envContent = dbHostRegex.ReplaceAllString(envContent, "#DB_HOST=")
		dbPortRegex := regexp.MustCompile(`(?m)^DB_PORT=`)
		envContent = dbPortRegex.ReplaceAllString(envContent, "#DB_PORT=")
		dbUserRegex := regexp.MustCompile(`(?m)^DB_USERNAME=`)
		envContent = dbUserRegex.ReplaceAllString(envContent, "#DB_USERNAME=")
		dbPassRegex := regexp.MustCompile(`(?m)^DB_PASSWORD=`)
		envContent = dbPassRegex.ReplaceAllString(envContent, "#DB_PASSWORD=")

		if err := os.WriteFile(envPath, []byte(envContent), 0644); err != nil {
			return err
		}
	}

	return nil
}

// User confirmation
func confirm(prompt string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)

	var suffix string
	if defaultYes {
		suffix = " [Y/n] "
	} else {
		suffix = " [y/N] "
	}

	fmt.Print(prompt + suffix)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response == "" {
		return defaultYes
	}
	return response == "y" || response == "yes"
}

// Provision a worktree
func provisionWorktree(worktreePath, branch string) error {
	// Setup environment
	printInfo("Setting up environment...")
	if err := setupEnvironment(worktreePath, branch); err != nil {
		printError(fmt.Sprintf("Failed to setup environment: %v", err))
	}

	// Install composer dependencies
	printInfo("Running composer install...")
	cmd := exec.Command("composer", "install", "--quiet")
	cmd.Dir = worktreePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		printError(fmt.Sprintf("Composer install failed: %v", err))
	}

	// Optionally generate application key
	if confirm("Generate application key?", true) {
		printInfo("Generating application key...")
		cmd = exec.Command("php", "artisan", "key:generate", "--quiet")
		cmd.Dir = worktreePath
		cmd.Run()
	}

	// Optionally run migrations with seeding
	if confirm("Run migrations with seeding?", false) {
		printInfo("Running migrations with seeding...")
		cmd = exec.Command("php", "artisan", "migrate", "--seed", "--quiet")
		cmd.Dir = worktreePath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

	// Link to Herd
	siteName, _ := getHerdSiteName(branch)
	printInfo(fmt.Sprintf("Linking to Herd as '%s'...", siteName))
	if err := linkToHerd(worktreePath, branch); err == nil {
		printSuccess(fmt.Sprintf("Site available at: https://%s.test", siteName))
	}

	// Offer to run npm run dev if package.json exists
	packageJSON := filepath.Join(worktreePath, "package.json")
	if _, err := os.Stat(packageJSON); err == nil {
		if confirm("Run 'npm run dev'?", false) {
			printInfo("Starting npm run dev...")
			cmd = exec.Command("npm", "run", "dev")
			cmd.Dir = worktreePath
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			cmd.Run()
		}
	}

	return nil
}

// Commands
func cmdNew(branch string) int {
	if branch == "" {
		printError("Branch name required.")
		fmt.Println("Usage: shep new <branch>")
		return 1
	}

	if !checkGitRepo() {
		return 1
	}

	if worktreeExists(branch) {
		printError(fmt.Sprintf("Worktree for branch '%s' already exists.", branch))
		return 1
	}

	// Create branch if it doesn't exist
	if !branchExists(branch) {
		if confirm(fmt.Sprintf("Branch '%s' does not exist. Create it?", branch), true) {
			printInfo(fmt.Sprintf("Creating branch '%s'...", branch))
			cmd := exec.Command("git", "branch", branch)
			if err := cmd.Run(); err != nil {
				printError(fmt.Sprintf("Failed to create branch: %v", err))
				return 1
			}
		} else {
			fmt.Println("Aborted.")
			return 0
		}
	}

	worktreePath, err := getWorktreePath(branch)
	if err != nil {
		printError(fmt.Sprintf("Failed to get worktree path: %v", err))
		return 1
	}

	// Create worktree
	printInfo(fmt.Sprintf("Creating worktree for '%s'...", branch))
	cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		printError(fmt.Sprintf("Failed to create worktree: %v", err))
		return 1
	}

	// Run provisioning
	provisionWorktree(worktreePath, branch)

	printSuccess(fmt.Sprintf("Worktree created at: %s", worktreePath))

	// Output path for shell wrapper to cd into
	fmt.Println(worktreePath)
	return 0
}

func cmdInit(branch string) int {
	if !checkGitRepo() {
		return 1
	}

	var worktreePath string
	var err error

	if branch == "" {
		// No branch specified, use current directory
		worktreePath, err = os.Getwd()
		if err != nil {
			printError(fmt.Sprintf("Failed to get current directory: %v", err))
			return 1
		}

		// Get branch name from current directory
		branch, err = getCurrentBranch()
		if err != nil || branch == "" || branch == "HEAD" {
			printError("Could not determine branch name. Please specify a branch.")
			fmt.Println("Usage: shep init [branch]")
			return 1
		}

		printInfo(fmt.Sprintf("Provisioning current directory as '%s'...", branch))
	} else {
		// Branch specified, check if worktree exists
		if !worktreeExists(branch) {
			printError(fmt.Sprintf("Worktree for branch '%s' does not exist.", branch))
			fmt.Printf("Use 'shep new %s' to create it.\n", branch)
			return 1
		}

		worktreePath, err = getWorktreePath(branch)
		if err != nil {
			printError(fmt.Sprintf("Failed to get worktree path: %v", err))
			return 1
		}
		printInfo(fmt.Sprintf("Provisioning worktree '%s'...", branch))
	}

	// Run provisioning
	provisionWorktree(worktreePath, branch)

	printSuccess(fmt.Sprintf("Provisioning complete for '%s'", branch))
	return 0
}

func cmdRemove(branch string) int {
	if branch == "" {
		printError("Branch name required.")
		fmt.Println("Usage: shep remove <branch>")
		return 1
	}

	if !checkGitRepo() {
		return 1
	}

	if !worktreeExists(branch) {
		printError(fmt.Sprintf("Worktree for branch '%s' does not exist.", branch))
		return 1
	}

	worktreePath, err := getWorktreePath(branch)
	if err != nil {
		printError(fmt.Sprintf("Failed to get worktree path: %v", err))
		return 1
	}

	if confirm(fmt.Sprintf("Remove worktree at '%s'?", worktreePath), false) {
		// Unlink from Herd first
		printInfo("Unlinking from Herd...")
		unlinkFromHerd(branch)

		printInfo(fmt.Sprintf("Removing worktree '%s'...", branch))
		cmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			printError(fmt.Sprintf("Failed to remove worktree: %v", err))
			return 1
		}

		cmd = exec.Command("git", "worktree", "prune")
		cmd.Run()

		printSuccess(fmt.Sprintf("Worktree '%s' removed.", branch))
	} else {
		fmt.Println("Aborted.")
	}

	return 0
}

func cmdList() int {
	if !checkGitRepo() {
		return 1
	}

	cmd := exec.Command("git", "worktree", "list")
	out, err := cmd.Output()
	if err != nil {
		printError(fmt.Sprintf("Failed to list worktrees: %v", err))
		return 1
	}

	worktrees := strings.TrimSpace(string(out))
	if worktrees == "" {
		printInfo("No worktrees found.")
		return 0
	}

	// Print header
	fmt.Println()
	fmt.Printf("%s%-20s %-50s %s%s\n", Dim, "Branch", "Path", "HEAD", NC)
	fmt.Printf("%s%-20s %-50s %s%s\n", Dim, "------", "----", "----", NC)

	// Parse and print worktrees
	lines := strings.Split(worktrees, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		path := fields[0]
		head := fields[1]
		branch := "(detached)"
		if len(fields) >= 3 {
			branch = strings.Trim(fields[2], "[]")
		}

		fmt.Printf("%-20s %-50s %s\n", branch, path, head)
	}
	fmt.Println()

	return 0
}

func cmdHelp() {
	help := `Shep - Laravel Worktree Manager

Usage: shep <command> [arguments]

Commands:
  new <branch>      Create a new worktree for a branch
  init [branch]     Provision an existing worktree (or current directory)
  remove <branch>   Remove a worktree
  list              List all worktrees
  help              Show this help message

Examples:
  shep new feature-login    Create worktree for feature-login branch
  shep init                 Provision the current directory
  shep init feature-login   Provision existing worktree
  shep remove feature-login Remove the worktree
  shep list                 Show all worktrees
`
	fmt.Print(help)
}

func main() {
	args := os.Args[1:]
	cmd := "help"
	if len(args) > 0 {
		cmd = args[0]
	}

	var exitCode int

	switch cmd {
	case "new":
		branch := ""
		if len(args) > 1 {
			branch = args[1]
		}
		exitCode = cmdNew(branch)
	case "init":
		branch := ""
		if len(args) > 1 {
			branch = args[1]
		}
		exitCode = cmdInit(branch)
	case "remove":
		branch := ""
		if len(args) > 1 {
			branch = args[1]
		}
		exitCode = cmdRemove(branch)
	case "list", "ls":
		exitCode = cmdList()
	case "help", "--help", "-h":
		cmdHelp()
		exitCode = 0
	default:
		printError(fmt.Sprintf("Unknown command: %s", cmd))
		cmdHelp()
		exitCode = 1
	}

	os.Exit(exitCode)
}
