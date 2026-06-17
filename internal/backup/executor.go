package backup

import (
	"fmt"
	"log"
	"os"

	"github.com/FranLegon/GitBackuper/internal/config"
	"github.com/FranLegon/GitBackuper/internal/git"
	"github.com/FranLegon/GitBackuper/internal/platform"
)

type Result struct {
	Succeeded []string
	Failed    map[string]error
}

type Executor struct {
	cfg       *config.BackupConfig
	platforms map[config.Platform]platform.Platform
}

func NewExecutor(cfg *config.BackupConfig, platforms map[config.Platform]platform.Platform) *Executor {
	return &Executor{cfg: cfg, platforms: platforms}
}

func (e *Executor) Run() *Result {
	result := &Result{
		Failed: make(map[string]error),
	}

	for i := range e.cfg.Sources {
		repo := &e.cfg.Sources[i]
		repoID := fmt.Sprintf("%s:%s/%s", repo.Platform, repo.Owner, repo.Name)
		log.Printf("[%d/%d] Processing %s", i+1, len(e.cfg.Sources), repoID)

		if err := e.backupRepo(repo); err != nil {
			log.Printf("  ERROR: %s: %v", repoID, err)
			result.Failed[repoID] = err
		} else {
			log.Printf("  OK: %s", repoID)
			result.Succeeded = append(result.Succeeded, repoID)
		}
	}
	return result
}

func (e *Executor) backupRepo(repo *config.Repository) error {
	if !e.cfg.ForcePrivate && !repo.Private {
		plat := e.platforms[repo.Platform]
		if plat != nil {
			private, err := plat.GetVisibility(repo.Owner, repo.Name)
			if err == nil {
				repo.Private = private
			}
		}
	}

	destsToSync := e.destinationsNeedingSync(repo)
	if len(destsToSync) == 0 {
		log.Printf("  All destinations up to date, skipping")
		return nil
	}

	tempDir, err := os.MkdirTemp("", "gitbackuper-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	log.Printf("  Cloning %s into %s", repo.CloneURL, tempDir)
	if err := git.Clone(repo.CloneURL, tempDir); err != nil {
		return fmt.Errorf("cloning: %w", err)
	}

	for _, dest := range destsToSync {
		if err := e.syncToDestination(tempDir, repo, dest); err != nil {
			return fmt.Errorf("syncing to %s:%s: %w", dest.Platform, dest.Owner, err)
		}
	}
	return nil
}

func (e *Executor) destinationsNeedingSync(repo *config.Repository) []config.Destination {
	sourceRefs, err := git.LsRemote(repo.CloneURL)
	if err != nil {
		log.Printf("  Warning: could not ls-remote source, proceeding with full backup: %v", err)
		return e.cfg.Destinations
	}

	var needSync []config.Destination
	for _, dest := range e.cfg.Destinations {
		destName := repo.Name
		if e.cfg.Prefix {
			destName = "backup_" + destName
		}

		plat := e.platforms[dest.Platform]
		if plat == nil {
			needSync = append(needSync, dest)
			continue
		}

		destURL := plat.CloneURL(dest.Owner, destName)
		destRefs, err := git.LsRemote(destURL)
		if err != nil {
			needSync = append(needSync, dest)
			continue
		}

		if refsMatch(sourceRefs, destRefs) {
			log.Printf("  Skipping %s:%s/%s (already up to date)", dest.Platform, dest.Owner, destName)
		} else {
			needSync = append(needSync, dest)
		}
	}
	return needSync
}

func refsMatch(source, dest map[string]string) bool {
	for ref, hash := range source {
		if destHash, ok := dest[ref]; !ok || destHash != hash {
			return false
		}
	}
	return true
}

func (e *Executor) syncToDestination(repoDir string, repo *config.Repository, dest config.Destination) error {
	destName := repo.Name
	if e.cfg.Prefix {
		destName = "backup_" + destName
	}

	plat := e.platforms[dest.Platform]
	if plat == nil {
		return fmt.Errorf("no platform implementation for %q", dest.Platform)
	}

	private := e.cfg.ForcePrivate || repo.Private

	exists, err := plat.RepoExists(dest.Owner, destName)
	if err != nil {
		return fmt.Errorf("checking if repo exists: %w", err)
	}

	if !exists {
		log.Printf("  Creating %s:%s/%s (private=%v)", dest.Platform, dest.Owner, destName, private)
		if err := plat.CreateRepo(dest.Owner, destName, private); err != nil {
			return fmt.Errorf("creating repo: %w", err)
		}
	}

	remoteName := fmt.Sprintf("dest-%s-%s", dest.Platform, dest.Owner)
	destURL := plat.CloneURL(dest.Owner, destName)

	if err := git.AddRemote(repoDir, remoteName, destURL); err != nil {
		return fmt.Errorf("adding remote: %w", err)
	}

	if exists {
		log.Printf("  Fetching destination %s", remoteName)
		if err := git.FetchRemote(repoDir, remoteName); err != nil {
			log.Printf("  Warning: could not fetch destination (may be empty): %v", err)
		} else {
			diverged, err := CheckDivergence(repoDir, remoteName)
			if err != nil {
				log.Printf("  Warning: divergence check failed: %v", err)
			} else if len(diverged) > 0 {
				log.Printf("  Divergence detected on branches: %v. Saving backup branches.", diverged)
				if err := SaveDivergentState(repoDir, remoteName, diverged); err != nil {
					return fmt.Errorf("saving divergent state: %w", err)
				}
			}
		}
	}

	log.Printf("  Force pushing to %s:%s/%s", dest.Platform, dest.Owner, destName)
	if err := git.ForcePushAll(repoDir, remoteName); err != nil {
		return fmt.Errorf("force pushing: %w", err)
	}

	return nil
}
