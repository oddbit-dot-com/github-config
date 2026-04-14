package api

import (
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const defaultBranchName = "main"

// builtInDefaultBranchProtection returns the default branch protection configuration
// matching the current behavior in the codebase
func builtInDefaultBranchProtection() *github.BranchProtectionArgs {
	return &github.BranchProtectionArgs{
		RequiredLinearHistory: pulumi.Bool(true),
		AllowsForcePushes:     pulumi.Bool(false),
		EnforceAdmins:         pulumi.Bool(false),
		ForcePushBypassers:    pulumi.ToStringArray([]string{"/larsks"}),
		RequiredPullRequestReviews: &github.BranchProtectionRequiredPullRequestReviewArray{
			github.BranchProtectionRequiredPullRequestReviewArgs{
				RequiredApprovingReviewCount: pulumi.Int(1),
			},
		},
	}
}

// copyBranchProtectionArgs creates a fresh BranchProtectionArgs instance with the given
// repositoryId and pattern, copying settings from the template to avoid shared state mutation
func copyBranchProtectionArgs(template *github.BranchProtectionArgs, repoID pulumi.IDOutput, pattern string) *github.BranchProtectionArgs {
	return &github.BranchProtectionArgs{
		// Set these first - they're repository-specific
		RepositoryId: repoID,
		Pattern:      pulumi.String(pattern),

		// Copy all fields from template
		RequiredLinearHistory:      template.RequiredLinearHistory,
		AllowsForcePushes:          template.AllowsForcePushes,
		EnforceAdmins:              template.EnforceAdmins,
		ForcePushBypassers:         template.ForcePushBypassers,
		RequiredPullRequestReviews: template.RequiredPullRequestReviews,
	}
}

// GetBranchProtectionRules returns the effective branch protection rules for a repository
// using the three-tier precedence system: repo-specific > org defaults > built-in defaults
// Returns fresh instances with RepositoryId and Pattern already set to prevent shared state mutation
func (o *Organization) GetBranchProtectionRules(repo *github.Repository, repoConfig *RepositoryConfig) BranchProtectionRules {
	var template *github.BranchProtectionArgs

	// Determine which template to use (repo-specific > org default > built-in)
	if repoConfig.BranchProtectionRules != nil {
		// Repo has explicit rules - copy each one
		result := make(BranchProtectionRules)
		for pattern, args := range repoConfig.BranchProtectionRules {
			result[pattern] = copyBranchProtectionArgs(args, repo.ID(), pattern)
		}
		return result
	}

	if o.DefaultBranchProtection != nil {
		template = o.DefaultBranchProtection
	} else {
		template = builtInDefaultBranchProtection()
	}

	// Return fresh instance for default branch
	return BranchProtectionRules{
		defaultBranchName: copyBranchProtectionArgs(template, repo.ID(), defaultBranchName),
	}
}
