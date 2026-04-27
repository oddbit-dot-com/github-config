package api

import (
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const defaultBranchName = "main"

// applyRepositoryDefaults sets nil repository fields to their default values
func applyRepositoryDefaults(args *github.RepositoryArgs) {
	if args.HasWiki == nil {
		args.HasWiki = pulumi.Bool(false)
	}
	if args.HasDiscussions == nil {
		args.HasDiscussions = pulumi.Bool(false)
	}
	if args.HasIssues == nil {
		args.HasIssues = pulumi.Bool(true)
	}
	if args.AutoInit == nil {
		args.AutoInit = pulumi.Bool(true)
	}
}

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

func copyBranchProtectionArgs(template *github.BranchProtectionArgs, repoID pulumi.IDOutput, pattern string) *github.BranchProtectionArgs {
	result := *template
	result.RepositoryId = repoID
	result.Pattern = pulumi.String(pattern)
	return &result
}

// DefaultIssueLabels returns GitHub's default issue labels
// These can be used as a base and merged with custom labels using MergeLabels()
// Note: Name field is omitted - it will default to the map key
func DefaultIssueLabels() IssueLabels {
	return IssueLabels{
		"bug": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("d73a4a"),
			Description: pulumi.String("Something isn't working"),
		},
		"documentation": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("0075ca"),
			Description: pulumi.String("Improvements or additions to documentation"),
		},
		"duplicate": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("cfd3d7"),
			Description: pulumi.String("This issue or pull request already exists"),
		},
		"enhancement": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("a2eeef"),
			Description: pulumi.String("New feature or request"),
		},
		"good first issue": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("7057ff"),
			Description: pulumi.String("Good for newcomers"),
		},
		"help wanted": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("008672"),
			Description: pulumi.String("Extra attention is needed"),
		},
		"invalid": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("e4e669"),
			Description: pulumi.String("This doesn't seem right"),
		},
		"question": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("d876e3"),
			Description: pulumi.String("Further information is requested"),
		},
		"wontfix": &github.IssueLabelsLabelArgs{
			Color:       pulumi.String("ffffff"),
			Description: pulumi.String("This will not be worked on"),
		},
	}
}

// copyIssueLabelArgs creates a fresh IssueLabelsLabelArgs instance,
// copying all fields from the template. If the template has a Name set,
// it uses that; otherwise it falls back to the provided labelName.
func copyIssueLabelArgs(template *github.IssueLabelsLabelArgs, labelName string) *github.IssueLabelsLabelArgs {
	// Use template Name if provided, otherwise use labelName from map key
	name := template.Name
	if name == nil {
		name = pulumi.String(labelName)
	}

	return &github.IssueLabelsLabelArgs{
		Name:        name,
		Color:       template.Color,
		Description: template.Description,
	}
}

// MergeLabels merges multiple IssueLabels maps into a single map.
// Later maps override earlier maps when label names conflict.
// This is useful for combining default labels with custom labels.
//
// Example:
//
//	Labels: api.MergeLabels(
//	    api.DefaultIssueLabels(),
//	    api.IssueLabels{
//	        "backend": &github.IssueLabelsLabelArgs{
//	            Color:       pulumi.String("0000ff"),
//	            Description: pulumi.String("Backend code"),
//	        },
//	    },
//	)
func MergeLabels(maps ...IssueLabels) IssueLabels {
	result := make(IssueLabels)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}
