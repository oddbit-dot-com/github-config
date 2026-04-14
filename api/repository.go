package api

import (
	"github.com/pulumi/pulumi-github/sdk/v6/go/github"
)

// RepositoryConfig defines configuration for a single repository
type RepositoryConfig struct {
	// Standard Pulumi GitHub repository arguments
	*github.RepositoryArgs

	// Branch protection rules (pattern -> protection args)
	// If nil, organization defaults apply
	BranchProtectionRules BranchProtectionRules
}

// BranchProtectionRules maps branch patterns to protection configurations
type BranchProtectionRules map[string]*github.BranchProtectionArgs

// Repositories maps repository names to their configurations
type Repositories map[string]*RepositoryConfig
