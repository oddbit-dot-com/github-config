package api

import (
	"os"
)

// ProviderConfig holds GitHub provider configuration
type ProviderConfig struct {
	// GitHub token. If nil, uses default (GITHUB_TOKEN env var or gh CLI)
	Token *string

	// GitHub owner (organization or user). Used when creating providers.
	// In org-mode: if nil, falls back to organization name
	// In standalone mode: required to create a provider from ProviderConfig
	Owner *string
}

// ProviderFromEnv creates a provider config that reads from the specified
// environment variable, falling back to GITHUB_TOKEN or gh CLI if not set
func ProviderFromEnv(envVar string) *ProviderConfig {
	token := os.Getenv(envVar)
	if token == "" {
		// Fall back to default - return nil token to use Pulumi's default resolution
		return &ProviderConfig{Token: nil}
	}
	return &ProviderConfig{Token: &token}
}
