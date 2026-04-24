// organizations/caara_races/organization.go
package caara_races

import (
	"os"

	"github.com/oddbit-dot-com/github-config/api"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var Organization = api.Organization{
	Name:                 "caara-races",
	GithubProviderConfig: api.NewGithubProviderConfig(),
	VaultProviderConfig: api.NewVaultProviderConfig().
		WithMountPoint("caara-races").
		WithToken(os.Getenv("VAULT_TOKEN_CAARA_RACES")).
		WithJWTMount("github-actions").
		WithJWT(os.Getenv("VAULT_JWT_CAARA_RACES")).
		WithJWTRole("caara-races-reader"),

	Settings: &github.OrganizationSettingsArgs{
		BillingEmail: pulumi.String("lars@oddbit.com"),
	},

	Members: api.Members{
		"larsks": api.PermissionAdmin,
	},

	Repositories: []*api.Repository{
		{
			Name: "caara-races-website",
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("Proposed website for CAARA race and event support"),
				AutoInit:    pulumi.Bool(false),
				HomepageUrl: pulumi.String("https://caara-races.oddbit.com/"),
				Pages: &github.RepositoryPagesArgs{
					BuildType: pulumi.String(api.PageBuildWorkflow),
				},
			},
			Secrets: api.ActionsSecrets{
				"RCLONE_CLIENT_SECRET": api.VaultSecretRef{
					Path: "github-publish-workflow",
					Key:  "client-secret",
				},
				"GOOGLE_GEOCODING_API_KEY": api.VaultSecretRef{
					Path: "google/geocoding-api",
					Key:  "api-key",
				},
				"MAPTILER_API_KEY": api.VaultSecretRef{
					Path: "maptiler",
					Key:  "api-key",
				},
				"GOOGLE_SHEETS_READER_SA": api.VaultSecretRef{
					Path:     "google/sheets",
					Key:      "service-account",
					Encoding: api.EncodingBase64,
				},
			},
		},
		{
			Name: "members-only",
			RepositoryArgs: &github.RepositoryArgs{
				AutoInit:   pulumi.Bool(false),
				Visibility: pulumi.String("private"),
			},
			BranchProtectionRules: api.BranchProtectionRules{},
			Teams: map[string]string{
				"webworkers": "push",
			},
		},
	},

	Teams: api.Teams{
		"webworkers": api.Team{
			Settings: &github.TeamArgs{
				Description: pulumi.String("Website maintainers"),
			},
			Members: map[string]string{
				"larsks": api.MembershipMaintainer,
			},
		},
	},
}
