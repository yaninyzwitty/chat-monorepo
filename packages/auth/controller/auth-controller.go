package controller

import (
	"context"
	"fmt"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	authv1 "github.com/yaninyzwitty/chat/gen/auth/v1"
	myJwt "github.com/yaninyzwitty/chat/packages/auth/jwt"
	database "github.com/yaninyzwitty/chat/packages/db"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/monitoring"
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

	c.Db = database.DbConnect(ctx, cfg, token)
	return c
}

func (c *AuthController) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	var userID gocql.UUID
	var username string

	query := "SELECT id, username FROM users WHERE email = ? LIMIT 1"
	if err := c.Db.Query(query, req.Email).Consistency(gocql.One).Scan(&userID, &username); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	tokens, err := myJwt.GenerateJWTPair(userID.String(), username, []string{"user"})
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}
	return &authv1.LoginResponse{
		Tokens: tokens,
	}, nil
}

func (c *AuthController) RefreshToken(ctx context.Context, req *authv1.RefreshTokenRequest) (*authv1.RefreshTokenResponse, error) {
	if req.RefreshToken == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token and user id are required")
	}

	// TODO-- USE ERR GROUP FOR THESE TWO
	valid, err := c.RefreshTokenStore.ValidateRefreshToken(ctx, req.UserId, req.RefreshToken)
	if err != nil || !valid {
		return nil, fmt.Errorf("invalid refresh token")
	}
	var username string
	query := "SELECT  username FROM users WHERE id = ? LIMIT 1"
	if err := c.Db.Query(query, req.UserId).Consistency(gocql.One).Scan(&username); err != nil {
		return nil, fmt.Errorf("invalid user")
	}

	// Generate new access token
	tokens, err := myJwt.GenerateJWTPair(req.UserId, username, []string{"user"})
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return &authv1.RefreshTokenResponse{
		Tokens: tokens,
	}, nil
}

func (c *AuthController) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	claims, err := myJwt.ValidateJWT(req.AccessToken)
	if err != nil {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

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

func (c *AuthController) Logout(ctx context.Context, req *authv1.LogoutRequest) (*authv1.LogoutResponse, error) {
	if req.RefreshToken == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token and user id required")

	}

	if err := c.RefreshTokenStore.DeleteRefreshToken(ctx, req.UserId); err != nil {
		return &authv1.LogoutResponse{Success: false}, fmt.Errorf("failed to delete refresh token: %w", err)
	}
	return &authv1.LogoutResponse{Success: true}, nil

}
