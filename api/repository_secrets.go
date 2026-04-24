package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
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

		resourceName := fmt.Sprintf("github_repository_environment.%s.%s",
			helpers.Slugify(r.Name), helpers.Slugify(envName))
		if _, err := github.NewRepositoryEnvironment(ctx, resourceName, &argsCopy, opts...); err != nil {
			return fmt.Errorf("failed to create environment %s for %s: %w", envName, r.Name, err)
		}
	}
	return nil
}

func (r *Repository) ensureRepoSecrets(ctx *pulumi.Context, opts []pulumi.ResourceOption) error {
	if len(r.Secrets) == 0 {
		return nil
	}
	vaultProvider, err := r.getVaultProvider()
	if err != nil {
		return err
	}

	mountPoint := r.organization.VaultProviderConfig.MountPoint
	for secretName, ref := range r.Secrets {
		value, err := readVaultSecret(ctx, mountPoint, vaultProvider, ref)
		if err != nil {
			return fmt.Errorf("failed to read vault secret for %s/%s: %w", r.Name, secretName, err)
		}

		resourceName := fmt.Sprintf("github_actions_secret.%s.%s",
			helpers.Slugify(r.Name), helpers.Slugify(secretName))
		_, err = github.NewActionsSecret(ctx, resourceName, &github.ActionsSecretArgs{
			Repository:     pulumi.String(r.Name),
			SecretName:     pulumi.String(secretName),
			PlaintextValue: value,
		}, opts...)
		if err != nil {
			return fmt.Errorf("failed to create repo secret %s/%s: %w", r.Name, secretName, err)
		}
	}
	return nil
}

func (r *Repository) ensureEnvironmentSecrets(ctx *pulumi.Context, repo *github.Repository, opts []pulumi.ResourceOption) error {
	if len(r.EnvironmentSecrets) == 0 {
		return nil
	}
	vaultProvider, err := r.getVaultProvider()
	if err != nil {
		return err
	}

	mountPoint := r.organization.VaultProviderConfig.MountPoint
	for envName, secrets := range r.EnvironmentSecrets {
		if _, declared := r.Environments[envName]; !declared {
			return fmt.Errorf("environment %q referenced in EnvironmentSecrets of %s is not declared in Environments", envName, r.Name)
		}
		for secretName, ref := range secrets {
			value, err := readVaultSecret(ctx, mountPoint, vaultProvider, ref)
			if err != nil {
				return fmt.Errorf("failed to read vault secret for %s/%s/%s: %w", r.Name, envName, secretName, err)
			}

			resourceName := fmt.Sprintf("github_actions_environment_secret.%s.%s.%s",
				helpers.Slugify(r.Name), helpers.Slugify(envName), helpers.Slugify(secretName))
			_, err = github.NewActionsEnvironmentSecret(ctx, resourceName, &github.ActionsEnvironmentSecretArgs{
				Repository:     repo.Name,
				Environment:    pulumi.String(envName),
				SecretName:     pulumi.String(secretName),
				PlaintextValue: value,
			}, opts...)
			if err != nil {
				return fmt.Errorf("failed to create env secret %s/%s/%s: %w", r.Name, envName, secretName, err)
			}
		}
	}
	return nil
}

func (r *Repository) getVaultProvider() (*vault.Provider, error) {
	if r.organization == nil || r.organization.vaultProvider == nil {
		return nil, fmt.Errorf("repository %s has secrets but its organization has no vault provider configured", r.Name)
	}
	return r.organization.vaultProvider, nil
}

