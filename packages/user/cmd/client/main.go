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
	"strconv"
	"syscall"
	"time"

	"github.com/rs/cors"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

	// ✅ grpc.Dial is correct
	userAddr := fmt.Sprintf("localhost:%d", cfg.UserClientPort)
	userConn, err := grpc.NewClient(userAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to dial grpc auth server: %w", err)
	}
	defer func() {

		if err := userConn.Close(); err != nil {
			slog.Error("failed to close", "error", err)
			os.Exit(1)
		}
	}()

	userClient := userv1.NewUserServiceClient(userConn)

	// Health route
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.Error("failed to write response", "error", err)
		}
	})
	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		type createUserPayload struct {
			Name      string `json:"name"`
			AliasName string `json:"alias_name"`
			Email     string `json:"email"`
			Password  string `json:"password"`
		}

		var payload createUserPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json body", http.StatusBadRequest)
			return
		}

		// Map to gRPC request
		grpcReq := &userv1.CreateUserRequest{
			Name:      payload.Name,
			AliasName: payload.AliasName,
			Email:     payload.Email,
			Password:  payload.Password,
		}

		// Call gRPC
		resp, err := userClient.CreateUser(r.Context(), grpcReq)
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				http.Error(w, st.Message(), httpStatusFromGrpc(st.Code()))
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		// Return JSON response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp.User); err != nil {
			slog.Error("failed to encode", "error", err)
		}
	})

	mux.HandleFunc("GET /users/", func(w http.ResponseWriter, r *http.Request) {
		// Expected: /users/{id}
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing user id", http.StatusBadRequest)
			return
		}

		grpcReq := &userv1.GetUserRequest{Id: id}
		resp, err := userClient.GetUser(r.Context(), grpcReq)
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				http.Error(w, st.Message(), httpStatusFromGrpc(st.Code()))
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp.User); err != nil {
			slog.Error("failed to encode JSON response", "error", err)
		}
	})

	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" {
			return // let the /users/{id} handler catch other patterns
		}

		query := r.URL.Query()
		pageLimitStr := query.Get("page_limit")
		pageToken := query.Get("page_token")

		var pageLimit uint32 = 20 // default
		if pageLimitStr != "" {
			if v, err := strconv.Atoi(pageLimitStr); err == nil {
				pageLimit = uint32(v)
			}
		}

		grpcReq := &userv1.ListUsersRequest{
			PageLimit: pageLimit,
			PageToken: []byte(pageToken),
		}

		resp, err := userClient.ListUsers(r.Context(), grpcReq)
		if err != nil {
			st, ok := status.FromError(err)
			if ok {
				http.Error(w, st.Message(), httpStatusFromGrpc(st.Code()))
				return
			}
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"users":      resp.Users,
			"page_token": string(resp.PageToken),
		}); err != nil {
			slog.Error("failed to encode JSON response", "error", err)
		}
	})

	// Wrap mux with CORS
	handler := cors.AllowAll().Handler(mux)

	// Server setup
	serverAddr := fmt.Sprintf(":%d", cfg.AuthPort)
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

func httpStatusFromGrpc(code codes.Code) int {
	switch code {
	case codes.NotFound:
		return http.StatusNotFound
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	default:
		return http.StatusInternalServerError
	}
}
