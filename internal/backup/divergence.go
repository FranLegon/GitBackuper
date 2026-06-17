package backup

import (
	"fmt"
	"time"

	"github.com/FranLegon/GitBackuper/internal/git"
)

func CheckDivergence(repoDir, destRemote string) ([]string, error) {
	destBranches, err := git.ListRemoteBranches(repoDir, destRemote)
	if err != nil {
		return nil, fmt.Errorf("listing destination branches: %w", err)
	}

	localBranches, err := git.ListLocalBranches(repoDir)
	if err != nil {
		return nil, fmt.Errorf("listing local branches: %w", err)
	}

	localSet := make(map[string]bool)
	for _, b := range localBranches {
		localSet[b] = true
	}

	var diverged []string
	for _, branch := range destBranches {
		if branch == "HEAD" {
			continue
		}
		if !localSet[branch] {
			continue
		}

		destRef := fmt.Sprintf("%s/%s", destRemote, branch)
		sourceRef := branch

		if !git.RefExists(repoDir, destRef) || !git.RefExists(repoDir, sourceRef) {
			continue
		}

		isAnc, err := git.IsAncestor(repoDir, destRef, sourceRef)
		if err != nil {
			continue
		}
		if !isAnc {
			diverged = append(diverged, branch)
		}
	}
	return diverged, nil
}

func SaveDivergentState(repoDir, destRemote string, divergedBranches []string) error {
	timestamp := time.Now().Format("20060102-150405")

	for _, branch := range divergedBranches {
		destRef := fmt.Sprintf("%s/%s", destRemote, branch)
		backupBranch := fmt.Sprintf("backup-pre-sync-%s-%s", timestamp, branch)

		if err := git.CreateBranch(repoDir, backupBranch, destRef); err != nil {
			return fmt.Errorf("creating backup branch for %s: %w", branch, err)
		}

		if err := git.PushBranch(repoDir, destRemote, backupBranch); err != nil {
			return fmt.Errorf("pushing backup branch %s: %w", backupBranch, err)
		}
	}
	return nil
}
