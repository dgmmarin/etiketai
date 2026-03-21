// Package oauth verifies Google ID tokens via Google's tokeninfo endpoint.
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// GoogleClaims holds the fields we care about from the tokeninfo response.
type GoogleClaims struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Aud           string `json:"aud"`
}

var httpClient = &http.Client{Timeout: 5 * time.Second}

// VerifyGoogleIDToken validates an ID token via Google's tokeninfo endpoint
// and returns the verified claims. Returns an error if the token is invalid,
// expired, or the audience doesn't match clientID.
func VerifyGoogleIDToken(ctx context.Context, idToken, clientID string) (*GoogleClaims, error) {
	url := "https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tokeninfo request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google tokeninfo returned %d — token invalid or expired", resp.StatusCode)
	}

	var claims GoogleClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("decode tokeninfo: %w", err)
	}

	if claims.Email == "" || claims.Sub == "" {
		return nil, fmt.Errorf("tokeninfo missing email or sub")
	}

	// Verify the token was issued for our app.
	if clientID != "" && claims.Aud != clientID {
		return nil, fmt.Errorf("token audience %q does not match client ID", claims.Aud)
	}

	return &claims, nil
}
