package api

import (
	"fmt"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	vault "github.com/pulumi/pulumi-vault/sdk/v6/go/vault"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// ownerKind is implemented by Organization and User to identify which GitHub
// API namespace ("orgs" or "users") to use when looking up app installations.
type ownerKind interface {
	githubAPIKind() string
}

// Owner holds configuration common to both Organization and User.
// The name mirrors GitHub API terminology where an owner is either a user or an org.
type Owner struct {
	Name                    string
	GithubProviderConfig    *GithubProviderConfig
	DefaultBranchProtection *github.BranchProtectionArgs
	Labels                  IssueLabels
	VaultProviderConfig     *VaultProviderConfig
	githubProvider          *github.Provider
	vaultProvider           *vault.Provider
}

func (o *Owner) ensureGithubProvider(ctx *pulumi.Context, kind ownerKind) error {
	if o.GithubProviderConfig == nil {
		o.GithubProviderConfig = NewGithubProviderConfig()
	}
	provider, err := CreateGitHubProvider(ctx, o.GithubProviderConfig, o.Name, "", kind.githubAPIKind())
	if err != nil {
		return fmt.Errorf("failed to create github provider for %s: %w", o.Name, err)
	}
	o.githubProvider = provider
	return nil
}

func (o *Owner) ensureVaultProvider(ctx *pulumi.Context) error {
	vaultProvider, err := CreateVaultProvider(ctx, o.VaultProviderConfig, o.Name)
	if err != nil {
		return fmt.Errorf("failed to create vault provider for %s: %w", o.Name, err)
	}
	o.vaultProvider = vaultProvider
	return nil
}

func (o *Owner) ensureRepositories(ctx *pulumi.Context, repos []*Repository) error {
	for _, repo := range repos {
		repo.owner = o
		if err := repo.ensure(ctx); err != nil {
			return fmt.Errorf("failed to ensure repository %s/%s: %w", o.Name, repo.Name, err)
		}
	}
	return nil
}
