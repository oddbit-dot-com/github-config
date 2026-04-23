package api

import (
	"fmt"
	"os"
	"strings"

	"github.com/oddbit-dot-com/github-config/helpers"
	vault "github.com/pulumi/pulumi-vault/sdk/v6/go/vault"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type VaultProviderConfig struct {
	Address    *string
	Token      *string
	MountPoint string
}

type VaultSecretRef struct {
	Path string
	Key  string
}

type OrgSecretRef struct {
	VaultSecretRef
	Visibility string
}

type ActionsSecrets map[string]VaultSecretRef
type OrgActionsSecrets map[string]OrgSecretRef

func NewVaultProviderConfig(address, token string, mountpoint ...string) *VaultProviderConfig {
	if address == "" {
		address = os.Getenv("VAULT_ADDR")
	}
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}
	mount := "secret"
	if len(mountpoint) > 0 && mountpoint[0] != "" {
		mount = mountpoint[0]
	}
	cfg := &VaultProviderConfig{MountPoint: strings.TrimPrefix(strings.TrimSuffix(mount, "/"), "/")}
	if address != "" {
		cfg.Address = &address
	}
	if token != "" {
		cfg.Token = &token
	}
	return cfg
}

func CreateVaultProvider(ctx *pulumi.Context, config *VaultProviderConfig, orgName string) (*vault.Provider, error) {
	if config == nil {
		return nil, nil
	}

	address := ""
	if config.Address != nil {
		address = *config.Address
	}
	if address == "" {
		address = os.Getenv("VAULT_ADDR")
	}
	if address == "" {
		return nil, fmt.Errorf("vault address not configured for %s: set VAULT_ADDR or pass an explicit address", orgName)
	}

	token := ""
	if config.Token != nil {
		token = *config.Token
	}
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}
	if token == "" {
		return nil, fmt.Errorf("vault token not configured for %s: set VAULT_TOKEN or pass an explicit token", orgName)
	}

	providerArgs := &vault.ProviderArgs{
		Address: pulumi.String(address),
		Token:   pulumi.String(token),
	}

	resourceName := fmt.Sprintf("vault-provider.%s", helpers.Slugify(orgName))
	return vault.NewProvider(ctx, resourceName, providerArgs)
}
