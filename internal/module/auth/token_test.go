package auth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testTokenManager(t *testing.T) *TokenManager {
	t.Helper()

	manager, err := NewTokenManager(TokenConfig{
		AccessSecret:  []byte(strings.Repeat("a", minSigningKeySize)),
		RefreshSecret: []byte(strings.Repeat("b", minSigningKeySize)),
		Issuer:        "cloud-backend",
		Audience:      "cloud-backend-api",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    30 * 24 * time.Hour,
		MaxSessionTTL: 90 * 24 * time.Hour,
		ClockSkew:     time.Second,
	})
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	return manager
}

func TestIssuePairAndVerifyTokens(t *testing.T) {
	manager := testTokenManager(t)

	pair, err := manager.IssuePair("user-42", "role-7")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	access, err := manager.VerifyAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken() error = %v", err)
	}
	if access.UserID != "user-42" || access.RoleID != "role-7" || access.Type != TokenTypeAccess {
		t.Fatalf("unexpected access claims: %+v", access)
	}

	refresh, err := manager.VerifyRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Fatalf("VerifyRefreshToken() error = %v", err)
	}
	if refresh.UserID != "user-42" || refresh.RoleID != "role-7" || refresh.Type != TokenTypeRefresh {
		t.Fatalf("unexpected refresh claims: %+v", refresh)
	}

	if _, err := manager.VerifyRefreshToken(pair.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("VerifyRefreshToken(access token) error = %v, want invalid token", err)
	}
	if _, err := manager.VerifyAccessToken(pair.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("VerifyAccessToken(refresh token) error = %v, want invalid token", err)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	manager := testTokenManager(t)
	pair, err := manager.IssuePair("user-42", "role-7")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	parts := strings.Split(pair.AccessToken, ".")
	if len(parts) != 3 || len(parts[2]) == 0 {
		t.Fatalf("issued token has invalid JWT format: %q", pair.AccessToken)
	}
	if parts[2][0] == 'A' {
		parts[2] = "B" + parts[2][1:]
	} else {
		parts[2] = "A" + parts[2][1:]
	}
	tampered := strings.Join(parts, ".")
	if _, err := manager.VerifyAccessToken(tampered); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("VerifyAccessToken(tampered) error = %v, want ErrInvalidToken", err)
	}
}

func TestRefreshIssuesNewPair(t *testing.T) {
	manager := testTokenManager(t)
	pair, err := manager.IssuePair("user-42", "role-7")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	refreshed, err := manager.Refresh(pair.RefreshToken, "role-8")
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if refreshed.RefreshToken == pair.RefreshToken {
		t.Fatal("Refresh() reused the original refresh token")
	}
	claims, err := manager.VerifyAccessToken(refreshed.AccessToken)
	if err != nil {
		t.Fatalf("VerifyAccessToken(refreshed token) error = %v", err)
	}
	if claims.RoleID != "role-8" {
		t.Fatalf("refreshed token role = %q, want current role", claims.RoleID)
	}
	if !claims.SessionExpiresAt.Equal(
		mustVerifyRefresh(t, manager, pair.RefreshToken).SessionExpiresAt,
	) {
		t.Fatal("refresh extended the absolute session expiration")
	}
	if _, err := manager.VerifyRefreshToken(refreshed.RefreshToken); err != nil {
		t.Fatalf("VerifyRefreshToken(refreshed token) error = %v", err)
	}
}

func TestSetAndClearAuthCookies(t *testing.T) {
	manager := testTokenManager(t)
	pair, err := manager.IssuePair("user-42", "role-7")
	if err != nil {
		t.Fatalf("IssuePair() error = %v", err)
	}

	recorder := httptest.NewRecorder()
	if err := manager.SetAuthCookies(recorder, pair); err != nil {
		t.Fatalf("SetAuthCookies() error = %v", err)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("SetAuthCookies() set %d cookies, want 2", len(cookies))
	}
	for _, cookie := range cookies {
		if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
			t.Errorf("cookie security attributes are incorrect: %+v", cookie)
		}
	}
	request := httptest.NewRequest(http.MethodGet, "/auth/refresh", nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	if _, err := manager.VerifyAccessCookie(request); err != nil {
		t.Errorf("VerifyAccessCookie() error = %v", err)
	}
	if _, err := manager.VerifyRefreshCookie(request); err != nil {
		t.Errorf("VerifyRefreshCookie() error = %v", err)
	}

	clearRecorder := httptest.NewRecorder()
	ClearAuthCookies(clearRecorder)
	cleared := clearRecorder.Result().Cookies()
	if len(cleared) != 2 {
		t.Fatalf("ClearAuthCookies() set %d cookies, want 2", len(cleared))
	}
	for _, cookie := range cleared {
		if cookie.MaxAge != -1 || cookie.Value != "" {
			t.Errorf("cookie was not expired: %+v", cookie)
		}
	}
}

func TestNewTokenManagerRejectsInvalidConfiguration(t *testing.T) {
	_, err := NewTokenManager(TokenConfig{
		AccessSecret:  []byte(strings.Repeat("a", minSigningKeySize)),
		RefreshSecret: []byte(strings.Repeat("a", minSigningKeySize)),
		Issuer:        "issuer",
		Audience:      "audience",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    time.Hour,
		MaxSessionTTL: 24 * time.Hour,
	})
	if !errors.Is(err, ErrInvalidTokenConfig) {
		t.Fatalf("NewTokenManager() error = %v, want ErrInvalidTokenConfig", err)
	}
}

func mustVerifyRefresh(t *testing.T, manager *TokenManager, raw string) *TokenClaims {
	t.Helper()
	claims, err := manager.VerifyRefreshToken(raw)
	if err != nil {
		t.Fatalf("VerifyRefreshToken() error = %v", err)
	}
	return claims
}
