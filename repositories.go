package main

import (
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
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
)
