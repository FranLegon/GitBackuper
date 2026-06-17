package platform

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/FranLegon/GitBackuper/internal/config"
)

type GitHub struct{}

type ghRepoEntry struct {
	Name      string `json:"name"`
	IsPrivate bool   `json:"isPrivate"`
	SSHUrl    string `json:"sshUrl"`
	CloneURL  string `json:"url"`
}

func (g *GitHub) ListRepos(owner string) ([]config.Repository, error) {
	cmd := exec.Command("gh", "repo", "list", owner, "--limit", "1000", "--json", "name,isPrivate,sshUrl,url")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh repo list failed: %w: %s", err, getStderr(err))
	}

	var entries []ghRepoEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("parsing gh repo list output: %w", err)
	}

	repos := make([]config.Repository, 0, len(entries))
	for _, e := range entries {
		repos = append(repos, config.Repository{
			Platform: config.PlatformGitHub,
			Owner:    owner,
			Name:     e.Name,
			CloneURL: fmt.Sprintf("https://github.com/%s/%s.git", owner, e.Name),
			Private:  e.IsPrivate,
		})
	}
	return repos, nil
}

func (g *GitHub) RepoExists(owner, repo string) (bool, error) {
	cmd := exec.Command("gh", "repo", "view", owner+"/"+repo, "--json", "name")
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("gh repo view failed: %w", err)
	}
	return true, nil
}

func (g *GitHub) GetVisibility(owner, repo string) (bool, error) {
	cmd := exec.Command("gh", "repo", "view", owner+"/"+repo, "--json", "isPrivate", "--jq", ".isPrivate")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("gh repo view visibility failed: %w", err)
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func (g *GitHub) CreateRepo(owner, repo string, private bool) error {
	visibility := "--public"
	if private {
		visibility = "--private"
	}

	cmd := exec.Command("gh", "repo", "create", owner+"/"+repo, visibility, "--confirm")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gh repo create failed: %w: %s", err, string(out))
	}
	return nil
}

func (g *GitHub) CloneURL(owner, repo string) string {
	return fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
}

func getStderr(err error) string {
	if exitErr, ok := err.(*exec.ExitError); ok {
		return string(exitErr.Stderr)
	}
	return ""
}
