package source

import (
	"fmt"
	"strings"

	"github.com/FranLegon/GitBackuper/internal/config"
)

type ParsedURI struct {
	Platform config.Platform
	Owner    string
	Repo     string // empty means "all repos from owner"
}

func ParseURIs(raw string) ([]ParsedURI, error) {
	if raw == "" {
		return nil, fmt.Errorf("source URI string is empty")
	}

	parts := strings.Split(raw, ",")
	uris := make([]ParsedURI, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		u, err := ParseSingleURI(part)
		if err != nil {
			return nil, err
		}
		uris = append(uris, u)
	}
	if len(uris) == 0 {
		return nil, fmt.Errorf("no valid URIs found in %q", raw)
	}
	return uris, nil
}

func ParseSingleURI(raw string) (ParsedURI, error) {
	colonIdx := strings.Index(raw, ":")
	if colonIdx < 0 {
		return ParsedURI{}, fmt.Errorf("invalid URI %q: expected format 'platform:owner[/repo]'", raw)
	}

	platformStr := raw[:colonIdx]
	path := raw[colonIdx+1:]

	p, err := config.ParsePlatform(platformStr)
	if err != nil {
		return ParsedURI{}, fmt.Errorf("invalid URI %q: %w", raw, err)
	}

	if path == "" {
		return ParsedURI{}, fmt.Errorf("invalid URI %q: path cannot be empty", raw)
	}

	slashIdx := strings.Index(path, "/")
	if slashIdx < 0 {
		return ParsedURI{Platform: p, Owner: path, Repo: ""}, nil
	}

	owner := path[:slashIdx]
	repo := path[slashIdx+1:]
	if owner == "" || repo == "" {
		return ParsedURI{}, fmt.Errorf("invalid URI %q: owner and repo must not be empty", raw)
	}

	return ParsedURI{Platform: p, Owner: owner, Repo: repo}, nil
}
