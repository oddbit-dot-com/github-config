package main

import (
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

var (
	org = &github.OrganizationSettingsArgs{
		BillingEmail: pulumi.String("lars@oddbit.com"),
		Blog:         pulumi.String("https://baystateradio.org/news/"),
		Description:  pulumi.String("Amateur radio information for Eastern Massachusetts and beyond"),
		Location:     pulumi.String("Boston, MA"),
	}

	members = map[string]string{
		"larsks": "admin",
	}
)
