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
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	authjWT "github.com/yaninyzwitty/chat/packages/auth/jwt"
	database "github.com/yaninyzwitty/chat/packages/db"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/monitoring"
	"github.com/yaninyzwitty/chat/packages/user/controller"
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

	if cfg.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	addr := fmt.Sprintf(":%d", cfg.MetricsPort2)
	slog.Info("addr", "val", addr)
	// Prometheus metrics
	reg := prometheus.NewRegistry()
	monitoring.StartPrometheusServer(reg, addr)

	// gRPC server setup
	grpcServer := grpc.NewServer(
		// auth interceptor
		grpc.UnaryInterceptor(authjWT.AuthInterceptor()),
	)
	reflection.Register(grpcServer)

	// start godotenv

	if err := godotenv.Load(); err != nil {
		slog.Warn("Failed to load .env")
	}

	// Create controller with DB + metrics
	dbToken := os.Getenv("ASTRA_DB_TOKEN")
	if dbToken == "" {
		return errors.New("ASTRA_DB_TOKEN environment variable is not set")
	}
	
	db := database.ConnectAstra(cfg, dbToken)
	userController := controller.NewUserController(ctx, cfg, reg, dbToken, db)
	userv1.RegisterUserServiceServer(grpcServer, userController)

	errorGroup, ctx := errgroup.WithContext(ctx)

	// Start gRPC server goroutine
	errorGroup.Go(func() error {
		address := fmt.Sprintf(":%d", cfg.UserPort)

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

		// close DB
		if db != nil {
			db.Close()
			slog.Info("closed Cassandra session")
		}

		return ctx.Err()
	})

	return errorGroup.Wait()
}
