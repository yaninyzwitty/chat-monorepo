package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/cors"
	authv1 "github.com/yaninyzwitty/chat/gen/auth/v1"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var logger *slog.Logger

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	if err := runClient(ctx, logger); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("error running application", slog.String("error", err.Error()))
	}
	logger.Info("server stopped cleanly")
}

func runClient(ctx context.Context, logger *slog.Logger) error {
	// Parse flags
	cp := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	cfg := &config.Config{}
	if *cp != "" {
		if err := cfg.LoadConfig(*cp); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// REST router
	mux := http.NewServeMux()

	authAddr := fmt.Sprintf("localhost:%d", cfg.AuthPort)
	authConn, err := grpc.NewClient(authAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to dial grpc auth server: %w", err)
	}
	defer func() {
		if err := authConn.Close(); err != nil {
			slog.Error("failed to close authConn", "error", err)
		}
	}()
	authClient := authv1.NewAuthServiceClient(authConn)

	// Health route
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			logger.Error("failed to write health response", slog.String("error", err.Error()))
		}
	})

	// ---- LOGIN ----
	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		grpcRes, err := authClient.Login(r.Context(), &authv1.LoginRequest{
			Email:    req.Email,
			Password: req.Password,
		})
		if err != nil {
			writeGrpcError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"token": grpcRes.GetTokens()})
	})

	// ---- REFRESH ----
	mux.HandleFunc("POST /refresh-token", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
			UserId       string `json:"user_id"`
		}
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		grpcRes, err := authClient.RefreshToken(r.Context(), &authv1.RefreshTokenRequest{
			RefreshToken: req.RefreshToken,
			UserId:       req.UserId,
		})
		if err != nil {
			writeGrpcError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"token": grpcRes.GetTokens()})
	})

	// ---- VALIDATE ----
	mux.HandleFunc("POST /validate-token", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AccessToken string `json:"access_token"`
		}
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		grpcRes, err := authClient.ValidateToken(r.Context(), &authv1.ValidateTokenRequest{
			AccessToken: req.AccessToken,
		})
		if err != nil {
			writeGrpcError(w, err)
			return
		}

		claims := grpcRes.GetClaims()

		writeJSON(w, http.StatusOK, map[string]any{
			"valid": grpcRes.Valid,
			"claims": map[string]any{
				"user_id":    claims.GetUserId(),
				"username":   claims.GetUsername(),
				"roles":      claims.GetRoles(),
				"issued_at":  timestamppb.New(claims.GetIssuedAt().AsTime()),
				"expires_at": timestamppb.New(claims.GetExpiresAt().AsTime()),
			},
		})
	})

	// ---- LOGOUT ----
	mux.HandleFunc("POST /logout", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
			UserId       string `json:"user_id"`
		}
		if decodeErr := json.NewDecoder(r.Body).Decode(&req); decodeErr != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		grpcRes, err := authClient.Logout(r.Context(), &authv1.LogoutRequest{
			RefreshToken: req.RefreshToken,
			UserId:       req.UserId,
		})
		if err != nil {
			writeGrpcError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"success": grpcRes.GetSuccess()})
	})

	// Wrap mux with CORS
	handler := cors.New(cors.Options{
		//				TODO -- ADJUST // AllowedOrigins:   []string{"http://localhost:3000"}, // adjust as needed
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}).Handler(mux)
	// Server setup
	serverAddr := fmt.Sprintf(":%d", cfg.AuthClientPort)
	srv := &http.Server{Addr: serverAddr, Handler: handler}

	go func() {
		logger.Info("starting REST proxy", slog.String("listen", serverAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("http server error", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeGrpcError(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		http.Error(w, st.Message(), http.StatusBadRequest)
	case codes.NotFound:
		http.Error(w, st.Message(), http.StatusNotFound)
	case codes.Unauthenticated:
		http.Error(w, st.Message(), http.StatusUnauthorized)
	case codes.PermissionDenied:
		http.Error(w, st.Message(), http.StatusForbidden)
	default:
		http.Error(w, st.Message(), http.StatusInternalServerError)
	}
}
