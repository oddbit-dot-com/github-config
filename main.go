package main

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	repositories = map[string]*github.RepositoryArgs{
		"github-config": {
			Description: pulumi.String("Manage github configuration for baystateradio organization"),
		},
		"baystateradio.org": {
			Description: pulumi.String("Sources for baystateradio.org website"),
			HomepageUrl: pulumi.String("https://baystateradio.org"),
			Pages: &github.RepositoryPagesArgs{
				BuildType: pulumi.String("workflow"),
				Cname:     pulumi.String("baystateradio.org"),
			},
		},
	}
)

func ensureRepository(ctx *pulumi.Context, name string, args *github.RepositoryArgs) (*github.Repository, error) {
	if args == nil {
		args = &github.RepositoryArgs{}
	}

	args.Name = pulumi.String(name)

	if args.HasWiki == nil {
		args.HasWiki = pulumi.Bool(false)
	}

	if args.HasDiscussions == nil {
		args.HasDiscussions = pulumi.Bool(false)
	}

	if args.AutoInit == nil {
		args.AutoInit = pulumi.Bool(true)
	}

	resourceName := fmt.Sprintf("module.repo_%s.github_repository", strings.ToLower(strings.ReplaceAll(name, ".", "_")))
	return github.NewRepository(ctx, resourceName, args)
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		for name, args := range repositories {
			_, err := ensureRepository(ctx, name, args)
			if err != nil {
				return err
			}
		}
		return nil
	})
}
