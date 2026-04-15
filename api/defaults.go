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
