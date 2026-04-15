package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Repository represents a GitHub repository that implements the Ensurable interface
type Repository struct {
	// Name of the repository
	Name string

	// Standard Pulumi GitHub repository arguments
	*github.RepositoryArgs

	// Branch protection rules (pattern -> protection args)
	// If nil, defaults apply based on context
	BranchProtectionRules BranchProtectionRules

	// Optional provider for standalone mode
	// If nil, uses Pulumi's default provider
	Provider *github.Provider

	// Parent organization for org-mode
	// Used for defaults resolution and provider inheritance
	// If nil, repository is in standalone mode
	organization *Organization
}

// BranchProtectionRules maps branch patterns to protection configurations
type BranchProtectionRules map[string]*github.BranchProtectionArgs

// Ensure provisions the repository and its branch protection rules
func (r *Repository) Ensure(ctx *pulumi.Context) error {
	// Apply repository defaults
	if r.HasWiki == nil {
		r.HasWiki = pulumi.Bool(false)
	}

	if r.HasDiscussions == nil {
		r.HasDiscussions = pulumi.Bool(false)
	}

	if r.HasIssues == nil {
		r.HasIssues = pulumi.Bool(true)
	}

	if r.AutoInit == nil {
		r.AutoInit = pulumi.Bool(true)
	}

	// Set repository name
	r.RepositoryArgs.Name = pulumi.String(r.Name)

	// Get provider to use
	provider := r.getProvider()

	// Create provider option
	var opts []pulumi.ResourceOption
	if provider != nil {
		opts = append(opts, pulumi.Provider(provider))
	}

	// Create repository
	resourceName := fmt.Sprintf("github_repository.%s", helpers.Slugify(r.Name))
	repo, err := github.NewRepository(ctx, resourceName, r.RepositoryArgs, opts...)
	if err != nil {
		return err
	}

	// Get effective branch protection rules
	branchRules := r.getBranchProtectionRules(repo)

	// Create branch protection rules
	for pattern, args := range branchRules {
		resourceName := fmt.Sprintf("github_branch_protection.%s.%s",
			helpers.Slugify(r.Name), helpers.Slugify(pattern))
		if _, err := github.NewBranchProtection(ctx, resourceName, args, opts...); err != nil {
			return err
		}
	}

	return nil
}

// getProvider returns the appropriate provider for this repository
func (r *Repository) getProvider() *github.Provider {
	if r.organization != nil {
		// Org mode: use org's provider
		return r.organization.provider
	}
	// Standalone mode: use explicit provider or nil for default
	return r.Provider
}

// getBranchProtectionRules returns the effective branch protection rules
// using the three-tier precedence system: repo-specific > org defaults > built-in defaults
func (r *Repository) getBranchProtectionRules(repo *github.Repository) BranchProtectionRules {
	// If repo has explicit rules, use those
	if r.BranchProtectionRules != nil {
		result := make(BranchProtectionRules)
		for pattern, args := range r.BranchProtectionRules {
			result[pattern] = copyBranchProtectionArgs(args, repo.ID(), pattern)
		}
		return result
	}

	// Check for org defaults
	var template *github.BranchProtectionArgs
	if r.organization != nil && r.organization.DefaultBranchProtection != nil {
		template = r.organization.DefaultBranchProtection
	} else {
		template = builtInDefaultBranchProtection()
	}

	return BranchProtectionRules{
		defaultBranchName: copyBranchProtectionArgs(template, repo.ID(), defaultBranchName),
	}
}
