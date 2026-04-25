package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func (o *Organization) ensureOrgSecrets(ctx *pulumi.Context, githubProvider *github.Provider) error {
	if len(o.Secrets) == 0 {
		return nil
	}

	if o.vaultProvider != nil {
		bindOrgVaultSecrets(o.Secrets, o.vaultProvider, o.VaultProviderConfig.MountPoint)
	}

	var githubOpts []pulumi.ResourceOption
	if githubProvider != nil {
		githubOpts = append(githubOpts, pulumi.Provider(githubProvider))
	}

	for secretName, ref := range o.Secrets {
		value, err := ref.Resolve(ctx)
		if err != nil {
			return fmt.Errorf("failed to resolve secret for %s/%s: %w", o.Name, secretName, err)
		}

		visibility := ref.Visibility
		if visibility == "" {
			visibility = VisibilityAll
		}

		resourceName := helpers.ResourceName("github_actions_organization_secret", o.Name, secretName)
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
