package controller

import (
	"context"
	"fmt"
	"time"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	authv1 "github.com/yaninyzwitty/chat/gen/auth/v1"
	myJwt "github.com/yaninyzwitty/chat/packages/auth/jwt"
	database "github.com/yaninyzwitty/chat/packages/db"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/monitoring"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuthController struct {
	authv1.UnimplementedAuthServiceServer
	Db                *gocql.Session
	M                 *monitoring.Metrics
	Config            *config.Config
	RefreshTokenStore *myJwt.RefreshTokenStore
}

func NewAuthController(ctx context.Context, cfg *config.Config, reg *prometheus.Registry, token string, rts *myJwt.RefreshTokenStore) *AuthController {
	m := monitoring.NewMetrics(reg)
	c := &AuthController{
		Config:            cfg,
		M:                 m,
		RefreshTokenStore: rts,
	}

	c.Db = database.ConnectAstra(cfg, token)
	return c
}

// --- LOGIN ---
func (c *AuthController) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	start := time.Now()
	const op = "login"

	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	var userID gocql.UUID
	var username, hashedPassword string

	query := "SELECT id, name, password FROM chat.users WHERE email = ? LIMIT 1"
	if err := c.Db.Query(query, req.Email).Consistency(gocql.One).Scan(&userID, &username, &hashedPassword); err != nil {
		c.observeError(op, "cassandra")
		return nil, status.Errorf(codes.Internal, "invalid credentials %v", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(req.Password)); err != nil {
		c.observeError(op, "bcrypt")
		return nil, status.Errorf(codes.Internal, "failed to compare hash and password %v", err)
	}

	tokens, err := myJwt.GenerateJWTPair(userID.String(), username, req.Email, []string{"user"})
	if err != nil {
		c.observeError(op, "jwt")
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	refreshToken, err := c.RefreshTokenStore.CreateOrGetRefreshToken(ctx, userID.String())
	if err != nil {
		c.observeError(op, "redis")
		return nil, status.Errorf(codes.Unauthenticated, "failed to create or get refresh token %v", err)
	}
	tokens.RefreshToken = refreshToken

	c.observeDuration(op, "cassandra", start)
	return &authv1.LoginResponse{Tokens: tokens}, nil
}

// --- REFRESH TOKEN ---
func (c *AuthController) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	start := time.Now()
	const op = "refresh_token"

	if req.RefreshToken == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token and user id are required")
	}

	eg, egCtx := errgroup.WithContext(ctx)
	var username, email string

	eg.Go(func() error {
		valid, err := c.RefreshTokenStore.ValidateRefreshToken(egCtx, req.UserId, req.RefreshToken)
		if err != nil {
			return fmt.Errorf("failed to validate refresh token: %w", err)
		}
		if !valid {
			return fmt.Errorf("invalid refresh token")
		}
		return nil
	})

	eg.Go(func() error {
		query := "SELECT name, email FROM chat.users WHERE id = ? LIMIT 1"
		if err := c.Db.Query(query, req.UserId).
			Consistency(gocql.One).
			Scan(&username, &email); err != nil {
			return fmt.Errorf("invalid user: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		c.observeError(op, "cassandra")
		return nil, status.Error(codes.Unauthenticated, err.Error())
	}

	tokens, err := myJwt.GenerateJWTPair(req.UserId, username, email, []string{"user"})
	if err != nil {
		c.observeError(op, "jwt")
		return nil, status.Errorf(codes.Internal, "failed to generate access token: %v", err)
	}

	refreshToken, err := c.RefreshTokenStore.CreateOrGetRefreshToken(ctx, req.UserId)
	if err != nil {
		c.observeError(op, "redis")
		return nil, status.Errorf(codes.Unauthenticated, "failed to create or get refresh token %v", err)
	}
	tokens.RefreshToken = refreshToken

	c.observeDuration(op, "cassandra", start)
	return &authv1.RefreshTokenResponse{Tokens: tokens}, nil
}

// --- VALIDATE TOKEN ---
func (c *AuthController) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	start := time.Now()
	const op = "validate_token"

	claims, err := myJwt.ValidateJWT(req.AccessToken)
	if err != nil {
		c.observeError(op, "jwt")
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	c.observeDuration(op, "jwt", start)
	return &authv1.ValidateTokenResponse{
		Valid: true,
		Claims: &authv1.Claims{
			UserId:    claims.UserID,
			Username:  claims.Username,
			Roles:     claims.Roles,
			IssuedAt:  timestamppb.New(claims.IssuedAt.Time),
			ExpiresAt: timestamppb.New(claims.ExpiresAt.Time),
		},
	}, nil
}

// --- LOGOUT ---
func (c *AuthController) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	start := time.Now()
	const op = "logout"

	if req.RefreshToken == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token and user id required")
	}

	if err := c.RefreshTokenStore.DeleteRefreshToken(ctx, req.UserId); err != nil {
		c.observeError(op, "redis")
		return &authv1.LogoutResponse{Success: false}, fmt.Errorf("failed to delete refresh token: %w", err)
	}

	c.observeDuration(op, "redis", start)
	return &authv1.LogoutResponse{Success: true}, nil
}

// --- helpers for metrics ---
func (c *AuthController) observeDuration(op, db string, start time.Time) {
	c.M.Duration.WithLabelValues(op, db).Observe(time.Since(start).Seconds())
}

func (c *AuthController) observeError(op, db string) {
	c.M.Errors.WithLabelValues(op, db).Inc()
}
