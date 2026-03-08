package auth

import (
	"errors"
	"os"
	"time"

	"ductifact/internal/application/ports"

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
// The secret key is read from the environment variable JWT_SECRET.
// It panics if JWT_SECRET is not set — this is intentional to prevent
// running in production with a weak or missing secret.
func NewJWTProvider() *JWTProvider {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is required but not set")
	}

	return &JWTProvider{
		secretKey:     []byte(secret),
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
