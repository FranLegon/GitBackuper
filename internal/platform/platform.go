package platform

import "github.com/FranLegon/GitBackuper/internal/config"

type Platform interface {
	ListRepos(owner string) ([]config.Repository, error)
	RepoExists(owner, repo string) (bool, error)
	GetVisibility(owner, repo string) (private bool, err error)
	CreateRepo(owner, repo string, private bool) error
	CloneURL(owner, repo string) string
}

func ForPlatform(p config.Platform) Platform {
	switch p {
	case config.PlatformGitHub:
		return &GitHub{}
	case config.PlatformGitLab:
		return &GitLab{}
	default:
		return nil
	}
}
