package platform

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/FranLegon/GitBackuper/internal/config"
)

type GitLab struct{}

type glProjectEntry struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Visibility string `json:"visibility"`
	HTTPUrl    string `json:"http_url_to_repo"`
}

func (g *GitLab) ListRepos(owner string) ([]config.Repository, error) {
	// Try group first, then user
	repos, err := g.listGroupRepos(owner)
	if err != nil {
		repos, err = g.listUserRepos(owner)
		if err != nil {
			return nil, fmt.Errorf("could not list repos for %q (tried group and user): %w", owner, err)
		}
	}
	return repos, nil
}

func (g *GitLab) listGroupRepos(owner string) ([]config.Repository, error) {
	cmd := exec.Command("glab", "api", fmt.Sprintf("groups/%s/projects?per_page=100&include_subgroups=false", urlEncode(owner)))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("glab api groups failed: %w", err)
	}
	return g.parseProjects(owner, out)
}

func (g *GitLab) listUserRepos(owner string) ([]config.Repository, error) {
	cmd := exec.Command("glab", "api", fmt.Sprintf("users/%s/projects?per_page=100", urlEncode(owner)))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("glab api users failed: %w", err)
	}
	return g.parseProjects(owner, out)
}

func (g *GitLab) parseProjects(owner string, data []byte) ([]config.Repository, error) {
	var entries []glProjectEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parsing glab response: %w", err)
	}

	repos := make([]config.Repository, 0, len(entries))
	for _, e := range entries {
		repos = append(repos, config.Repository{
			Platform: config.PlatformGitLab,
			Owner:    owner,
			Name:     e.Path,
			CloneURL: e.HTTPUrl,
			Private:  e.Visibility == "private",
		})
	}
	return repos, nil
}

func (g *GitLab) RepoExists(owner, repo string) (bool, error) {
	cmd := exec.Command("glab", "api", fmt.Sprintf("projects/%s", urlEncode(owner+"/"+repo)))
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() != 0 {
			return false, nil
		}
		return false, fmt.Errorf("glab api project check failed: %w", err)
	}
	return true, nil
}

func (g *GitLab) GetVisibility(owner, repo string) (bool, error) {
	cmd := exec.Command("glab", "api", fmt.Sprintf("projects/%s", urlEncode(owner+"/"+repo)))
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("glab api get visibility failed: %w", err)
	}

	var project struct {
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal(out, &project); err != nil {
		return false, fmt.Errorf("parsing project visibility: %w", err)
	}
	return project.Visibility == "private", nil
}

func (g *GitLab) CreateRepo(owner, repo string, private bool) error {
	visibility := "public"
	if private {
		visibility = "private"
	}

	body := fmt.Sprintf(`{"name":"%s","namespace_id":null,"visibility":"%s","path":"%s"}`, repo, visibility, repo)

	// Try to create in group namespace first
	nsID, err := g.getNamespaceID(owner)
	if err == nil && nsID != "" {
		body = fmt.Sprintf(`{"name":"%s","namespace_id":%s,"visibility":"%s","path":"%s"}`, repo, nsID, visibility, repo)
	}

	cmd := exec.Command("glab", "api", "projects", "--method", "POST", "--raw-field", fmt.Sprintf("name=%s", repo), "--raw-field", fmt.Sprintf("visibility=%s", visibility), "--raw-field", fmt.Sprintf("path=%s", repo))
	if nsID != "" {
		cmd = exec.Command("glab", "api", "projects", "--method", "POST", "--raw-field", fmt.Sprintf("name=%s", repo), "--raw-field", fmt.Sprintf("namespace_id=%s", nsID), "--raw-field", fmt.Sprintf("visibility=%s", visibility), "--raw-field", fmt.Sprintf("path=%s", repo))
	}
	_ = body

	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("glab repo create failed: %w: %s", err, string(out))
	}
	return nil
}

func (g *GitLab) getNamespaceID(owner string) (string, error) {
	cmd := exec.Command("glab", "api", fmt.Sprintf("namespaces?search=%s", urlEncode(owner)))
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var namespaces []struct {
		ID       int    `json:"id"`
		FullPath string `json:"full_path"`
	}
	if err := json.Unmarshal(out, &namespaces); err != nil {
		return "", err
	}

	for _, ns := range namespaces {
		if strings.EqualFold(ns.FullPath, owner) {
			return fmt.Sprintf("%d", ns.ID), nil
		}
	}
	return "", fmt.Errorf("namespace not found for %q", owner)
}

func (g *GitLab) CloneURL(owner, repo string) string {
	return fmt.Sprintf("https://gitlab.com/%s/%s.git", owner, repo)
}

func urlEncode(s string) string {
	return strings.ReplaceAll(s, "/", "%2F")
}
