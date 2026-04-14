package main

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/oddbit-dot-com/github-config/organizations/baystateradio"
	"github.com/oddbit-dot-com/github-config/organizations/caara_races"
	"github.com/oddbit-dot-com/github-config/organizations/oddbit_dot_com"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		resources := []api.Ensurable{
			&baystateradio.Organization,
			&caara_races.Organization,
			&oddbit_dot_com.Organization,
		}

		for _, org := range resources {
			if err := org.Ensure(ctx); err != nil {
				return err
			}
		}

		return nil
	})
}
