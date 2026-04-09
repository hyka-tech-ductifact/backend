package auth

import (
	"errors"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Token type constants used to differentiate access and refresh tokens.
const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrInvalidRefreshToken = errors.New("invalid or expired refresh token")
	ErrWrongTokenType      = errors.New("wrong token type")
)

// JWTProvider implements ports.TokenProvider using golang-jwt.
type JWTProvider struct {
	secretKey            []byte
	tokenDuration        time.Duration
	refreshTokenDuration time.Duration
}

// NewJWTProvider creates a new JWTProvider.
// The secret key comes from the config (originally JWT_SECRET env var).
// Panics if the secret is empty — this is a programming error.
func NewJWTProvider(cfg config.JWT) *JWTProvider {
	if cfg.Secret == "" {
		panic("JWT secret cannot be empty")
	}

	return &JWTProvider{
		secretKey:            []byte(cfg.Secret),
		tokenDuration:        cfg.TokenDuration,
		refreshTokenDuration: cfg.RefreshTokenDuration,
	}
}

// jwtClaims extends jwt.RegisteredClaims with custom fields.
type jwtClaims struct {
	Email     string `json:"email"`
	TokenType string `json:"type"`
	jwt.RegisteredClaims
}

// GenerateTokenPair creates a signed access token and refresh token for the given user.
// The access token is short-lived; the refresh token is long-lived.
// Both contain a "type" claim to prevent misuse (e.g. using a refresh token as access).
func (p *JWTProvider) GenerateTokenPair(userID uuid.UUID, email string) (*ports.TokenPair, error) {
	accessToken, err := p.generateToken(userID, email, tokenTypeAccess, p.tokenDuration)
	if err != nil {
		return nil, err
	}

	refreshToken, err := p.generateToken(userID, email, tokenTypeRefresh, p.refreshTokenDuration)
	if err != nil {
		return nil, err
	}

	return &ports.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateToken parses and validates an access token, returning the claims.
// It rejects refresh tokens — use ValidateRefreshToken for those.
func (p *JWTProvider) ValidateToken(tokenString string) (*ports.TokenClaims, error) {
	claims, err := p.parseToken(tokenString)
	if err != nil {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != tokenTypeAccess {
		return nil, ErrInvalidToken
	}

	return p.toClaims(claims)
}

// ValidateRefreshToken parses and validates a refresh token, returning the claims.
// It rejects access tokens — use ValidateToken for those.
func (p *JWTProvider) ValidateRefreshToken(tokenString string) (*ports.TokenClaims, error) {
	claims, err := p.parseToken(tokenString)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	if claims.TokenType != tokenTypeRefresh {
		return nil, ErrInvalidRefreshToken
	}

	return p.toClaims(claims)
}

// --- private helpers ---

// generateToken creates a signed JWT with the given type and duration.
func (p *JWTProvider) generateToken(userID uuid.UUID, email, tokenType string, duration time.Duration) (string, error) {
	now := time.Now()

	claims := jwtClaims{
		Email:     email,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			Issuer:    "ductifact",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(p.secretKey)
}

// parseToken parses a JWT string and returns the custom claims.
func (p *JWTProvider) parseToken(tokenString string) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return p.secretKey, nil
	})

	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// toClaims converts internal JWT claims to the port's TokenClaims.
func (p *JWTProvider) toClaims(claims *jwtClaims) (*ports.TokenClaims, error) {
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &ports.TokenClaims{
		UserID: userID,
		Email:  claims.Email,
	}, nil
}
