package oddbit_dot_com

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var Organization = api.Organization{
	Name:           "oddbit-dot-com",
	ProviderConfig: api.ProviderFromEnv("GITHUB_TOKEN_ODDBIT_DOT_COM"),

	Settings: &github.OrganizationSettingsArgs{
		BillingEmail: pulumi.String("lars@oddbit.com"),
	},

	Members: api.Members{
		"larsks": "admin",
	},

	Repositories: []*api.Repository{
		{
			Name: "oddbit-infra",
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Manage oddbit general infrastructure (vault, keycloak, ...)"),
				AutoInit:    pulumi.Bool(false),
			},
		},
		{
			Name: "github-config",
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Manage github configuration for oddbit.com organizations"),
				AutoInit:    pulumi.Bool(false),
			},
		},
		{
			Name: "oddbit-dns",
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Manage oddbit.com DNS"),
			},
		},
	},
}
