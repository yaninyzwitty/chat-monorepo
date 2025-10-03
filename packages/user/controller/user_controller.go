package controller

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/monitoring"
	"github.com/yaninyzwitty/chat/packages/user/handler"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserController struct {
	userv1.UnimplementedUserServiceServer
	h      *handler.UserHandler
	M      *monitoring.Metrics
	Config *config.Config
}

func NewUserController(ctx context.Context, cfg *config.Config, reg *prometheus.Registry, token string, db *gocql.Session) *UserController {
	m := monitoring.NewMetrics(reg)

	h := handler.NewUserHandler(db) // handler only gets DB session

	return &UserController{
		Config: cfg,
		M:      m,
		h:      h,
	}
}

// --- CREATE USER ---
func (c *UserController) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	start := time.Now()
	const op = "create_user"

	if req.Name == "" || req.AliasName == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "name, alias name, email, and password are required")
	}

	// hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.observeError(op, "bcrypt")
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	userID := gocql.TimeUUID()
	now := time.Now()

	user := &userv1.User{
		Id:        userID.String(),
		Name:      req.Name,
		Email:     req.Email,
		AliasName: req.AliasName,
		CreatedAt: timestamppb.New(now),
		UpdatedAt: timestamppb.New(now),
	}

	// delegate DB insert to handler
	if err := c.h.CreateUser(ctx, user, string(hashedPassword)); err != nil {
		c.observeError(op, "cassandra")
		return nil, err
	}

	c.observeDuration(op, "cassandra", start)
	return &userv1.CreateUserResponse{User: user}, nil
}

// --- GET USER ---
func (c *UserController) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	start := time.Now()
	const op = "get_user"

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	user, err := c.h.GetUser(ctx, req.Id)
	if err != nil {
		c.observeError(op, "cassandra")
		return nil, err
	}

	c.observeDuration(op, "cassandra", start)
	return &userv1.GetUserResponse{User: user}, nil
}

// --- LIST USERS ---
func (c *UserController) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	start := time.Now()
	const op = "list_users"

	usersResp, err := c.h.ListUsers(ctx, int32(req.GetPageLimit()), req.GetPageToken())
	if err != nil {
		c.observeError(op, "cassandra")
		return nil, err
	}

	c.observeDuration(op, "cassandra", start)
	return usersResp, nil
}

// --- metrics helpers ---
func (c *UserController) observeDuration(op, db string, start time.Time) {
	c.M.Duration.WithLabelValues(op, db).Observe(time.Since(start).Seconds())
}

func (c *UserController) observeError(op, db string) {
	c.M.Errors.WithLabelValues(op, db).Inc()
}
