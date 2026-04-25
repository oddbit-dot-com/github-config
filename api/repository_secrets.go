package api

import (
	"fmt"

	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
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

	if r.owner != nil && r.owner.vaultProvider != nil {
		bindVaultSecrets(r.Secrets, r.owner.vaultProvider, r.owner.VaultProviderConfig.MountPoint)
	}

	return provisionSecrets(ctx, r.Secrets,
		func(secretName string, value pulumi.StringOutput) error {
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

	for envName, secrets := range r.EnvironmentSecrets {
		if _, declared := r.Environments[envName]; !declared {
			return fmt.Errorf("environment %q referenced in EnvironmentSecrets of %s is not declared in Environments", envName, r.Name)
		}
		if r.owner != nil && r.owner.vaultProvider != nil {
			bindVaultSecrets(secrets, r.owner.vaultProvider, r.owner.VaultProviderConfig.MountPoint)
		}
		if err := provisionSecrets(ctx, secrets,
			func(secretName string, value pulumi.StringOutput) error {
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
