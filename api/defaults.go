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

// GetBranchProtectionRules returns the effective branch protection rules for a repository
// using the three-tier precedence system: repo-specific > org defaults > built-in defaults
func (o *Organization) GetBranchProtectionRules(repo *RepositoryConfig) BranchProtectionRules {
	// If repo specifies rules, use those
	if repo.BranchProtectionRules != nil {
		return repo.BranchProtectionRules
	}

	// If org has defaults, apply to default branch
	if o.DefaultBranchProtection != nil {
		return BranchProtectionRules{
			defaultBranchName: o.DefaultBranchProtection,
		}
	}

	// Use built-in defaults
	return BranchProtectionRules{
		defaultBranchName: builtInDefaultBranchProtection(),
	}
}
