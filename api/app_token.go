package api

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type appCredentials struct {
	clientID   string
	privateKey *rsa.PrivateKey
}

// installationTokenFunc is the function used to generate installation tokens.
// It can be replaced in tests.
var installationTokenFunc = generateInstallationToken

func loadAppCredentials() (*appCredentials, bool) {
	clientID := os.Getenv("GITHUB_APP_CLIENT_ID")
	privateKeyPEM := os.Getenv("GITHUB_APP_PRIVATE_KEY")
	if clientID == "" || privateKeyPEM == "" {
		return nil, false
	}
	key, err := parseRSAPrivateKey([]byte(privateKeyPEM))
	if err != nil {
		return nil, false
	}
	return &appCredentials{clientID: clientID, privateKey: key}, true
}

func parseRSAPrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return key, nil
}

func signAppJWT(creds *appCredentials) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		// Back-date by 60 seconds to account for clock skew
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)),
		ExpiresAt: jwt.NewNumericDate(now.Add(9 * time.Minute)),
		Issuer:    creds.clientID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(creds.privateKey)
}

func generateInstallationToken(creds *appCredentials, owner, apiKind string) (string, error) {
	appJWT, err := signAppJWT(creds)
	if err != nil {
		return "", fmt.Errorf("failed to sign app JWT: %w", err)
	}

	client := &http.Client{}

	installationID, err := getInstallationID(client, appJWT, owner, apiKind)
	if err != nil {
		return "", err
	}

	return getAccessToken(client, appJWT, installationID, owner)
}

func getInstallationID(client *http.Client, appJWT, owner, apiKind string) (int64, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet,
		fmt.Sprintf("https://api.github.com/%s/%s/installation", apiKind, owner), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build installation request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to find installation for %s: %w", owner, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to find installation for %s: HTTP %d", owner, resp.StatusCode)
	}

	var installation struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&installation); err != nil {
		return 0, fmt.Errorf("failed to parse installation response for %s: %w", owner, err)
	}
	return installation.ID, nil
}

func getAccessToken(client *http.Client, appJWT string, installationID int64, owner string) (string, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID), nil)
	if err != nil {
		return "", fmt.Errorf("failed to build access token request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+appJWT)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get installation token for %s: %w", owner, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("failed to get installation token for %s: HTTP %d", owner, resp.StatusCode)
	}

	var tokenResponse struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to parse token response for %s: %w", owner, err)
	}
	return tokenResponse.Token, nil
}
