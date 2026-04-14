package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Organization defines configuration for a GitHub organization
type Organization struct {
	// Name of the organization (used as GitHub owner in provider)
	Name string

	// Optional provider configuration (token). If nil, uses default credentials.
	ProviderConfig *ProviderConfig

	// Organization settings (billing email, blog, description, etc.)
	Settings *github.OrganizationSettingsArgs

	// Organization members (username -> role mapping)
	Members Members

	// Optional default branch protection applied to all repositories
	// unless they specify their own BranchProtectionRules
	DefaultBranchProtection *github.BranchProtectionArgs

	// Repository configurations
	Repositories Repositories
}

// Members maps usernames to roles
type Members map[string]string

// Ensure provisions all resources for this organization
func (o *Organization) Ensure(ctx *pulumi.Context) error {
	// Create organization-specific provider
	provider, err := o.createProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to create provider for %s: %w", o.Name, err)
	}

	// Provision organization settings
	if err := o.ensureSettings(ctx, provider); err != nil {
		return fmt.Errorf("failed to ensure settings for %s: %w", o.Name, err)
	}

	// Provision members
	if err := o.ensureMembers(ctx, provider); err != nil {
		return fmt.Errorf("failed to ensure members for %s: %w", o.Name, err)
	}

	// Provision repositories
	if err := o.ensureRepositories(ctx, provider); err != nil {
		return fmt.Errorf("failed to ensure repositories for %s: %w", o.Name, err)
	}

	return nil
}

// createProvider creates a GitHub provider for this organization
func (o *Organization) createProvider(ctx *pulumi.Context) (*github.Provider, error) {
	providerArgs := &github.ProviderArgs{
		Owner: pulumi.String(o.Name),
	}

	// Set token if provided
	if o.ProviderConfig != nil && o.ProviderConfig.Token != nil {
		providerArgs.Token = pulumi.String(*o.ProviderConfig.Token)
	}

	resourceName := fmt.Sprintf("github-provider.%s", helpers.Slugify(o.Name))
	return github.NewProvider(ctx, resourceName, providerArgs)
}

// ensureSettings provisions organization settings
func (o *Organization) ensureSettings(ctx *pulumi.Context, provider *github.Provider) error {
	if o.Settings == nil {
		return nil
	}

	o.Settings.Name = pulumi.String(o.Name)
	resourceName := fmt.Sprintf("github_organization_settings.%s", helpers.Slugify(o.Name))
	_, err := github.NewOrganizationSettings(ctx, resourceName, o.Settings, pulumi.Provider(provider))
	return err
}

// ensureMembers provisions organization memberships
func (o *Organization) ensureMembers(ctx *pulumi.Context, provider *github.Provider) error {
	for username, role := range o.Members {
		membershipArgs := &github.MembershipArgs{
			Username: pulumi.String(username),
			Role:     pulumi.String(role),
		}
		resourceName := fmt.Sprintf("github_organization_membership.%s.%s",
			helpers.Slugify(o.Name), helpers.Slugify(username))
		if _, err := github.NewMembership(ctx, resourceName, membershipArgs, pulumi.Provider(provider)); err != nil {
			return err
		}
	}
	return nil
}

// ensureRepositories provisions repositories and their branch protection rules
func (o *Organization) ensureRepositories(ctx *pulumi.Context, provider *github.Provider) error {
	for repoName, repoConfig := range o.Repositories {
		if err := o.ensureRepository(ctx, provider, repoName, repoConfig); err != nil {
			return err
		}
	}
	return nil
}

// ensureRepository provisions a single repository and its branch protection
func (o *Organization) ensureRepository(ctx *pulumi.Context, provider *github.Provider, repoName string, repoConfig *RepositoryConfig) error {
	// Set repository name and defaults
	repoConfig.Name = pulumi.String(repoName)

	if repoConfig.HasWiki == nil {
		repoConfig.HasWiki = pulumi.Bool(false)
	}

	if repoConfig.HasDiscussions == nil {
		repoConfig.HasDiscussions = pulumi.Bool(false)
	}

	if repoConfig.AutoInit == nil {
		repoConfig.AutoInit = pulumi.Bool(true)
	}

	// Create repository
	resourceName := fmt.Sprintf("github_repository.%s", helpers.Slugify(repoName))
	repo, err := github.NewRepository(ctx, resourceName, repoConfig.RepositoryArgs, pulumi.Provider(provider))
	if err != nil {
		return err
	}

	// Get effective branch protection rules
	branchRules := o.GetBranchProtectionRules(repoConfig)

	// Create branch protection rules
	for pattern, args := range branchRules {
		args.RepositoryId = repo.ID()
		args.Pattern = pulumi.String(pattern)

		resourceName := fmt.Sprintf("github_branch_protection.%s.%s",
			helpers.Slugify(repoName), helpers.Slugify(pattern))
		if _, err := github.NewBranchProtection(ctx, resourceName, args, pulumi.Provider(provider)); err != nil {
			return err
		}
	}

	return nil
}
