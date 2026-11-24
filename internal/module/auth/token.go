package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenCookieName  = "access_token"
	RefreshTokenCookieName = "refresh_token"

	accessCookiePath  = "/"
	refreshCookiePath = "/auth"
	minSigningKeySize = 32
)

var (
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrWrongTokenType     = errors.New("token has the wrong type")
	ErrInvalidTokenConfig = errors.New("invalid token configuration")
)

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type TokenConfig struct {
	AccessSecret  []byte
	RefreshSecret []byte
	Issuer        string
	Audience      string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	MaxSessionTTL time.Duration
	ClockSkew     time.Duration
}

type TokenManager struct {
	accessSecret  []byte
	refreshSecret []byte
	issuer        string
	audience      string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	maxSessionTTL time.Duration
	clockSkew     time.Duration
}

type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type TokenScope struct {
	IDToaNha string
	IDKhuVuc string
}

type TokenClaims struct {
	UserID           string
	RoleID           string
	IDToaNha         string
	IDKhuVuc         string
	Type             TokenType
	IssuedAt         time.Time
	ExpiresAt        time.Time
	SessionExpiresAt time.Time
}

type jwtClaims struct {
	UserID           string           `json:"uid"`
	RoleID           string           `json:"rid"`
	IDToaNha         string           `json:"building_id,omitempty"`
	IDKhuVuc         string           `json:"area_id,omitempty"`
	TokenType        TokenType        `json:"typ"`
	SessionExpiresAt *jwt.NumericDate `json:"sxp"`
	jwt.RegisteredClaims
}

func NewTokenManager(cfg TokenConfig) (*TokenManager, error) {
	if len(cfg.AccessSecret) < minSigningKeySize ||
		len(cfg.RefreshSecret) < minSigningKeySize ||
		strings.TrimSpace(cfg.Issuer) == "" ||
		strings.TrimSpace(cfg.Audience) == "" ||
		cfg.AccessTTL <= 0 ||
		cfg.RefreshTTL <= cfg.AccessTTL ||
		cfg.MaxSessionTTL < cfg.RefreshTTL ||
		cfg.ClockSkew < 0 {
		return nil, ErrInvalidTokenConfig
	}

	if string(cfg.AccessSecret) == string(cfg.RefreshSecret) {
		return nil, fmt.Errorf("%w: access and refresh signing keys must differ", ErrInvalidTokenConfig)
	}

	return &TokenManager{
		accessSecret:  append([]byte(nil), cfg.AccessSecret...),
		refreshSecret: append([]byte(nil), cfg.RefreshSecret...),
		issuer:        cfg.Issuer,
		audience:      cfg.Audience,
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
		maxSessionTTL: cfg.MaxSessionTTL,
		clockSkew:     cfg.ClockSkew,
	}, nil
}

func (m *TokenManager) IssuePair(userID, role string) (*TokenPair, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(role) == "" {
		return nil, errors.New("user ID and role ID must not be empty")
	}

	now := time.Now().UTC()
	sessionExpiry := now.Add(m.maxSessionTTL)
	return m.issuePair(userID, role, TokenScope{}, sessionExpiry, now)
}

func (m *TokenManager) IssuePairWithScope(
	userID, role string,
	scope TokenScope,
) (*TokenPair, error) {
	if strings.TrimSpace(userID) == "" || strings.TrimSpace(role) == "" {
		return nil, errors.New("user ID and role ID must not be empty")
	}

	now := time.Now().UTC()
	sessionExpiry := now.Add(m.maxSessionTTL)
	return m.issuePair(userID, role, scope, sessionExpiry, now)
}

func (m *TokenManager) issuePair(
	userID, roleID string,
	scope TokenScope,
	sessionExpiry, now time.Time,
) (*TokenPair, error) {
	accessExpiry := minTime(now.Add(m.accessTTL), sessionExpiry)
	access, _, err := m.issue(
		userID,
		roleID,
		scope,
		TokenTypeAccess,
		accessExpiry,
		sessionExpiry,
		m.accessSecret,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("issue access token: %w", err)
	}

	refreshExpiry := minTime(now.Add(m.refreshTTL), sessionExpiry)
	refresh, _, err := m.issue(
		userID,
		roleID,
		scope,
		TokenTypeRefresh,
		refreshExpiry,
		sessionExpiry,
		m.refreshSecret,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("issue refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:      access,
		RefreshToken:     refresh,
		AccessExpiresAt:  accessExpiry,
		RefreshExpiresAt: refreshExpiry,
	}, nil
}

func (m *TokenManager) Refresh(refreshToken, currentRoleID string) (*TokenPair, error) {
	claims, err := m.VerifyRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(currentRoleID) == "" {
		return nil, errors.New("current role ID must not be empty")
	}
	return m.issuePair(
		claims.UserID,
		currentRoleID,
		TokenScope{IDToaNha: claims.IDToaNha, IDKhuVuc: claims.IDKhuVuc},
		claims.SessionExpiresAt,
		time.Now().UTC(),
	)
}

func (m *TokenManager) VerifyAccessToken(raw string) (*TokenClaims, error) {
	return m.verify(raw, TokenTypeAccess, m.accessSecret)
}

func (m *TokenManager) VerifyAccessCookie(r *http.Request) (*TokenClaims, error) {
	cookie, err := r.Cookie(AccessTokenCookieName)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return m.VerifyAccessToken(cookie.Value)
}

func (m *TokenManager) VerifyRefreshToken(raw string) (*TokenClaims, error) {
	return m.verify(raw, TokenTypeRefresh, m.refreshSecret)
}

func (m *TokenManager) VerifyRefreshCookie(r *http.Request) (*TokenClaims, error) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return m.VerifyRefreshToken(cookie.Value)
}

func (m *TokenManager) SetAuthCookies(w http.ResponseWriter, pair *TokenPair) error {
	now := time.Now().UTC()
	if pair == nil ||
		pair.AccessToken == "" ||
		pair.RefreshToken == "" ||
		!pair.AccessExpiresAt.After(now) ||
		!pair.RefreshExpiresAt.After(now) {
		return errors.New("invalid token pair")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     AccessTokenCookieName,
		Value:    pair.AccessToken,
		Path:     accessCookiePath,
		Expires:  pair.AccessExpiresAt,
		MaxAge:   cookieMaxAge(pair.AccessExpiresAt, now),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    pair.RefreshToken,
		Path:     refreshCookiePath,
		Expires:  pair.RefreshExpiresAt,
		MaxAge:   cookieMaxAge(pair.RefreshExpiresAt, now),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func ClearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, expiredCookie(AccessTokenCookieName, accessCookiePath))
	http.SetCookie(w, expiredCookie(RefreshTokenCookieName, refreshCookiePath))
}

func (m *TokenManager) issue(
	userID, roleID string,
	scope TokenScope,
	tokenType TokenType,
	expiresAt time.Time,
	sessionExpiry time.Time,
	key []byte,
	now time.Time,
) (string, time.Time, error) {
	jti, err := newTokenID()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate token ID: %w", err)
	}

	claims := jwtClaims{
		UserID:           userID,
		RoleID:           roleID,
		IDToaNha:         strings.TrimSpace(scope.IDToaNha),
		IDKhuVuc:         strings.TrimSpace(scope.IDKhuVuc),
		TokenType:        tokenType,
		SessionExpiresAt: jwt.NewNumericDate(sessionExpiry),
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			Audience:  jwt.ClaimStrings{m.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signed, expiresAt, nil
}

func (m *TokenManager) verify(raw string, expectedType TokenType, key []byte) (*TokenClaims, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrInvalidToken
	}

	claims := &jwtClaims{}
	parsed, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return key, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithLeeway(m.clockSkew),
	)
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, ErrWrongTokenType
	}
	if strings.TrimSpace(claims.UserID) == "" ||
		strings.TrimSpace(claims.RoleID) == "" ||
		claims.Subject != claims.UserID ||
		claims.ID == "" ||
		claims.IssuedAt == nil ||
		claims.ExpiresAt == nil ||
		claims.SessionExpiresAt == nil ||
		claims.ExpiresAt.After(claims.SessionExpiresAt.Time) ||
		!claims.SessionExpiresAt.Time.After(time.Now().Add(-m.clockSkew)) {
		return nil, ErrInvalidToken
	}

	return &TokenClaims{
		UserID:           claims.UserID,
		RoleID:           claims.RoleID,
		IDToaNha:         claims.IDToaNha,
		IDKhuVuc:         claims.IDKhuVuc,
		Type:             claims.TokenType,
		IssuedAt:         claims.IssuedAt.Time,
		ExpiresAt:        claims.ExpiresAt.Time,
		SessionExpiresAt: claims.SessionExpiresAt.Time,
	}, nil
}

func newTokenID() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func cookieMaxAge(expiresAt, now time.Time) int {
	seconds := int(expiresAt.Sub(now).Seconds())
	if seconds < 1 {
		return 1
	}
	return seconds
}

func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func expiredCookie(name, path string) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		Expires:  time.Unix(1, 0).UTC(),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}
