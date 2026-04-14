package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type (
	RepositoryConfig struct {
		*github.RepositoryArgs
		BranchProtectionArgs map[string]*github.BranchProtectionArgs
	}
)

var (
	defaultBranchName = "main"

	org = &github.OrganizationSettingsArgs{
		BillingEmail: pulumi.String("lars@oddbit.com"),
		Blog:         pulumi.String("https://baystateradio.org/news/"),
		Description:  pulumi.String("Amateur radio information for Eastern Massachusetts and beyond"),
		Location:     pulumi.String("Boston, MA"),
	}

	repositories = map[string]*RepositoryConfig{
		"github-config": {
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Manage github configuration for baystateradio organization"),
				AutoInit:    pulumi.Bool(false),
			},
		},
		"baystateradio.org": {
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Sources for baystateradio.org website"),
				HomepageUrl: pulumi.String("https://baystateradio.org"),
				AutoInit:    pulumi.Bool(false),
				Pages: &github.RepositoryPagesArgs{
					BuildType: pulumi.String("workflow"),
					Cname:     pulumi.String("baystateradio.org"),
				},
			},
		},
	}

	nonAlphanumeric = regexp.MustCompile(`[^a-z0-9_-]+`)
)

func slugify(v string) string {
	return nonAlphanumeric.ReplaceAllString(strings.ToLower(v), "_")
}

func defaultBranchProtectionArgs(repo *github.Repository, pattern string) *github.BranchProtectionArgs {
	return &github.BranchProtectionArgs{
		RepositoryId:          repo.NodeId,
		Pattern:               pulumi.String(pattern),
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

func ensureRepository(ctx *pulumi.Context, repositoryName string, repositoryConfig *RepositoryConfig) error {
	repositoryConfig.Name = pulumi.String(repositoryName)

	if repositoryConfig.HasWiki == nil {
		repositoryConfig.HasWiki = pulumi.Bool(false)
	}

	if repositoryConfig.HasDiscussions == nil {
		repositoryConfig.HasDiscussions = pulumi.Bool(false)
	}

	if repositoryConfig.AutoInit == nil {
		repositoryConfig.AutoInit = pulumi.Bool(true)
	}

	resourceName := fmt.Sprintf("github_repository.%s", slugify(repositoryName))
	repo, err := github.NewRepository(ctx, resourceName, repositoryConfig.RepositoryArgs)
	if err != nil {
		return err
	}

	if repositoryConfig.BranchProtectionArgs == nil {
		resourceName := fmt.Sprintf("github_branch_protection.%s.%s", slugify(repositoryName), defaultBranchName)
		if _, err := github.NewBranchProtection(ctx, resourceName, defaultBranchProtectionArgs(repo, defaultBranchName)); err != nil {
			return err
		}
	} else {
		for pattern, branchProtectionArgs := range repositoryConfig.BranchProtectionArgs {
			resourceName := fmt.Sprintf("github_branch_protection.%s.%s", slugify(repositoryName), slugify(pattern))
			branchProtectionArgs.RepositoryId = repo.NodeId
			branchProtectionArgs.Pattern = pulumi.String(pattern)
			if _, err := github.NewBranchProtection(ctx, resourceName, branchProtectionArgs); err != nil {
				return err
			}
		}
	}

	return nil
}

func ensureOrganization(ctx *pulumi.Context, name string, config *github.OrganizationSettingsArgs) error {
	config.Name = pulumi.String(name)
	resourceName := fmt.Sprintf("github_organization_settings.%s", slugify(name))
	if _, err := github.NewOrganizationSettings(ctx, resourceName, config); err != nil {
		return err
	}

	return nil
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		if err := ensureOrganization(ctx, "baystateradio", org); err != nil {
			return err
		}
		names := make([]string, 0, len(repositories))
		for name := range repositories {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			config := repositories[name]
			if err := ensureRepository(ctx, name, config); err != nil {
				return err
			}
		}
		return nil
	})
}
