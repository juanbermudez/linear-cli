package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	// LinearTokenEndpoint is the OAuth token endpoint
	LinearTokenEndpoint = "https://api.linear.app/oauth/token"

	// LinearAPIEndpoint is the GraphQL API endpoint
	LinearAPIEndpoint = "https://api.linear.app/graphql"

	// DefaultClientID is the public client ID for Linear Agent CLI
	DefaultClientID = "984973f7762db2dc5dd3c939e3f5139c"

	// ServiceName is the keyring service name
	ServiceName = "agent-linear-cli"

	// TokenExpiryBuffer is how early to refresh before actual expiry
	TokenExpiryBuffer = 5 * time.Minute
)

// AuthMethod represents the authentication method in use
type AuthMethod string

const (
	AuthMethodNone              AuthMethod = "none"
	AuthMethodAPIKey            AuthMethod = "api_key"
	AuthMethodClientCredentials AuthMethod = "client_credentials"
)

// TokenInfo contains OAuth token information
type TokenInfo struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int       `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
	Scope       string    `json:"scope,omitempty"`
}

// AuthStatus represents the current authentication status
type AuthStatus struct {
	Authenticated bool       `json:"authenticated"`
	Method        AuthMethod `json:"method"`
	Source        string     `json:"source"` // "env", "keychain", "config"
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	User          *UserInfo  `json:"user,omitempty"`
}

// UserInfo contains authenticated user information
type UserInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Email       string `json:"email"`
	Admin       bool   `json:"admin"`
}

// Manager handles authentication operations
type Manager struct {
	storage Storage
}

// NewManager creates a new auth manager
func NewManager() *Manager {
	return &Manager{
		storage: NewKeyringStorage(),
	}
}

// GetToken returns the current access token using priority order:
// 1. Environment variables (LINEAR_API_KEY or LINEAR_CLIENT_ID+LINEAR_CLIENT_SECRET)
// 2. Keychain storage
// 3. Config file (legacy)
func (m *Manager) GetToken(ctx context.Context) (string, AuthMethod, error) {
	// Priority 1: Personal API key from environment
	if apiKey := os.Getenv("LINEAR_API_KEY"); apiKey != "" {
		return apiKey, AuthMethodAPIKey, nil
	}

	// Priority 2: Client credentials from environment
	clientID := os.Getenv("LINEAR_CLIENT_ID")
	clientSecret := os.Getenv("LINEAR_CLIENT_SECRET")
	if clientID != "" && clientSecret != "" {
		token, err := m.fetchClientCredentialsToken(ctx, clientID, clientSecret)
		if err != nil {
			return "", AuthMethodNone, fmt.Errorf("client credentials auth failed: %w", err)
		}
		return token, AuthMethodClientCredentials, nil
	}

	// Priority 3: Stored API key in keychain
	if apiKey, err := m.storage.GetAPIKey(); err == nil && apiKey != "" {
		return apiKey, AuthMethodAPIKey, nil
	}

	// Priority 4: Stored OAuth token in keychain
	if tokenInfo, err := m.storage.GetTokenInfo(); err == nil && tokenInfo != nil {
		// Check if token needs refresh
		if time.Now().Add(TokenExpiryBuffer).Before(tokenInfo.ExpiresAt) {
			return tokenInfo.AccessToken, AuthMethodClientCredentials, nil
		}
		// Token expired, try to refresh using stored credentials
		if clientSecret, err := m.storage.GetClientSecret(); err == nil && clientSecret != "" {
			storedClientID, _ := m.storage.GetClientID()
			if storedClientID == "" {
				storedClientID = DefaultClientID
			}
			token, err := m.fetchClientCredentialsToken(ctx, storedClientID, clientSecret)
			if err != nil {
				return "", AuthMethodNone, fmt.Errorf("token refresh failed: %w", err)
			}
			return token, AuthMethodClientCredentials, nil
		}
	}

	return "", AuthMethodNone, errors.New("not authenticated: run 'linear auth login' or set LINEAR_API_KEY (get key from https://linear.app/settings/api)")
}

// GetStatus returns the current authentication status
func (m *Manager) GetStatus(ctx context.Context) (*AuthStatus, error) {
	status := &AuthStatus{
		Authenticated: false,
		Method:        AuthMethodNone,
	}

	// Check environment variables first
	if apiKey := os.Getenv("LINEAR_API_KEY"); apiKey != "" {
		status.Authenticated = true
		status.Method = AuthMethodAPIKey
		status.Source = "env:LINEAR_API_KEY"
		return status, nil
	}

	clientID := os.Getenv("LINEAR_CLIENT_ID")
	clientSecret := os.Getenv("LINEAR_CLIENT_SECRET")
	if clientID != "" && clientSecret != "" {
		status.Authenticated = true
		status.Method = AuthMethodClientCredentials
		status.Source = "env:LINEAR_CLIENT_ID"
		return status, nil
	}

	// Check keychain
	if apiKey, err := m.storage.GetAPIKey(); err == nil && apiKey != "" {
		status.Authenticated = true
		status.Method = AuthMethodAPIKey
		status.Source = "keychain"
		return status, nil
	}

	if tokenInfo, err := m.storage.GetTokenInfo(); err == nil && tokenInfo != nil {
		status.Authenticated = true
		status.Method = AuthMethodClientCredentials
		status.Source = "keychain"
		status.ExpiresAt = &tokenInfo.ExpiresAt
		return status, nil
	}

	return status, nil
}

// LoginWithAPIKey stores an API key
func (m *Manager) LoginWithAPIKey(apiKey string) error {
	// Validate the API key format
	if !strings.HasPrefix(apiKey, "lin_api_") {
		return errors.New("invalid API key format: should start with 'lin_api_'")
	}

	return m.storage.SetAPIKey(apiKey)
}

// LoginWithClientCredentials stores client credentials and fetches initial token
func (m *Manager) LoginWithClientCredentials(ctx context.Context, clientID, clientSecret string) error {
	// Validate by fetching a token
	_, err := m.fetchClientCredentialsToken(ctx, clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("invalid credentials: %w", err)
	}

	// Store credentials
	if err := m.storage.SetClientID(clientID); err != nil {
		return err
	}
	return m.storage.SetClientSecret(clientSecret)
}

// Logout removes all stored credentials
func (m *Manager) Logout() error {
	var errs []error

	if err := m.storage.DeleteAPIKey(); err != nil {
		errs = append(errs, err)
	}
	if err := m.storage.DeleteTokenInfo(); err != nil {
		errs = append(errs, err)
	}
	if err := m.storage.DeleteClientID(); err != nil {
		errs = append(errs, err)
	}
	if err := m.storage.DeleteClientSecret(); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("logout completed with errors: %v", errs)
	}
	return nil
}

// fetchClientCredentialsToken fetches a new token using client credentials grant
func (m *Manager) fetchClientCredentialsToken(ctx context.Context, clientID, clientSecret string) (string, error) {
	data := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", LinearTokenEndpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error            string `json:"error"`
			ErrorDescription string `json:"error_description"`
		}
		json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.ErrorDescription != "" {
			return "", fmt.Errorf("%s: %s", errResp.Error, errResp.ErrorDescription)
		}
		return "", fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenInfo
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	// Calculate expiry time
	tokenResp.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Store the token
	if err := m.storage.SetTokenInfo(&tokenResp); err != nil {
		// Non-fatal: log but continue
		fmt.Fprintf(os.Stderr, "warning: failed to cache token: %v\n", err)
	}

	return tokenResp.AccessToken, nil
}
