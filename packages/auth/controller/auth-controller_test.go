package controller

import (
	"context"
	"net"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	authv1 "github.com/yaninyzwitty/chat/gen/auth/v1"
	authJwt "github.com/yaninyzwitty/chat/packages/auth/jwt"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func dialer(listener *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(ctx context.Context, s string) (net.Conn, error) {
		return listener.Dial()
	}
}

func TestLogin(t *testing.T) {
	t.Parallel()

	// Setup interceptor (your JWT middleware)
	interceptor := authJwt.AuthInterceptor()

	// Create in-memory listener
	listener := bufconn.Listen(bufSize)
	server := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor),
	)
	// Build dependencies for AuthController
	cfg := &config.Config{} // mock or load a test config
	reg := prometheus.NewRegistry()
	redisClient := redis.NewClient(&redis.Options{}) // mock Redis client for testing
	refreshTokenStore := authJwt.NewRefreshTokenStore(redisClient)
	token := "test-token" // fake token just for DbConnect init

	authService := NewAuthController(context.Background(), cfg, reg, token, refreshTokenStore)
	authv1.RegisterAuthServiceServer(server, authService)

	// Start gRPC server in a goroutine
	go func() {
		if err := server.Serve(listener); err != nil {
			// Can't call t.Fatalf from goroutine, just log the error
			t.Logf("server exited with error: %v", err)
		}
	}()

	// Create client connection
	ctx := context.Background()
	conn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(dialer(listener)),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	require.NoError(t, err)
	if err := conn.Close(); err != nil {
		t.Logf("error closing connection: %v", err)
	}

	client := authv1.NewAuthServiceClient(conn)

	// ---- Actual test case ----
	t.Run("missing email/password should fail", func(t *testing.T) {
		_, err := client.Login(ctx, &authv1.LoginRequest{
			Email:    "",
			Password: "",
		})
		require.Error(t, err)
	})

}
