package config

import "fmt"

type Platform string

const (
	PlatformGitHub Platform = "github"
	PlatformGitLab Platform = "gitlab"
)

type Repository struct {
	Platform Platform
	Owner    string
	Name     string
	CloneURL string
	Private  bool
}

type Destination struct {
	Platform Platform
	Owner    string
}

type BackupConfig struct {
	Sources      []Repository
	Destinations []Destination
	Prefix       bool
	ForcePrivate bool
}

func ParsePlatform(s string) (Platform, error) {
	switch s {
	case "github":
		return PlatformGitHub, nil
	case "gitlab":
		return PlatformGitLab, nil
	default:
		return "", fmt.Errorf("unsupported platform %q: must be 'github' or 'gitlab'", s)
	}
}

func ParseDestinations(raw string) ([]Destination, error) {
	if raw == "" {
		return nil, fmt.Errorf("--dest is required")
	}

	var dests []Destination
	for _, part := range splitComma(raw) {
		platform, owner, err := splitPlatformOwner(part)
		if err != nil {
			return nil, fmt.Errorf("invalid destination %q: %w", part, err)
		}
		dests = append(dests, Destination{Platform: platform, Owner: owner})
	}
	return dests, nil
}

func splitPlatformOwner(s string) (Platform, string, error) {
	for i, c := range s {
		if c == ':' {
			p, err := ParsePlatform(s[:i])
			if err != nil {
				return "", "", err
			}
			owner := s[i+1:]
			if owner == "" {
				return "", "", fmt.Errorf("owner cannot be empty")
			}
			return p, owner, nil
		}
	}
	return "", "", fmt.Errorf("expected format 'platform:owner' (e.g. github:myuser)")
}

func splitComma(s string) []string {
	var parts []string
	current := ""
	for _, c := range s {
		if c == ',' {
			if current != "" {
				parts = append(parts, current)
			}
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
