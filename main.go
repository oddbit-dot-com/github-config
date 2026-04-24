package main

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/oddbit-dot-com/github-config/organizations/baystateradio"
	"github.com/oddbit-dot-com/github-config/organizations/caara_races"
	"github.com/oddbit-dot-com/github-config/organizations/manymanymeatballs"
	"github.com/oddbit-dot-com/github-config/organizations/oddbit_dot_com"
	"github.com/oddbit-dot-com/github-config/users/larsks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		resources := []api.Ensurable{
			&baystateradio.Organization,
			&caara_races.Organization,
			&oddbit_dot_com.Organization,
			&manymanymeatballs.Organization,
			&larsks.User,
		}

		for _, org := range resources {
			if err := org.Ensure(ctx); err != nil {
				return err
			}
		}

		return nil
	})
}
