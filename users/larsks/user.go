package larsks

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var User = api.User{
	Owner: api.Owner{
		Name:                 "larsks",
		GithubProviderConfig: api.NewGithubProviderConfig(),
	},
	SshKeys: []*github.UserSshKeyArgs{
		&github.UserSshKeyArgs{
			Title: pulumi.String("lars@oddbit.com"),
			Key:   pulumi.String("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFVJHRH2xg2joG1xJIrNTalRkzT6BM8rQT+OXoFiKn16"),
		},
	},

	Repositories: []*api.Repository{
		&api.Repository{
			Name:                  "hobby-spending",
			BranchProtectionRules: api.BranchProtectionRules{},
			RepositoryArgs: &github.RepositoryArgs{
				Visibility: pulumi.String(api.VisibilityPrivate),
			},
		},
		&api.Repository{
			Name:          "workinghours",
			DefaultBranch: "master",
			RepositoryArgs: &github.RepositoryArgs{
				Description: pulumi.String("A terrible utility for doing awful things to your repository history"),
			},
			BranchProtectionRules: api.BranchProtectionRules{},
		},
	},
}
