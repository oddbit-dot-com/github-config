// organizations/caara_races/organization.go
package caara_races

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var Organization = api.Organization{
	Name:           "caara-races",
	ProviderConfig: api.ProviderFromEnv("GITHUB_TOKEN_CAARA_RACES"),

	Settings: &github.OrganizationSettingsArgs{
		BillingEmail: pulumi.String("lars@oddbit.com"),
	},

	Members: api.Members{
		"larsks": "admin",
	},

	Repositories: api.Repositories{
		"caara-races-website": {
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Proposed website for CAARA race and event support"),
				AutoInit:    pulumi.Bool(false),
				HomepageUrl: pulumi.String("https://caara-races.oddbit.com/"),
				Pages: &github.RepositoryPagesArgs{
					BuildType: pulumi.String("workflow"),
				},
			},
		},
		"members-only": {
			RepositoryArgs: &github.RepositoryArgs{
				AutoInit:   pulumi.Bool(false),
				Visibility: pulumi.String("private"),
			},
			BranchProtectionRules: api.BranchProtectionRules{},
		},
	},
}
