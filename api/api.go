package api

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

type (
	Ensurable interface {
		Ensure(ctx *pulumi.Context) error
	}
)
