package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Organization defines configuration for a GitHub organization
type Organization struct {
	Owner

	// Organization settings (billing email, blog, description, etc.)
	Settings *github.OrganizationSettingsArgs

	// Organization members (username -> role mapping)
	Members Members

	// Teams
	Teams Teams

	Secrets OrgActionsSecrets

	// Repository configurations
	Repositories []*Repository
}

// Members maps usernames to roles
type Members map[string]string

type Teams map[string]Team

type Team struct {
	Settings *github.TeamArgs
	Members  map[string]string
}

// Ensure provisions all resources for this organization
func (o *Organization) Ensure(ctx *pulumi.Context) error {
	if err := o.ensureGithubProvider(ctx); err != nil {
		return err
	}
	if err := o.ensureVaultProvider(ctx); err != nil {
		return err
	}
	if err := o.ensureOrgSecrets(ctx, o.githubProvider); err != nil {
		return fmt.Errorf("failed to ensure org secrets for %s: %w", o.Name, err)
	}
	if err := o.ensureSettings(ctx, o.githubProvider); err != nil {
		return fmt.Errorf("failed to ensure settings for %s: %w", o.Name, err)
	}
	if err := o.ensureMembers(ctx, o.githubProvider); err != nil {
		return fmt.Errorf("failed to ensure members for %s: %w", o.Name, err)
	}
	if err := o.ensureTeams(ctx, o.githubProvider); err != nil {
		return fmt.Errorf("failed to ensure teams for %s: %w", o.Name, err)
	}
	if err := o.ensureRepositories(ctx, o.Repositories); err != nil {
		return fmt.Errorf("failed to ensure repositories for %s: %w", o.Name, err)
	}
	return nil
}

// ensureSettings provisions organization settings
func (o *Organization) ensureSettings(ctx *pulumi.Context, provider *github.Provider) error {
	if o.Settings == nil {
		return nil
	}

	o.Settings.Name = pulumi.String(o.Name)
	resourceName := fmt.Sprintf("github_organization_settings.%s", helpers.Slugify(o.Name))
	_, err := github.NewOrganizationSettings(ctx, resourceName, o.Settings, pulumi.Provider(provider))
	if err != nil {
		return fmt.Errorf("failed to create organization settings for %s: %w", o.Name, err)
	}
	return nil
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
			return fmt.Errorf("failed to create membership for %s/%s: %w", o.Name, username, err)
		}
	}
	return nil
}
