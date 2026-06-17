package prereq

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type tool struct {
	name       string
	installCmd []installOption
	authCheck  func() bool
	authCmd    []string
}

type installOption struct {
	platform string
	command  string
	args     []string
}

func Check() error {
	tools := []tool{
		{
			name: "git",
			installCmd: []installOption{
				{platform: "windows", command: "winget", args: []string{"install", "--id", "Git.Git", "-e", "--source", "winget"}},
				{platform: "darwin", command: "brew", args: []string{"install", "git"}},
				{platform: "linux", command: "sudo", args: []string{"apt-get", "install", "-y", "git"}},
			},
		},
		{
			name: "gh",
			installCmd: []installOption{
				{platform: "windows", command: "winget", args: []string{"install", "--id", "GitHub.cli", "-e", "--source", "winget"}},
				{platform: "darwin", command: "brew", args: []string{"install", "gh"}},
				{platform: "linux", command: "sudo", args: []string{"apt-get", "install", "-y", "gh"}},
			},
			authCheck: func() bool { return isGHLoggedIn() },
			authCmd:   []string{"gh", "auth", "login"},
		},
		{
			name: "glab",
			installCmd: []installOption{
				{platform: "windows", command: "winget", args: []string{"install", "glab.glab"}},
				{platform: "darwin", command: "brew", args: []string{"install", "glab"}},
				{platform: "linux", command: "sudo", args: []string{"apt-get", "install", "-y", "glab"}},
			},
			authCheck: func() bool { return isGlabLoggedIn() },
			authCmd:   []string{"glab", "auth", "login"},
		},
	}

	for _, t := range tools {
		if !isInstalled(t.name) {
			if err := findAndAddToPath(t.name); err == nil {
				fmt.Printf("Found %s and added its directory to PATH for this session.\n", t.name)
			} else {
				if err := promptInstall(t); err != nil {
					return err
				}
				if !isInstalled(t.name) {
					if err := findAndAddToPath(t.name); err != nil {
						return fmt.Errorf("%s is still not available in PATH after installation — you may need to restart your terminal", t.name)
					}
					fmt.Printf("Found %s and added its directory to PATH for this session.\n", t.name)
				}
			}
		}

		if t.authCheck != nil && !t.authCheck() {
			if err := promptLogin(t); err != nil {
				return err
			}
		}
	}
	return nil
}

func isInstalled(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func isGHLoggedIn() bool {
	cmd := exec.Command("gh", "auth", "status")
	return cmd.Run() == nil
}

func isGlabLoggedIn() bool {
	cmd := exec.Command("glab", "auth", "status")
	return cmd.Run() == nil
}

func promptInstall(t tool) error {
	opt := getInstallOption(t.installCmd)
	if opt == nil {
		return fmt.Errorf("%s is not installed and no automatic installer is available for %s — please install it manually", t.name, runtime.GOOS)
	}

	fmt.Printf("\n%s is not installed. Install it now using '%s %s'? [Y/n] ", t.name, opt.command, strings.Join(opt.args, " "))
	if !askYesNo() {
		return fmt.Errorf("%s is required but not installed — aborting", t.name)
	}

	fmt.Printf("Installing %s...\n", t.name)
	cmd := exec.Command(opt.command, opt.args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", t.name, err)
	}
	fmt.Printf("%s installed successfully.\n\n", t.name)
	return nil
}

func promptLogin(t tool) error {
	fmt.Printf("\n%s is not authenticated. Log in now? [Y/n] ", t.name)
	if !askYesNo() {
		return fmt.Errorf("%s authentication is required — aborting", t.name)
	}

	fmt.Printf("Running: %s\n", strings.Join(t.authCmd, " "))
	cmd := exec.Command(t.authCmd[0], t.authCmd[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s login failed: %w", t.name, err)
	}
	fmt.Println()
	return nil
}

func getInstallOption(opts []installOption) *installOption {
	for i := range opts {
		if opts[i].platform == runtime.GOOS {
			if isInstalled(opts[i].command) {
				return &opts[i]
			}
		}
	}
	return nil
}

func askYesNo() bool {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		return answer == "" || answer == "y" || answer == "yes"
	}
	return false
}

func findAndAddToPath(name string) error {
	binary := name
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}

	searchDirs := getSearchDirs()

	for _, dir := range searchDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return filepath.SkipDir
			}
			if info.IsDir() {
				return nil
			}
			if strings.EqualFold(info.Name(), binary) {
				binDir := filepath.Dir(path)
				addToProcessPath(binDir)
				return filepath.SkipAll
			}
			return nil
		})
		if err == filepath.SkipAll {
			break
		}
		_ = err
	}

	return verifyInstalled(name)
}

func getSearchDirs() []string {
	switch runtime.GOOS {
	case "windows":
		home := os.Getenv("USERPROFILE")
		programFiles := os.Getenv("ProgramFiles")
		localAppData := os.Getenv("LOCALAPPDATA")
		var dirs []string
		if localAppData != "" {
			dirs = append(dirs, filepath.Join(localAppData, "Programs"))
			dirs = append(dirs, filepath.Join(localAppData, "Microsoft", "WinGet", "Links"))
		}
		if programFiles != "" {
			dirs = append(dirs, programFiles)
		}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, "AppData", "Local", "Programs"))
			dirs = append(dirs, filepath.Join(home, "scoop", "shims"))
		}
		return dirs
	case "linux":
		home := os.Getenv("HOME")
		dirs := []string{
			"/usr/local/bin",
			"/usr/bin",
			"/snap/bin",
		}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, ".local", "bin"))
			dirs = append(dirs, filepath.Join(home, "bin"))
		}
		return dirs
	case "darwin":
		home := os.Getenv("HOME")
		dirs := []string{
			"/usr/local/bin",
			"/opt/homebrew/bin",
			"/usr/bin",
		}
		if home != "" {
			dirs = append(dirs, filepath.Join(home, ".local", "bin"))
		}
		return dirs
	default:
		return nil
	}
}

func addToProcessPath(dir string) {
	pathSep := string(os.PathListSeparator)
	current := os.Getenv("PATH")
	for _, d := range strings.Split(current, pathSep) {
		if strings.EqualFold(d, dir) {
			return
		}
	}
	os.Setenv("PATH", current+pathSep+dir)
}

func verifyInstalled(name string) error {
	if isInstalled(name) {
		return nil
	}
	return fmt.Errorf("%s not found", name)
}
