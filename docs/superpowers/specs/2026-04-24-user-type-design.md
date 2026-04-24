# User Type Design

Date: 2026-04-24

## Goal

Add a `User` type to manage personal GitHub repositories and user-level resources (SSH keys, GPG keys) alongside the existing `Organization` type, sharing as much code as possible.

## Approach

Extract a shared `Owner` struct (embedded in both `Organization` and `User`) that holds all fields and methods common to both types. The name `Owner` mirrors GitHub API terminology, where an owner can be either a user or an organization.

## New `Owner` struct (`api/owner.go`)

Holds everything shared between `Organization` and `User`:

```go
type Owner struct {
    Name                    string
    GithubProviderConfig    *GithubProviderConfig
    DefaultBranchProtection *github.BranchProtectionArgs
    Labels                  IssueLabels
    Repositories            []*Repository
    VaultProviderConfig     *VaultProviderConfig
    githubProvider          *github.Provider
    vaultProvider           *vault.Provider
}
```

Shared methods migrate here from `Organization`:
- `ensureGithubProvider(ctx)` — creates and caches the GitHub provider
- `ensureVaultProvider(ctx)` — creates and caches the Vault provider
- `ensureRepositories(ctx)` — provisions all repositories

## Refactored `Organization` (`api/organization.go`)

Embeds `Owner` and retains only org-specific fields:

```go
type Organization struct {
    Owner
    Settings  *github.OrganizationSettingsArgs
    Members   Members
    Teams     Teams
    Secrets   OrgActionsSecrets
}
```

`Organization.Ensure()` calls shared `Owner` methods then handles settings, members, teams, and org secrets. Existing org packages compile without modification — embedded fields promote to the outer type, so `org.Name`, `org.Repositories`, etc. continue to work unchanged.

## New `User` struct (`api/user.go`)

```go
type User struct {
    Owner
    SshKeys []github.UserSshKeyArgs
    GpgKeys []github.UserGpgKeyArgs
}
```

`User.Ensure()` calls shared `Owner` methods for providers and repositories, then provisions SSH and GPG keys. Resources are named:
- `github_user_ssh_key.{username}.{title-slug}`
- `github_user_gpg_key.{username}.{key-id-slug}`

## `Repository` back-reference change (`api/repository.go`)

```go
// before
organization *Organization

// after
owner *Owner
```

All internal references (`r.organization.Name`, `r.organization.provider`, etc.) become `r.owner.Name`, `r.owner.githubProvider`, etc. No change to the public `Repository` API since the field is unexported.

## Directory structure

User definitions live under `users/` analogous to `organizations/`:

```
users/
  larsks/
    user.go     // var User = api.User{...}
```

Example user definition:

```go
package larsks

import (
    "github.com/oddbit-dot-com/github-config/api"
    "github.com/pulumi/pulumi-github/sdk/v6/go/github"
    "github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var User = api.User{
    Name:                 "larsks",
    GithubProviderConfig: api.NewGithubProviderConfig(),

    SshKeys: []github.UserSshKeyArgs{
        {Title: pulumi.String("my-key"), Key: pulumi.String("ssh-rsa ...")},
    },

    GpgKeys: []github.UserGpgKeyArgs{
        {ArmoredPublicKey: pulumi.String("-----BEGIN PGP PUBLIC KEY BLOCK-----...")},
    },

    Repositories: []*api.Repository{
        {
            Name: "my-project",
            RepositoryArgs: &github.RepositoryArgs{
                Description: pulumi.String("An example repository"),
            },
        },
    },
}
```

`main.go` adds users to the existing `[]api.Ensurable` slice:

```go
resources := []api.Ensurable{
    &baystateradio.Organization,
    &larsks.User,
    // ...
}
```

## What is NOT included

- Org-specific fields (`Members`, `Teams`, `Settings`, `OrgActionsSecrets`) — these remain on `Organization` only
- Any user-level GitHub settings beyond SSH and GPG keys (not currently supported by the Pulumi GitHub provider)
