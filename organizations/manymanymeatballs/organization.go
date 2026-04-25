package manymanymeatballs

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Organization defines the manymanymeatballs GitHub organization configuration
var Organization = api.Organization{
	Owner: api.Owner{
		Name:                 "manymanymeatballs",
		GithubProviderConfig: api.NewGithubProviderConfig(),
	},

	Settings: &github.OrganizationSettingsArgs{
		BillingEmail: pulumi.String("lars@oddbit.com"),
		Description:  pulumi.String("Mmm, meatballs."),
		Location:     pulumi.String("Boston, MA"),
	},

	Members: api.Members{
		"larsks": api.PermissionAdmin,
	},

	Repositories: []*api.Repository{
		{
			Name:          "manymanymeatballs.com",
			DefaultBranch: "gh-pages",
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Sources for manymanymeatballs.com website"),
				HomepageUrl: pulumi.String("https://manymanymeatballs.com"),
				AutoInit:    pulumi.Bool(false),
				Pages: &github.RepositoryPagesArgs{
					BuildType: pulumi.String(api.PagesBuildLegacy),
					Cname:     pulumi.String("manymanymeatballs.com"),
					Source: &github.RepositoryPagesSourceArgs{
						Branch: pulumi.String("gh-pages"),
						Path:   pulumi.String("/"),
					},
				},
			},
			BranchProtectionRules: api.BranchProtectionRules{},
		},
	},
}
