package api

import (
	"fmt"

	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Encoding string

const (
	EncodingNone   Encoding = ""
	EncodingBase64 Encoding = "base64"
)

// SecretRef resolves a secret value from an external provider.
type SecretRef interface {
	Resolve(ctx *pulumi.Context) (pulumi.StringOutput, error)
}

type OrgSecretRef struct {
	SecretRef
	Visibility string
}

type ActionsSecrets map[string]SecretRef
type OrgActionsSecrets map[string]OrgSecretRef

func provisionSecrets(
	ctx *pulumi.Context,
	secrets ActionsSecrets,
	create func(secretName string, value pulumi.StringOutput) error,
) error {
	for secretName, ref := range secrets {
		value, err := ref.Resolve(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve secret for %s: %w", secretName, err)
		}
		if err := create(secretName, value); err != nil {
			return err
		}
	}
	return nil
}

type LiteralSecretRef struct {
	Value string
}

func (r *LiteralSecretRef) Resolve(_ *pulumi.Context) (pulumi.StringOutput, error) {
	return pulumi.String(r.Value).ToStringOutput(), nil
}
