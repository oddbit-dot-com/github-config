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
	Repositories []*Repository

	// Cached provider instance (created in Ensure)
	provider *github.Provider
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

	// Store provider for repositories to use
	o.provider = provider

	// Provision organization settings
	if err := o.ensureSettings(ctx, provider); err != nil {
		return fmt.Errorf("failed to ensure settings for %s: %w", o.Name, err)
	}

	// Provision members
	if err := o.ensureMembers(ctx, provider); err != nil {
		return fmt.Errorf("failed to ensure members for %s: %w", o.Name, err)
	}

	// Provision repositories
	if err := o.ensureRepositories(ctx); err != nil {
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
	return github.NewProvider(ctx, resourceName, providerArgs, pulumi.IgnoreChanges([]string{"token"}))
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
func (o *Organization) ensureRepositories(ctx *pulumi.Context) error {
	for _, repo := range o.Repositories {
		// Set organization reference for provider and defaults inheritance
		repo.organization = o

		if err := repo.Ensure(ctx); err != nil {
			return err
		}
	}
	return nil
}
