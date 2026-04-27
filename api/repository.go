package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// BranchProtectionRules maps branch name patterns to their protection configurations.
// Keys are glob patterns (e.g., "main", "release/*"), values are Pulumi GitHub branch protection args.
// The rules follow a three-tier precedence: repository-specific > organization defaults > built-in defaults.
type BranchProtectionRules map[string]*github.BranchProtectionArgs

// IssueLabels maps label names to their configurations.
// Keys are label names, values are Pulumi GitHub issue label args.
// Used for both repository-specific and organization-wide label definitions.
// Labels can be merged using MergeLabels to combine org-level and repo-level labels.
type IssueLabels map[string]*github.IssueLabelsLabelArgs

// Repository represents a GitHub repository managed by an Owner (Organization or User).
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

	// Teams maps GitHub team slugs to permission levels for this repository.
	// Valid permissions: "pull", "triage", "push", "maintain", "admin"
	// The Pulumi GitHub provider accepts either a team ID or team slug in TeamId,
	// so slugs can be used directly.
	Teams map[string]string

	// Collaborators maps GitHub usernames to permission levels for this repository.
	// Valid permissions: "pull", "triage", "push", "maintain", "admin"
	Collaborators map[string]string

	// DefaultBranch sets the default branch for the repository using a separate
	// BranchDefault resource (the RepositoryArgs.DefaultBranch field is deprecated).
	DefaultBranch string

	// Secrets maps secret names to Vault secret references for GitHub Actions repo secrets.
	Secrets ActionsSecrets

	// Environments maps environment names to their configuration for this repository.
	Environments Environments

	// DeployKeys maps deploy key titles to their Pulumi configuration.
	DeployKeys DeployKeys

	// EnvironmentSecrets maps environment names to their GitHub Actions secrets,
	// where each secret value is read from Vault.
	EnvironmentSecrets EnvironmentSecrets

	// Parent owner (organization or user), set by Owner.ensureRepositories()
	owner *Owner
}

type Environments map[string]*github.RepositoryEnvironmentArgs
type DeployKey struct {
	Key      SecretRef
	ReadOnly *bool
}

type DeployKeys map[string]*DeployKey
type EnvironmentSecrets map[string]ActionsSecrets

func (r *Repository) ensure(ctx *pulumi.Context) error {
	if r.RepositoryArgs == nil {
		r.RepositoryArgs = &github.RepositoryArgs{}
	}
	applyRepositoryDefaults(r.RepositoryArgs)
	r.RepositoryArgs.Name = pulumi.String(r.Name)

	var opts []pulumi.ResourceOption
	if r.owner.githubProvider != nil {
		opts = append(opts, pulumi.Provider(r.owner.githubProvider))
	}

	// Create repository
	resourceName := r.resourceName("github_repository")
	repo, err := github.NewRepository(ctx, resourceName, r.RepositoryArgs, opts...)
	if err != nil {
		return fmt.Errorf("failed to create repository %s: %w", r.Name, err)
	}

	if r.DefaultBranch != "" {
		bdResourceName := helpers.ResourceName("github_branch_default", r.owner.Name, r.Name)
		_, err = github.NewBranchDefault(ctx, bdResourceName, &github.BranchDefaultArgs{
			Repository: repo.Name,
			Branch:     pulumi.String(r.DefaultBranch),
		}, opts...)
		if err != nil {
			return fmt.Errorf("failed to set default branch for %s: %w", r.Name, err)
		}
	}

	// Get effective branch protection rules
	branchRules := r.getBranchProtectionRules(repo)

	// Create branch protection rules
	for pattern, args := range branchRules {
		resourceName := r.resourceName("github_branch_protection", pattern)

		// By default pulumi attempts to create before delete when performing a replace
		// operation, but you can't have more than one branch protection rule for the
		// same branch so the initial create fails. Set DeleteBeforeReplace to reverse
		// the order of operation.
		bpOpts := append(opts, pulumi.DeleteBeforeReplace(true))
		if _, err := github.NewBranchProtection(ctx, resourceName, args, bpOpts...); err != nil {
			return fmt.Errorf("failed to create branch protection for %s (pattern %s): %w", r.Name, pattern, err)
		}
	}

	if err := r.ensureTeamAccess(ctx, repo, opts); err != nil {
		return err
	}

	if err := r.ensureCollaborators(ctx, repo, opts); err != nil {
		return err
	}

	if err := r.ensureDeployKeys(ctx, repo, opts); err != nil {
		return err
	}

	// Get effective issue labels
	issueLabels := r.getIssueLabels(repo)

	// Convert map to array for IssueLabels resource
	labelArray := make(github.IssueLabelsLabelArray, 0, len(issueLabels))
	for _, labelArgs := range issueLabels {
		labelArray = append(labelArray, labelArgs)
	}

	// Create single IssueLabels resource for all labels
	resourceName = r.resourceName("github_issue_labels")
	_, err = github.NewIssueLabels(ctx, resourceName, &github.IssueLabelsArgs{
		Repository: repo.Name,
		Labels:     labelArray,
	}, opts...)
	if err != nil {
		return fmt.Errorf("failed to create issue labels for %s: %w", r.Name, err)
	}

	if err := r.ensureEnvironments(ctx, repo, opts); err != nil {
		return err
	}

	if err := r.ensureRepoSecrets(ctx, repo, opts); err != nil {
		return err
	}

	if err := r.ensureEnvironmentSecrets(ctx, repo, opts); err != nil {
		return err
	}

	return nil
}

func (r *Repository) ensureTeamAccess(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	for teamSlug, permission := range r.Teams {
		resourceName := r.resourceName("github_team_repository", teamSlug)
		_, err := github.NewTeamRepository(ctx, resourceName, &github.TeamRepositoryArgs{
			TeamId:     pulumi.String(teamSlug),
			Repository: repo.Name,
			Permission: pulumi.String(permission),
		}, opts...)
		if err != nil {
			return fmt.Errorf("failed to grant team %s access to %s: %w", teamSlug, r.Name, err)
		}
	}
	return nil
}

func (r *Repository) ensureCollaborators(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	for username, permission := range r.Collaborators {
		resourceName := r.resourceName("github_repository_collaborator", username)
		_, err := github.NewRepositoryCollaborator(ctx, resourceName, &github.RepositoryCollaboratorArgs{
			Repository: repo.Name,
			Username:   pulumi.String(username),
			Permission: pulumi.String(permission),
		}, opts...)
		if err != nil {
			return fmt.Errorf("failed to add collaborator %s to %s: %w", username, r.Name, err)
		}
	}
	return nil
}

func (r *Repository) ensureDeployKeys(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	if r.owner.vaultProvider != nil {
		for _, dk := range r.DeployKeys {
			if v, ok := dk.Key.(*VaultSecretRef); ok {
				v.provider = r.owner.vaultProvider
				v.mountPoint = r.owner.VaultProviderConfig.MountPoint
			}
		}
	}

	for title, dk := range r.DeployKeys {
		key, err := dk.Key.Resolve(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve deploy key %s for %s: %w", title, r.Name, err)
		}

		readOnly := true
		if dk.ReadOnly != nil {
			readOnly = *dk.ReadOnly
		}

		resourceName := r.resourceName("github_repository_deploy_key", title)
		if _, err := github.NewRepositoryDeployKey(ctx, resourceName, &github.RepositoryDeployKeyArgs{
			Repository: repo.Name,
			Title:      pulumi.String(title),
			Key:        key,
			ReadOnly:   pulumi.Bool(readOnly),
		}, opts...); err != nil {
			return fmt.Errorf("failed to create deploy key %s for %s: %w", title, r.Name, err)
		}
	}
	return nil
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

	var template *github.BranchProtectionArgs
	if r.owner.DefaultBranchProtection != nil {
		template = r.owner.DefaultBranchProtection
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

	if len(r.owner.Labels) > 0 {
		for name, args := range r.owner.Labels {
			result[name] = copyIssueLabelArgs(args, name)
		}
	}

	if len(r.Labels) > 0 {
		for name, args := range r.Labels {
			result[name] = copyIssueLabelArgs(args, name)
		}
	}

	if len(result) == 0 {
		for name, args := range DefaultIssueLabels() {
			result[name] = copyIssueLabelArgs(args, name)
		}
	}

	return result
}

func (r *Repository) resourceName(prefix string, extra ...string) string {
	parts := []string{prefix, r.owner.Name, r.Name}
	parts = append(parts, extra...)
	return helpers.ResourceName(parts...)
}
