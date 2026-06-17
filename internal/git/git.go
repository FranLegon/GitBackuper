package git

import (
	"fmt"
	"os/exec"
	"strings"
)

func Clone(url, destDir string) error {
	cmd := exec.Command("git", "clone", "--bare", url, destDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone --bare failed: %w: %s", err, string(out))
	}
	return nil
}

func AddRemote(repoDir, remoteName, url string) error {
	cmd := exec.Command("git", "-C", repoDir, "remote", "add", remoteName, url)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git remote add failed: %w: %s", err, string(out))
	}
	return nil
}

func FetchRemote(repoDir, remoteName string) error {
	cmd := exec.Command("git", "-C", repoDir, "fetch", remoteName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch %s failed: %w: %s", remoteName, err, string(out))
	}
	return nil
}

func ListRemoteBranches(repoDir, remoteName string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoDir, "for-each-ref", "--format=%(refname:short)", fmt.Sprintf("refs/remotes/%s/", remoteName))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref failed: %w", err)
	}

	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip remote prefix: "dest/main" -> "main"
		prefix := remoteName + "/"
		if strings.HasPrefix(line, prefix) {
			branches = append(branches, strings.TrimPrefix(line, prefix))
		} else {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

func ListLocalBranches(repoDir string) ([]string, error) {
	cmd := exec.Command("git", "-C", repoDir, "for-each-ref", "--format=%(refname:short)", "refs/heads/")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git for-each-ref heads failed: %w", err)
	}

	var branches []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

func IsAncestor(repoDir, ancestorRef, descendantRef string) (bool, error) {
	cmd := exec.Command("git", "-C", repoDir, "merge-base", "--is-ancestor", ancestorRef, descendantRef)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git merge-base --is-ancestor failed: %w", err)
}

func ForcePushAll(repoDir, remoteName string) error {
	cmd := exec.Command("git", "-C", repoDir, "push", "--force", "--all", remoteName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push --force --all failed: %w: %s", err, string(out))
	}

	cmdTags := exec.Command("git", "-C", repoDir, "push", "--force", "--tags", remoteName)
	if out, err := cmdTags.CombinedOutput(); err != nil {
		return fmt.Errorf("git push --force --tags failed: %w: %s", err, string(out))
	}
	return nil
}

func CreateBranch(repoDir, branchName, ref string) error {
	cmd := exec.Command("git", "-C", repoDir, "branch", branchName, ref)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git branch %s failed: %w: %s", branchName, err, string(out))
	}
	return nil
}

func PushBranch(repoDir, remoteName, branchName string) error {
	cmd := exec.Command("git", "-C", repoDir, "push", remoteName, branchName)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push %s %s failed: %w: %s", remoteName, branchName, err, string(out))
	}
	return nil
}

func RefExists(repoDir, ref string) bool {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "--verify", ref)
	return cmd.Run() == nil
}

func LsRemote(url string) (map[string]string, error) {
	cmd := exec.Command("git", "ls-remote", url)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-remote %s failed: %w", url, err)
	}

	refs := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		ref := parts[1]
		if strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/") {
			refs[ref] = parts[0]
		}
	}
	return refs, nil
}
