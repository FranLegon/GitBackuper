package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/FranLegon/GitBackuper/internal/backup"
	"github.com/FranLegon/GitBackuper/internal/config"
	"github.com/FranLegon/GitBackuper/internal/platform"
	"github.com/FranLegon/GitBackuper/internal/prereq"
	"github.com/FranLegon/GitBackuper/internal/source"
	"github.com/spf13/cobra"
)

var (
	sourcePlatformFlag string
	sourceOwnerFlag    string
	reposFlag          string
	sourceFlag         string
	destFlag           string
	prefixFlag         bool
	forcePrivateFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "gitbackuper",
	Short: "Backup Git repositories from GitHub/GitLab to one or more destinations",
	Long: `gitbackuper is a CLI tool that backs up repositories by wrapping git, gh, and glab.

It supports two mutually exclusive modes for specifying sources:

  Mode 1 (Flags):  --source-platform github --source-owner myorg [--repos repo1,repo2]
  Mode 2 (URI):    --source "github:myorg/repo1,github:myorg/repo2" or --source "github:myorg"

Both --prefix and --force-private must be explicitly set.`,
	RunE: runBackup,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.Flags().StringVar(&sourcePlatformFlag, "source-platform", "", "Source platform: github or gitlab")
	rootCmd.Flags().StringVar(&sourceOwnerFlag, "source-owner", "", "Source owner (user or organization/group)")
	rootCmd.Flags().StringVar(&reposFlag, "repos", "", "Comma-separated list of repo names (omit to backup all from owner)")
	rootCmd.Flags().StringVar(&sourceFlag, "source", "", "URI-style source: github:owner/repo1,github:owner/repo2 or github:owner")
	rootCmd.Flags().StringVar(&destFlag, "dest", "", "Comma-separated destinations: github:user1,gitlab:group1")
	rootCmd.Flags().BoolVar(&prefixFlag, "prefix", false, "Prepend 'backup_' to destination repo names (must be explicitly set)")
	rootCmd.Flags().BoolVar(&forcePrivateFlag, "force-private", false, "Force destination repos to be private (must be explicitly set)")

	rootCmd.MarkFlagsMutuallyExclusive("source-platform", "source")
	rootCmd.MarkFlagsMutuallyExclusive("source-owner", "source")
	rootCmd.MarkFlagsMutuallyExclusive("repos", "source")
}

func runBackup(cmd *cobra.Command, args []string) error {
	if err := prereq.Check(); err != nil {
		return err
	}

	if err := validateFlags(cmd); err != nil {
		return err
	}

	platforms := map[config.Platform]platform.Platform{
		config.PlatformGitHub: &platform.GitHub{},
		config.PlatformGitLab: &platform.GitLab{},
	}

	destinations, err := config.ParseDestinations(destFlag)
	if err != nil {
		return err
	}

	resolver := source.NewResolver(platforms)

	var repos []config.Repository
	if cmd.Flags().Changed("source") {
		repos, err = resolver.ResolveFromURIs(sourceFlag)
	} else {
		p, pErr := config.ParsePlatform(sourcePlatformFlag)
		if pErr != nil {
			return pErr
		}
		repoNames := source.ParseRepoNames(reposFlag)
		repos, err = resolver.ResolveFromFlags(p, sourceOwnerFlag, repoNames)
	}
	if err != nil {
		return fmt.Errorf("resolving sources: %w", err)
	}

	if len(repos) == 0 {
		return fmt.Errorf("no source repositories found")
	}

	cfg := &config.BackupConfig{
		Sources:      repos,
		Destinations: destinations,
		Prefix:       prefixFlag,
		ForcePrivate: forcePrivateFlag,
	}

	log.Printf("Starting backup: %d repos -> %d destinations (prefix=%v, force-private=%v)",
		len(cfg.Sources), len(cfg.Destinations), cfg.Prefix, cfg.ForcePrivate)

	executor := backup.NewExecutor(cfg, platforms)
	result := executor.Run()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "Backup complete: %d succeeded, %d failed\n",
		len(result.Succeeded), len(result.Failed))

	if len(result.Failed) > 0 {
		fmt.Fprintln(os.Stderr, "\nFailed repositories:")
		for repo, err := range result.Failed {
			fmt.Fprintf(os.Stderr, "  - %s: %v\n", repo, err)
		}
		os.Exit(1)
	}
	return nil
}

func validateFlags(cmd *cobra.Command) error {
	if !cmd.Flags().Changed("prefix") {
		return fmt.Errorf("--prefix must be explicitly set (e.g. --prefix=true or --prefix=false)")
	}
	if !cmd.Flags().Changed("force-private") {
		return fmt.Errorf("--force-private must be explicitly set (e.g. --force-private=true or --force-private=false)")
	}
	if !cmd.Flags().Changed("dest") || destFlag == "" {
		return fmt.Errorf("--dest is required (e.g. --dest github:myuser,gitlab:mygroup)")
	}

	mode1 := cmd.Flags().Changed("source-platform") || cmd.Flags().Changed("source-owner") || cmd.Flags().Changed("repos")
	mode2 := cmd.Flags().Changed("source")

	if !mode1 && !mode2 {
		return fmt.Errorf("must specify source repositories using either:\n" +
			"  Mode 1: --source-platform <github|gitlab> --source-owner <owner> [--repos repo1,repo2]\n" +
			"  Mode 2: --source <platform:owner[/repo],...>")
	}

	if mode1 {
		if !cmd.Flags().Changed("source-platform") {
			return fmt.Errorf("--source-platform is required when using flags-based mode")
		}
		if !cmd.Flags().Changed("source-owner") {
			return fmt.Errorf("--source-owner is required when using flags-based mode")
		}
		if _, err := config.ParsePlatform(sourcePlatformFlag); err != nil {
			return err
		}
		if strings.TrimSpace(sourceOwnerFlag) == "" {
			return fmt.Errorf("--source-owner cannot be empty")
		}
	}

	return nil
}
