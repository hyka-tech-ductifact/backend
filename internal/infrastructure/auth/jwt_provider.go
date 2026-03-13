package auth

import (
	"errors"
	"time"

	"ductifact/internal/application/ports"
	"ductifact/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

// JWTProvider implements ports.TokenProvider using golang-jwt.
type JWTProvider struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTProvider creates a new JWTProvider.
// The secret key comes from the config (originally JWT_SECRET env var).
// Panics if the secret is empty — this is a programming error.
func NewJWTProvider(cfg config.JWT) *JWTProvider {
	if cfg.Secret == "" {
		panic("JWT secret cannot be empty")
	}

	return &JWTProvider{
		secretKey:     []byte(cfg.Secret),
		tokenDuration: 24 * time.Hour, // Token expires in 24 hours
	}
}

// jwtClaims extends jwt.RegisteredClaims with custom fields.
type jwtClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user.
func (p *JWTProvider) GenerateToken(userID uuid.UUID, email string) (string, error) {
	now := time.Now()

	claims := jwtClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),                              // "sub" claim = user ID
			IssuedAt:  jwt.NewNumericDate(now),                      // "iat" claim
			ExpiresAt: jwt.NewNumericDate(now.Add(p.tokenDuration)), // "exp" claim
			Issuer:    "ductifact",                                  // "iss" claim
		},
	}

	// Create token with HS256 signing method
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret key
	return token.SignedString(p.secretKey)
}

// ValidateToken parses and validates a JWT, returning the claims.
func (p *JWTProvider) ValidateToken(tokenString string) (*ports.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify the signing method is what we expect (prevents algorithm switching attacks)
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

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalidToken
	}

	return &ports.TokenClaims{
		UserID: userID,
		Email:  claims.Email,
	}, nil
}
