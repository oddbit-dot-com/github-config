package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/resource"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func providerToken(r mockResource) string {
	v := r.inputs["token"]
	if v.IsNull() {
		return ""
	}
	if v.IsSecret() {
		v = v.SecretValue().Element
	}
	if v.IsString() {
		return v.StringValue()
	}
	return ""
}

func testPrivateKeyPEM(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate test RSA key: %v", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}))
}

// ensure resource import is used
var _ = resource.PropertyMap{}

func TestGitHubProviderExplicitTokenTakesPriority(t *testing.T) {
	t.Setenv("GITHUB_APP_CLIENT_ID", "Iv1.abc123")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", testPrivateKeyPEM(t))

	generatorCalled := false
	old := installationTokenFunc
	installationTokenFunc = func(_ *appCredentials, _ string, _ string) (string, error) {
		generatorCalled = true
		return "app-token", nil
	}
	defer func() { installationTokenFunc = old }()

	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Owner: Owner{
				Name:                 "testorg",
				GithubProviderConfig: NewGithubProviderConfig().WithToken("explicit-token"),
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}

	if generatorCalled {
		t.Error("installationTokenFunc should not be called when an explicit token is set")
	}

	providers := mocks.findByType("pulumi:providers:github")
	if len(providers) != 1 {
		t.Fatalf("expected 1 github provider, got %d", len(providers))
	}
	if got := providerToken(providers[0]); got != "explicit-token" {
		t.Errorf("expected token=explicit-token, got %q", got)
	}
}

func TestGitHubProviderUsesAppCredentials(t *testing.T) {
	t.Setenv("GITHUB_APP_CLIENT_ID", "Iv1.abc123")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", testPrivateKeyPEM(t))

	old := installationTokenFunc
	installationTokenFunc = func(_ *appCredentials, owner string, _ string) (string, error) {
		return "app-token-for-" + owner, nil
	}
	defer func() { installationTokenFunc = old }()

	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Owner: Owner{
				Name:                 "testorg",
				GithubProviderConfig: NewGithubProviderConfig(),
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}

	providers := mocks.findByType("pulumi:providers:github")
	if len(providers) != 1 {
		t.Fatalf("expected 1 github provider, got %d", len(providers))
	}
	if got := providerToken(providers[0]); got != "app-token-for-testorg" {
		t.Errorf("expected app-generated token, got %q", got)
	}
}

func TestGitHubProviderUsesOrgsAPIForOrganization(t *testing.T) {
	t.Setenv("GITHUB_APP_CLIENT_ID", "Iv1.abc123")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", testPrivateKeyPEM(t))

	var capturedAPIKind string
	old := installationTokenFunc
	installationTokenFunc = func(_ *appCredentials, _ string, apiKind string) (string, error) {
		capturedAPIKind = apiKind
		return "app-token", nil
	}
	defer func() { installationTokenFunc = old }()

	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Owner: Owner{
				Name:                 "testorg",
				GithubProviderConfig: NewGithubProviderConfig(),
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}

	if capturedAPIKind != "orgs" {
		t.Errorf("expected apiKind=orgs for Organization, got %q", capturedAPIKind)
	}
}

func TestGitHubProviderUsesUsersAPIForUser(t *testing.T) {
	t.Setenv("GITHUB_APP_CLIENT_ID", "Iv1.abc123")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", testPrivateKeyPEM(t))

	var capturedAPIKind string
	old := installationTokenFunc
	installationTokenFunc = func(_ *appCredentials, _ string, apiKind string) (string, error) {
		capturedAPIKind = apiKind
		return "app-token", nil
	}
	defer func() { installationTokenFunc = old }()

	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		user := &User{
			Owner: Owner{
				Name:                 "testuser",
				GithubProviderConfig: NewGithubProviderConfig(),
			},
		}
		return user.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}

	if capturedAPIKind != "users" {
		t.Errorf("expected apiKind=users for User, got %q", capturedAPIKind)
	}
}

func TestGitHubProviderNoTokenWhenNoCredentials(t *testing.T) {
	t.Setenv("GITHUB_APP_CLIENT_ID", "")
	t.Setenv("GITHUB_APP_PRIVATE_KEY", "")

	old := installationTokenFunc
	installationTokenFunc = func(_ *appCredentials, _ string, _ string) (string, error) {
		panic("should not be called")
	}
	defer func() { installationTokenFunc = old }()

	mocks := &mockMonitor{}
	err := pulumi.RunErr(func(ctx *pulumi.Context) error {
		org := &Organization{
			Owner: Owner{
				Name:                 "testorg",
				GithubProviderConfig: NewGithubProviderConfig(),
			},
		}
		return org.Ensure(ctx)
	}, pulumi.WithMocks("proj", "stack", mocks))
	if err != nil {
		t.Fatal(err)
	}

	providers := mocks.findByType("pulumi:providers:github")
	if len(providers) != 1 {
		t.Fatalf("expected 1 github provider, got %d", len(providers))
	}
	if got := providerToken(providers[0]); got != "" {
		t.Errorf("expected no token when no credentials are set, got %q", got)
	}
}
