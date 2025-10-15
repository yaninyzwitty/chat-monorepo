package jwt

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	authv1 "github.com/yaninyzwitty/chat/gen/auth/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// JwtKey is the HMAC secret used to sign tokens
var JwtKey = []byte(os.Getenv("JWT_AUTH_SECRET"))

// Claims represents JWT claims for a user
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// GenerateJWTPair generates a new access token and refresh token
func GenerateJWTPair(userID, username, email string, roles []string) (*authv1.TokenPair, error) {
	now := time.Now()
	exp := now.Add(60 * time.Minute)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "chat",
			Subject:   "user-token",
			Audience:  jwt.ClaimStrings{"chat"},
			ID:        uuid.New().String(),
		},
	}

	// generate access token
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(JwtKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// generate refresh token (longer expiration)

	// refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(JwtKey)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to sign refresh token: %w", err)
	// }

	return &authv1.TokenPair{
		AccessToken: accessToken,
		ExpiresAt:   timestamppb.New(exp),
	}, nil
}

// ValidateJWT parses and validates a JWT access token and returns Claims
func ValidateJWT(tokenStr string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return JwtKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
