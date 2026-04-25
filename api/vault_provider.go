package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/oddbit-dot-com/github-config/helpers"
	"github.com/pulumi/pulumi-vault/sdk/v6/go/vault"
	"github.com/pulumi/pulumi-vault/sdk/v6/go/vault/kv"
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

type Encoding string

const (
	EncodingNone   Encoding = ""
	EncodingBase64 Encoding = "base64"
)

// SecretRef resolves a secret value from an external provider.
type SecretRef interface {
	Resolve(ctx *pulumi.Context) (pulumi.StringPtrInput, error)
}

type VaultSecretRef struct {
	Path     string
	Key      string
	Encoding Encoding

	provider   *vault.Provider
	mountPoint string
}

func (r VaultSecretRef) applyEncoding(v string) (string, error) {
	switch r.Encoding {
	case EncodingNone:
		return v, nil
	case EncodingBase64:
		return base64.StdEncoding.EncodeToString([]byte(v)), nil
	default:
		return "", fmt.Errorf("unsupported encoding %q for vault secret at %s", r.Encoding, r.Path)
	}
}

func (r *VaultSecretRef) Resolve(ctx *pulumi.Context) (pulumi.StringPtrInput, error) {
	if r.provider == nil {
		return nil, fmt.Errorf("vault provider not configured for secret at %s", r.Path)
	}
	result, err := kv.LookupSecretV2(ctx, &kv.LookupSecretV2Args{
		Mount: r.mountPoint,
		Name:  r.Path,
	}, pulumi.Provider(r.provider))
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(result.DataJson), &data); err != nil {
		return nil, fmt.Errorf("invalid JSON from vault at %s: %w", r.Path, err)
	}
	v, ok := data[r.Key].(string)
	if !ok {
		return nil, fmt.Errorf("key %q not found or not a string in vault secret at %s", r.Key, r.Path)
	}
	v, err = r.applyEncoding(v)
	if err != nil {
		return nil, err
	}
	return pulumi.String(v).ToStringPtrOutput(), nil
}

func provisionSecrets(
	ctx *pulumi.Context,
	secrets ActionsSecrets,
	create func(secretName string, value pulumi.StringPtrInput) error,
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

type OrgSecretRef struct {
	SecretRef
	Visibility string
}

type ActionsSecrets map[string]SecretRef
type OrgActionsSecrets map[string]OrgSecretRef

func bindVaultSecrets(secrets ActionsSecrets, provider *vault.Provider, mountPoint string) {
	for _, ref := range secrets {
		if v, ok := ref.(*VaultSecretRef); ok {
			v.provider = provider
			v.mountPoint = mountPoint
		}
	}
}

func bindOrgVaultSecrets(secrets OrgActionsSecrets, provider *vault.Provider, mountPoint string) {
	for _, ref := range secrets {
		if v, ok := ref.SecretRef.(*VaultSecretRef); ok {
			v.provider = provider
			v.mountPoint = mountPoint
		}
	}
}

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

func CreateVaultProvider(ctx *pulumi.Context, config *VaultProviderConfig, ownerName string) (*vault.Provider, error) {
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
		return nil, fmt.Errorf("vault address not configured for %s: set VAULT_ADDR or pass an explicit address", ownerName)
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
		return nil, fmt.Errorf("vault auth not configured for %s: set VAULT_JWT (and configure a JWT role) or VAULT_TOKEN", ownerName)
	}

	resourceName := helpers.ResourceName("vault-provider", ownerName)
	return vault.NewProvider(ctx, resourceName, providerArgs)
}
