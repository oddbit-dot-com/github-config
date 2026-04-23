package baystateradio

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Organization defines the baystateradio GitHub organization configuration
var Organization = api.Organization{
	Name:                 "baystateradio",
	GithubProviderConfig: api.NewGithubProviderConfig(),

	Settings: &github.OrganizationSettingsArgs{
		BillingEmail: pulumi.String("lars@oddbit.com"),
		Blog:         pulumi.String("https://baystateradio.org/news/"),
		Description:  pulumi.String("Amateur radio information for Eastern Massachusetts and beyond"),
		Location:     pulumi.String("Boston, MA"),
	},

	Members: api.Members{
		"larsks": api.PermissionAdmin,
	},

	Repositories: []*api.Repository{
		{
			Name: "baystateradio.org",
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
	},
}
