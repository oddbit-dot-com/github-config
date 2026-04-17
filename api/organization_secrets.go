package api

import (
	"encoding/json"
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi-vault/sdk/v6/go/vault/kv"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func (o *Organization) ensureOrgSecrets(ctx *pulumi.Context, githubProvider *github.Provider) error {
	if len(o.Secrets) == 0 {
		return nil
	}
	if o.VaultProviderConfig == nil {
		return fmt.Errorf("organization %s has Secrets but no vault provider configured", o.Name)
	}

	var githubOpts []pulumi.ResourceOption
	if githubProvider != nil {
		githubOpts = append(githubOpts, pulumi.Provider(githubProvider))
	}

	for secretName, ref := range o.Secrets {
		value, err := o.readVaultSecret(ctx, ref.VaultSecretRef)
		if err != nil {
			return fmt.Errorf("failed to read vault secret for %s/%s: %w", o.Name, secretName, err)
		}

		visibility := ref.Visibility
		if visibility == "" {
			visibility = "all"
		}

		resourceName := fmt.Sprintf("github_actions_organization_secret.%s.%s",
			helpers.Slugify(o.Name), helpers.Slugify(secretName))
		_, err = github.NewActionsOrganizationSecret(ctx, resourceName, &github.ActionsOrganizationSecretArgs{
			SecretName:     pulumi.String(secretName),
			Visibility:     pulumi.String(visibility),
			PlaintextValue: value,
		}, githubOpts...)
		if err != nil {
			return fmt.Errorf("failed to create org secret %s/%s: %w", o.Name, secretName, err)
		}
	}
	return nil
}

func (o *Organization) readVaultSecret(ctx *pulumi.Context, ref VaultSecretRef) (pulumi.StringPtrInput, error) {
	result, err := kv.LookupSecretV2(ctx, &kv.LookupSecretV2Args{
		Mount: o.VaultProviderConfig.MountPoint,
		Name:  ref.Path,
	}, pulumi.Provider(o.vaultProvider))
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(result.DataJson), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON from vault at %s: %w", ref.Path, err)
	}
	v, ok := data[ref.Key].(string)
	if !ok {
		return nil, fmt.Errorf("key %q not found or not a string in vault secret at %s", ref.Key, ref.Path)
	}
	return pulumi.String(v).ToStringPtrOutput(), nil
}
