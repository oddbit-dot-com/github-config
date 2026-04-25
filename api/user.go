package api

import (
	"fmt"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// User defines configuration for a personal GitHub account
type User struct {
	Owner

	// SSH keys to register on this account
	SshKeys []*github.UserSshKeyArgs

	// GPG keys to register on this account
	GpgKeys []*github.UserGpgKeyArgs

	// Repository configurations
	Repositories []*Repository
}

func (u *User) githubAPIKind() string { return "users" }

// Ensure provisions all resources for this user account
func (u *User) Ensure(ctx *pulumi.Context) error {
	if err := u.ensureGithubProvider(ctx, u); err != nil {
		return err
	}
	if err := u.ensureVaultProvider(ctx); err != nil {
		return err
	}
	if err := u.ensureSshKeys(ctx); err != nil {
		return fmt.Errorf("failed to ensure SSH keys for %s: %w", u.Name, err)
	}
	if err := u.ensureGpgKeys(ctx); err != nil {
		return fmt.Errorf("failed to ensure GPG keys for %s: %w", u.Name, err)
	}
	if err := u.ensureRepositories(ctx, u.Repositories); err != nil {
		return fmt.Errorf("failed to ensure repositories for %s: %w", u.Name, err)
	}
	return nil
}

func (u *User) keyProvisionOpts() []pulumi.ResourceOption {
	var opts []pulumi.ResourceOption
	if u.githubProvider != nil {
		opts = append(opts, pulumi.Provider(u.githubProvider))
	}
	return append(opts, pulumi.DeleteBeforeReplace(true))
}

func (u *User) ensureSshKeys(ctx *pulumi.Context) error {
	opts := u.keyProvisionOpts()
	for i, key := range u.SshKeys {
		resourceName := helpers.ResourceName("github_user_ssh_key", u.Name, fmt.Sprintf("%d", i))
		if _, err := github.NewUserSshKey(ctx, resourceName, key, opts...); err != nil {
			return fmt.Errorf("failed to create SSH key %d for %s: %w", i, u.Name, err)
		}
	}
	return nil
}

func (u *User) ensureGpgKeys(ctx *pulumi.Context) error {
	opts := u.keyProvisionOpts()
	for i, key := range u.GpgKeys {
		resourceName := helpers.ResourceName("github_user_gpg_key", u.Name, fmt.Sprintf("%d", i))
		if _, err := github.NewUserGpgKey(ctx, resourceName, key, opts...); err != nil {
			return fmt.Errorf("failed to create GPG key %d for %s: %w", i, u.Name, err)
		}
	}
	return nil
}
