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
	JWT        *string
	JWTRole    *string
	JWTMount   *string
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

func NewVaultProviderConfig() *VaultProviderConfig {
	return &VaultProviderConfig{MountPoint: "secret"}
}

func (c *VaultProviderConfig) WithAddress(address string) *VaultProviderConfig {
	c.Address = &address
	return c
}

func (c *VaultProviderConfig) WithToken(token string) *VaultProviderConfig {
	c.Token = &token
	return c
}

func (c *VaultProviderConfig) WithMountPoint(mountpoint string) *VaultProviderConfig {
	c.MountPoint = strings.TrimPrefix(strings.TrimSuffix(mountpoint, "/"), "/")
	return c
}

func (c *VaultProviderConfig) WithJWTRole(role string) *VaultProviderConfig {
	c.JWTRole = &role
	return c
}

func (c *VaultProviderConfig) WithJWT(jwt string) *VaultProviderConfig {
	c.JWT = &jwt
	return c
}

func (c *VaultProviderConfig) WithJWTMount(mount string) *VaultProviderConfig {
	c.JWTMount = &mount
	return c
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

	jwt := ""
	if config.JWT != nil {
		jwt = *config.JWT
	}
	if jwt == "" {
		jwt = os.Getenv("VAULT_JWT")
	}

	token := ""
	if config.Token != nil {
		token = *config.Token
	}
	if token == "" {
		token = os.Getenv("VAULT_TOKEN")
	}

	providerArgs := &vault.ProviderArgs{
		Address:        pulumi.String(address),
		SkipChildToken: pulumi.Bool(true),
	}

	if config.JWTRole != nil && jwt != "" {
		jwtArgs := &vault.ProviderAuthLoginJwtArgs{
			Jwt:  pulumi.String(jwt),
			Role: pulumi.String(*config.JWTRole),
		}
		if config.JWTMount != nil {
			jwtArgs.Mount = pulumi.StringPtr(*config.JWTMount)
		}
		providerArgs.AuthLoginJwt = jwtArgs
		// pulumi-vault SDK v6.7.4 requires Token != nil unconditionally (returns
		// an error if nil), even when JWT auth is configured via AuthLoginJwt.
		// Passing an empty string satisfies the nil check. Verify when upgrading
		// pulumi-vault that this is still needed and that an empty token does not
		// interfere with JWT auth at the Vault API level.
		providerArgs.Token = pulumi.String("")
	} else if token != "" {
		providerArgs.Token = pulumi.String(token)
	} else {
		return nil, fmt.Errorf("vault auth not configured for %s: set VAULT_JWT (and configure a JWT role) or VAULT_TOKEN", orgName)
	}

	resourceName := fmt.Sprintf("vault-provider.%s", helpers.Slugify(orgName))
	return vault.NewProvider(ctx, resourceName, providerArgs)
}
