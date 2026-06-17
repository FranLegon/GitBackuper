package source

import (
	"fmt"
	"strings"

	"github.com/FranLegon/GitBackuper/internal/config"
	"github.com/FranLegon/GitBackuper/internal/platform"
)

type Resolver struct {
	platforms map[config.Platform]platform.Platform
}

func NewResolver(platforms map[config.Platform]platform.Platform) *Resolver {
	return &Resolver{platforms: platforms}
}

func (r *Resolver) ResolveFromFlags(p config.Platform, owner string, repoNames []string) ([]config.Repository, error) {
	plat := r.platforms[p]
	if plat == nil {
		return nil, fmt.Errorf("no platform implementation for %q", p)
	}

	if len(repoNames) == 0 {
		return plat.ListRepos(owner)
	}

	repos := make([]config.Repository, 0, len(repoNames))
	for _, name := range repoNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		repos = append(repos, config.Repository{
			Platform: p,
			Owner:    owner,
			Name:     name,
			CloneURL: plat.CloneURL(owner, name),
			Private:  false, // will be resolved during backup if needed
		})
	}
	return repos, nil
}

func (r *Resolver) ResolveFromURIs(raw string) ([]config.Repository, error) {
	uris, err := ParseURIs(raw)
	if err != nil {
		return nil, err
	}

	var repos []config.Repository
	for _, u := range uris {
		plat := r.platforms[u.Platform]
		if plat == nil {
			return nil, fmt.Errorf("no platform implementation for %q", u.Platform)
		}

		if u.Repo == "" {
			listed, err := plat.ListRepos(u.Owner)
			if err != nil {
				return nil, fmt.Errorf("listing repos for %s:%s: %w", u.Platform, u.Owner, err)
			}
			repos = append(repos, listed...)
		} else {
			repos = append(repos, config.Repository{
				Platform: u.Platform,
				Owner:    u.Owner,
				Name:     u.Repo,
				CloneURL: plat.CloneURL(u.Owner, u.Repo),
				Private:  false,
			})
		}
	}
	return repos, nil
}

func ParseRepoNames(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var names []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			names = append(names, p)
		}
	}
	return names
}

func (r *Resolver) ResolveVisibility(repo *config.Repository) error {
	plat := r.platforms[repo.Platform]
	if plat == nil {
		return fmt.Errorf("no platform for %q", repo.Platform)
	}
	private, err := plat.GetVisibility(repo.Owner, repo.Name)
	if err != nil {
		return err
	}
	repo.Private = private
	return nil
}
