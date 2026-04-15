package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BranchProtectionRules maps branch patterns to protection configurations
type BranchProtectionRules map[string]*github.BranchProtectionArgs

// IssueLabels maps label names to label configurations
type IssueLabels map[string]*github.IssueLabelsLabelArgs

// Repository represents a GitHub repository that implements the Ensurable interface
type Repository struct {
	// Name of the repository
	Name string

	// Standard Pulumi GitHub repository arguments
	*github.RepositoryArgs

	// Branch protection rules (pattern -> protection args)
	// If nil, defaults apply based on context
	BranchProtectionRules BranchProtectionRules

	// Issue labels for this repository
	// If nil, inherits from organization or built-in defaults
	Labels IssueLabels

	// Optional provider for standalone mode
	// If nil, uses Pulumi's default provider
	Provider *github.Provider

	// Parent organization for org-mode
	// Used for defaults resolution and provider inheritance
	// If nil, repository is in standalone mode
	organization *Organization
}

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

	// Get effective issue labels
	issueLabels := r.getIssueLabels(repo)

	// Convert map to array for IssueLabels resource
	labelArray := make(github.IssueLabelsLabelArray, 0, len(issueLabels))
	for _, labelArgs := range issueLabels {
		labelArray = append(labelArray, labelArgs)
	}

	// Create single IssueLabels resource for all labels
	resourceName = fmt.Sprintf("github_issue_labels.%s", helpers.Slugify(r.Name))
	_, err = github.NewIssueLabels(ctx, resourceName, &github.IssueLabelsArgs{
		Repository: repo.Name,
		Labels:     labelArray,
	}, opts...)
	if err != nil {
		return err
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

// getIssueLabels returns the effective issue labels using union semantics:
// - Repository labels are merged with organization labels
// - If neither org nor repo specifies labels, use GitHub's built-in defaults
// - Repository labels override organization labels with the same name
func (r *Repository) getIssueLabels(repo *github.Repository) IssueLabels {
	result := make(IssueLabels)

	// Step 1: Add organization labels (if any)
	if r.organization != nil && r.organization.Labels != nil && len(r.organization.Labels) > 0 {
		for name, args := range r.organization.Labels {
			result[name] = copyIssueLabelArgs(args, name)
		}
	}

	// Step 2: Add/override with repository-specific labels (if any)
	if r.Labels != nil && len(r.Labels) > 0 {
		for name, args := range r.Labels {
			result[name] = copyIssueLabelArgs(args, name)
		}
	}

	// Step 3: If no labels specified anywhere, use built-in defaults
	if len(result) == 0 {
		for name, args := range builtInDefaultIssueLabels() {
			result[name] = copyIssueLabelArgs(args, name)
		}
	}

	return result
}
