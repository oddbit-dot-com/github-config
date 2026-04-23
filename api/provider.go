package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// GithubProviderConfig holds GitHub provider configuration
type GithubProviderConfig struct {
	// GitHub token. If nil, uses default (GITHUB_TOKEN env var or gh CLI)
	Token *string

	// GitHub owner (organization or user). Used when creating providers.
	// In org-mode: if nil, falls back to organization name
	// In standalone mode: required to create a provider from GithubProviderConfig
	Owner *string
}

// NewGithubProviderConfig creates a new GithubProviderConfig with no defaults set.
func NewGithubProviderConfig() *GithubProviderConfig {
	return &GithubProviderConfig{}
}

// WithToken sets the GitHub token. If token is empty, it is ignored and the
// default token resolution (GITHUB_TOKEN env var or gh CLI) is used.
func (c *GithubProviderConfig) WithToken(token string) *GithubProviderConfig {
	if token != "" {
		c.Token = &token
	}
	return c
}

// WithOwner sets the GitHub owner (organization or user).
func (c *GithubProviderConfig) WithOwner(owner string) *GithubProviderConfig {
	c.Owner = &owner
	return c
}

// CreateGitHubProvider creates a GitHub provider for the given configuration.
// If config is nil or config.Owner is nil, uses defaultOwner as the owner.
// Returns (nil, nil) if no owner can be determined (both config.Owner and defaultOwner are empty).
//
// The resourceNameSuffix parameter should contain additional parts to append to the resource name:
// - For organizations: pass the org name as defaultOwner and empty suffix
// - For standalone repositories: pass empty defaultOwner and repo name as suffix
func CreateGitHubProvider(
	ctx *pulumi.Context,
	config *GithubProviderConfig,
	defaultOwner string,
	resourceNameSuffix string,
) (*github.Provider, error) {
	// Determine owner
	owner := defaultOwner
	if config != nil && config.Owner != nil {
		owner = *config.Owner
	}

	// If no owner determined, return nil (standalone mode without ProviderConfig.Owner)
	if owner == "" {
		return nil, nil
	}

	// Create provider args
	providerArgs := &github.ProviderArgs{
		Owner: pulumi.String(owner),
	}

	// Set token if provided
	if config != nil && config.Token != nil {
		providerArgs.Token = pulumi.String(*config.Token)
	}

	// Build resource name
	resourceName := fmt.Sprintf("github-provider.%s", helpers.Slugify(owner))
	if resourceNameSuffix != "" {
		resourceName = fmt.Sprintf("%s.%s", resourceName, helpers.Slugify(resourceNameSuffix))
	}

	return github.NewProvider(ctx, resourceName, providerArgs) //, pulumi.IgnoreChanges([]string{"token"}))
}
