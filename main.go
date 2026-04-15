package main

import (
	"github.com/oddbit-dot-com/github-config/api"
	"github.com/oddbit-dot-com/github-config/organizations/baystateradio"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		organizations := []*api.Organization{
			&baystateradio.Organization,
		}

		for _, org := range organizations {
			if err := org.Ensure(ctx); err != nil {
				return err
			}
		}

		return nil
	})
}
