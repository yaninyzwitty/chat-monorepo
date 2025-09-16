package jwt

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// contextKey is used to store values in context
type contextKey string

const UserContextKey contextKey = "user"

// AuthInterceptor returns a gRPC unary interceptor for authentication
func AuthInterceptor() grpc.UnaryServerInterceptor {
	// define public routes (no auth required)
	publicRoutes := map[string]struct{}{
		"Login":        {},
		"RefreshToken": {},
		"Logout":       {},
		"CreateUser":   {},
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		// gRPC full method is /package.Service/Method
		parts := strings.Split(info.FullMethod, "/")
		if len(parts) != 3 {
			return nil, status.Error(codes.Unauthenticated, "invalid gRPC method")
		}
		methodName := parts[2]

		// skip auth for public routes
		if _, ok := publicRoutes[methodName]; ok {
			return handler(ctx, req)
		}

		// extract bearer token from metadata
		token, err := AuthFromMD(ctx, "bearer")
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "failed to extract authorization header: %v", err)
		}

		// validate JWT token
		claims, err := ValidateJWT(token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "failed to validate JWT token: %v", err)
		}

		// inject user info (claims) into context
		ctx = context.WithValue(ctx, UserContextKey, claims)

		// call the handler with the updated context
		return handler(ctx, req)
	}
}
