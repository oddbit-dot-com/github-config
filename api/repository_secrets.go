package api

import (
	"fmt"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	vault "github.com/pulumi/pulumi-vault/sdk/v6/go/vault"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func (r *Repository) ensureEnvironments(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	for envName, args := range r.Environments {
		if args == nil {
			args = &github.RepositoryEnvironmentArgs{}
		}
		argsCopy := *args
		argsCopy.Repository = repo.Name
		argsCopy.Environment = pulumi.String(envName)

		resourceName := r.resourceName("github_repository_environment", envName)
		if _, err := github.NewRepositoryEnvironment(ctx, resourceName, &argsCopy, opts...); err != nil {
			return fmt.Errorf("failed to create environment %s for %s: %w", envName, r.Name, err)
		}
	}
	return nil
}

func (r *Repository) ensureRepoSecrets(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	if len(r.Secrets) == 0 {
		return nil
	}
	vaultProvider, err := r.getVaultProvider()
	if err != nil {
		return err
	}

	return provisionSecrets(ctx, r.owner.VaultProviderConfig.MountPoint, vaultProvider, r.Secrets,
		func(secretName string, value pulumi.StringPtrInput) error {
			resourceName := r.resourceName("github_actions_secret", secretName)
			_, err := github.NewActionsSecret(ctx, resourceName, &github.ActionsSecretArgs{
				Repository:     repo.Name,
				SecretName:     pulumi.String(secretName),
				PlaintextValue: value,
			}, opts...)
			if err != nil {
				return fmt.Errorf("failed to create repo secret %s/%s: %w", r.Name, secretName, err)
			}
			return nil
		})
}

func (r *Repository) ensureEnvironmentSecrets(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	if len(r.EnvironmentSecrets) == 0 {
		return nil
	}
	vaultProvider, err := r.getVaultProvider()
	if err != nil {
		return err
	}

	mountPoint := r.owner.VaultProviderConfig.MountPoint
	for envName, secrets := range r.EnvironmentSecrets {
		if _, declared := r.Environments[envName]; !declared {
			return fmt.Errorf("environment %q referenced in EnvironmentSecrets of %s is not declared in Environments", envName, r.Name)
		}
		if err := provisionSecrets(ctx, mountPoint, vaultProvider, secrets,
			func(secretName string, value pulumi.StringPtrInput) error {
				resourceName := r.resourceName("github_actions_environment_secret", envName, secretName)
				_, err := github.NewActionsEnvironmentSecret(ctx, resourceName, &github.ActionsEnvironmentSecretArgs{
					Repository:     repo.Name,
					Environment:    pulumi.String(envName),
					SecretName:     pulumi.String(secretName),
					PlaintextValue: value,
				}, opts...)
				if err != nil {
					return fmt.Errorf("failed to create env secret %s/%s/%s: %w", r.Name, envName, secretName, err)
				}
				return nil
			}); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) getVaultProvider() (*vault.Provider, error) {
	if r.owner == nil || r.owner.vaultProvider == nil {
		return nil, fmt.Errorf("repository %s has secrets but no vault provider is configured", r.Name)
	}
	return r.owner.vaultProvider, nil
}
