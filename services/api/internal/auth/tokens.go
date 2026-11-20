package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenIssuer      = "gsi-workorder"
	accessTokenType  = "access"
	refreshTokenType = "refresh"

	// AccessTokenTTL and RefreshTokenTTL are the M1 contract from the PRD.
	AccessTokenTTL  = 12 * time.Hour
	RefreshTokenTTL = 7 * 24 * time.Hour
)

var ErrInvalidToken = errors.New("auth: invalid token")

// Claims are the local JWT claims. Role is informational; authorization always
// reloads the current role from the database.
type Claims struct {
	Role      string `json:"role"`
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

// TokenPair is an access/refresh pair issued by the local application.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	ExpiresIn    int64
}

type TokenManager struct {
	secret []byte
	now    func() time.Time
}

func NewTokenManager(secret string) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		now:    time.Now,
	}
}

func newTokenManagerWithClock(secret string, now func() time.Time) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		now:    now,
	}
}

func (m *TokenManager) IssuePair(userID int64, role string) (TokenPair, error) {
	now := m.now().UTC()
	accessExpiresAt := now.Add(AccessTokenTTL)

	accessID, err := newTokenID()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate access token id: %w", err)
	}
	refreshID, err := newTokenID()
	if err != nil {
		return TokenPair{}, fmt.Errorf("generate refresh token id: %w", err)
	}

	accessClaims := Claims{
		Role:      role,
		TokenType: accessTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExpiresAt),
			ID:        accessID,
		},
	}
	refreshClaims := Claims{
		Role:      role,
		TokenType: refreshTokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(RefreshTokenTTL)),
			ID:        refreshID,
		},
	}

	accessToken, err := m.sign(accessClaims)
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, err := m.sign(refreshClaims)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExpiresAt,
		ExpiresIn:    int64(AccessTokenTTL / time.Second),
	}, nil
}

func (m *TokenManager) ParseAccess(token string) (*Claims, error) {
	return m.parse(token, accessTokenType)
}

func (m *TokenManager) ParseRefresh(token string) (*Claims, error) {
	return m.parse(token, refreshTokenType)
}

func (m *TokenManager) sign(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

func (m *TokenManager) parse(tokenString, expectedType string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, ErrInvalidToken
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(m.now),
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	if claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

func newTokenID() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
