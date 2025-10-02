package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	authv1 "github.com/yaninyzwitty/chat/gen/auth/v1"
	"github.com/yaninyzwitty/chat/packages/auth/controller"
	"github.com/yaninyzwitty/chat/packages/auth/jwt"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/monitoring"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Context that cancels on interrupt/terminate signals
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()
	if err := run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("error running application",
			slog.String("error", err.Error()),
		)
	}
	slog.Info("server stopped cleanly")
}

func run(ctx context.Context) error {
	// Parse flags
	cp := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg := &config.Config{}
	if *cp != "" {
		if err := cfg.LoadConfig(*cp); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Set up logging (add custom handler if you want debug output)
	// if cfg.Debug { ... }

	addr := fmt.Sprintf(":%d", cfg.MetricsPort1)
	// Prometheus metrics
	reg := prometheus.NewRegistry()
	monitoring.StartPrometheusServer(reg, addr)

	// gRPC server setup
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(jwt.AuthInterceptor()),
	)
	reflection.Register(grpcServer)

	// start godotenv
	if err := godotenv.Load(); err != nil {
		slog.Warn("Failed to load .env")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		return errors.New("REDIS_URL environment variable is not set")
	}

	redisClient, err := generateRedisClient(redisURL)
	if err != nil {
		return fmt.Errorf("failed to create redis client: %w", err)
	}

	rts := jwt.NewRefreshTokenStore(redisClient)

	// Create controller with DB + metrics
	dbToken := os.Getenv("ASTRA_DB_TOKEN")
	if dbToken == "" {
		return errors.New("ASTRA_DB_TOKEN environment variable is not set")
	}
	authController := controller.NewAuthController(ctx, cfg, reg, dbToken, rts)
	authv1.RegisterAuthServiceServer(grpcServer, authController)

	errorGroup, ctx := errgroup.WithContext(ctx)

	// Start gRPC server goroutine
	errorGroup.Go(func() error {
		address := fmt.Sprintf(":%d", cfg.AuthPort)
		lis, err := net.Listen("tcp", address)
		if err != nil {
			return fmt.Errorf("failed to listen on %q: %w", address, err)
		}
		slog.Info("starting [gRPC] user service",
			slog.String("address", address),
		)
		if err := grpcServer.Serve(lis); err != nil {
			return fmt.Errorf("failed to serve gRPC service: %w", err)
		}
		return nil
	})

	// Shutdown goroutine
	errorGroup.Go(func() error {
		<-ctx.Done() // wait for signal
		slog.Info("shutting down gRPC server gracefully...")

		// stop gRPC
		grpcServer.GracefulStop()

		// close DB (make sure this is correct for your controller)
		if authController.Db != nil {
			authController.Db.Close()
			slog.Info("closed Cassandra session")
		}

		return ctx.Err()
	})
	return errorGroup.Wait()
}

func generateRedisClient(redisUrl string) (*redis.Client, error) {
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	return client, nil
}
