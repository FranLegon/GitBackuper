# gitbackuper

A cross-platform CLI tool that backs up Git repositories from GitHub/GitLab to one or more destination platforms by wrapping `git`, `gh`, and `glab`.

## Prerequisites

- Go 1.21+
- [GitHub CLI (`gh`)](https://cli.github.com/) — authenticated via `gh auth login`
- [GitLab CLI (`glab`)](https://gitlab.com/gitlab-org/cli) — authenticated via `glab auth login`
- `git` available in PATH

## Installation

```bash
go install github.com/FranLegon/GitBackuper@latest
```

Or build from source:

```bash
git clone https://github.com/FranLegon/GitBackuper.git
cd GitBackuper
go build -o gitbackuper .
```

## Usage

The tool supports two mutually exclusive modes for specifying source repositories:

- **Mode 1 (Flags-based):** Specify the platform, owner, and optionally specific repos
- **Mode 2 (URI-style):** Use a compact `platform:owner[/repo]` format

Both `--prefix` and `--force-private` are **mandatory** and must be explicitly set.

### Examples

#### Backup all repos from a GitHub organization to another GitHub user

```bash
gitbackuper \
  --source-platform github \
  --source-owner my-company \
  --dest "github:my-backup-account" \
  --prefix=true \
  --force-private=true
```

This backs up every repo under `my-company` to `my-backup-account` with names like `backup_repo1`, `backup_repo2`, all as private repositories.

#### Backup specific repos without the prefix

```bash
gitbackuper \
  --source-platform github \
  --source-owner myuser \
  --repos "webapp,api-server,docs" \
  --dest "github:backup-org" \
  --prefix=false \
  --force-private=true
```

Only backs up `webapp`, `api-server`, and `docs`, keeping their original names but forcing private visibility.

#### Backup to multiple destinations (cross-platform)

```bash
gitbackuper \
  --source-platform github \
  --source-owner my-company \
  --repos "critical-service" \
  --dest "github:disaster-recovery,gitlab:dr-group" \
  --prefix=true \
  --force-private=true
```

Pushes `critical-service` to both GitHub (`disaster-recovery/backup_critical-service`) and GitLab (`dr-group/backup_critical-service`).

#### URI-style: backup specific repos

```bash
gitbackuper \
  --source "github:my-company/frontend,github:my-company/backend" \
  --dest "gitlab:backup-group" \
  --prefix=true \
  --force-private=false
```

Backs up `frontend` and `backend` to GitLab, mirroring the original visibility (public repos stay public).

#### URI-style: backup all repos from an owner

```bash
gitbackuper \
  --source "gitlab:my-team" \
  --dest "github:archive-user" \
  --prefix=false \
  --force-private=true
```

Backs up all repos from the GitLab group `my-team` to a GitHub user, keeping original names, all private.

#### Mirror visibility from source

```bash
gitbackuper \
  --source-platform github \
  --source-owner open-source-org \
  --dest "github:my-mirror" \
  --prefix=true \
  --force-private=false
```

Public repos in `open-source-org` will remain public in `my-mirror` (as `backup_<name>`), and private repos will remain private.

#### Backup from GitLab to both GitHub and GitLab

```bash
gitbackuper \
  --source "gitlab:engineering-team" \
  --dest "github:eng-backups,gitlab:eng-archive" \
  --prefix=true \
  --force-private=true
```

All repos from the GitLab `engineering-team` group are backed up to two destinations simultaneously.

#### My personal use case: backup all repos from my GitHub user to both GitHub and GitLab, with prefix and forced private visibility

```bash
go build -o bin\gitbackuper.exe main.go; bin\gitbackuper.exe --source "github:FranLegon" --dest "github:FranLegon-Org,gitlab:FranLegon,gitlab:franlegon-backups" --prefix=true --force-private=true
```

## Flags Reference

| Flag | Required | Description |
|------|----------|-------------|
| `--source-platform` | Mode 1 | Source platform: `github` or `gitlab` |
| `--source-owner` | Mode 1 | Source owner (user or organization/group) |
| `--repos` | No | Comma-separated list of repo names (omit to backup all) |
| `--source` | Mode 2 | URI-style source: `platform:owner[/repo],...` |
| `--dest` | Yes | Comma-separated destinations: `platform:owner,...` |
| `--prefix` | Yes | If `true`, prepend `backup_` to destination repo names |
| `--force-private` | Yes | If `true`, force all destination repos to be private. If `false`, mirror original visibility |

## Behavior Details

### Divergence Handling

When pushing to a destination repository that already exists, gitbackuper checks if the source history has diverged from the destination (i.e., the destination contains commits that are not ancestors of the source). If divergence is detected, the tool:

1. Creates backup branches named `backup-pre-sync-<YYYYMMDD-HHMMSS>-<branch>` for each diverged branch
2. Pushes these backup branches to the destination
3. Then performs the force push

This ensures no history is lost even when force-pushing.

### Temporary Directory Cleanup

Each repository is cloned into a system temporary directory (`os.MkdirTemp`), which works on both Linux and Windows. The directory is automatically cleaned up after the push operations complete, regardless of success or failure.

### Error Handling

If a single repository fails during backup, the error is logged and processing continues with the remaining repositories. A summary of successes and failures is printed at the end.

### Authentication

gitbackuper relies on the authentication configured in `gh` and `glab`. Ensure you are logged in:

```bash
gh auth login
glab auth login
```

## License

MIT
