package controller

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	database "github.com/yaninyzwitty/chat/packages/db"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/monitoring"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserController struct {
	userv1.UnimplementedUserServiceServer
	Db     *gocql.Session
	M      *monitoring.Metrics
	Config *config.Config
}

func NewUserController(ctx context.Context, cfg *config.Config, reg *prometheus.Registry, token string) *UserController {
	m := monitoring.NewMetrics(reg)
	c := &UserController{
		Config: cfg,
		M:      m,
	}

	// use token only if you are using serverless cassandra
	if token != "" {
		c.Db = database.ConnectAstra(cfg, token)
		return c
	} else {
		// use this for testing
		c.Db = database.ConnectLocal(cfg.DatabaseConfig.Local_Host, cfg.DatabaseConfig.LocalDBPort)
	}
	return c

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

	// insert into Cassandra
	if err := c.Db.Query(
		`INSERT INTO chat.users (id, name, alias_name, created_at, updated_at, email, password) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		userID, req.Name, req.AliasName, now, now, req.Email, string(hashedPassword),
	).Exec(); err != nil {
		c.observeError(op, "cassandra")
		return nil, status.Errorf(codes.Internal, "failed to insert user: %v", err)
	}

	c.observeDuration(op, "cassandra", start)
	return &userv1.CreateUserResponse{
		User: &userv1.User{
			Id:        userID.String(),
			Name:      req.Name,
			Email:     req.Email,
			AliasName: req.AliasName,
			CreatedAt: timestamppb.New(now),
			UpdatedAt: timestamppb.New(now),
		},
	}, nil
}

// --- GET USER ---
func (c *UserController) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	start := time.Now()
	const op = "get_user"

	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	userID, err := gocql.ParseUUID(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID: %v", err)
	}

	var (
		name      string
		aliasName string
		email     string
		createdAt time.Time
		updatedAt time.Time
	)

	err = c.Db.Query(
		`SELECT name, alias_name, created_at, updated_at, email FROM chat.users WHERE id = ?`,
		userID,
	).Consistency(gocql.One).Scan(&name, &aliasName, &createdAt, &updatedAt, &email)

	if err != nil {
		c.observeError(op, "cassandra")
		if err == gocql.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to query user: %v", err)
	}

	c.observeDuration(op, "cassandra", start)
	return &userv1.GetUserResponse{
		User: &userv1.User{
			Id:        req.Id,
			Name:      name,
			AliasName: aliasName,
			Email:     email,
			CreatedAt: timestamppb.New(createdAt),
			UpdatedAt: timestamppb.New(updatedAt),
		},
	}, nil
}

// --- LIST USERS ---
func (c *UserController) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	start := time.Now()
	const op = "list_users"

	pageSize := int(req.GetPageLimit())
	if pageSize <= 0 {
		pageSize = 10
	}

	q := c.Db.Query(
		`SELECT id, name, alias_name, created_at, updated_at, email FROM chat.users`,
	).PageSize(pageSize)

	if len(req.GetPageToken()) > 0 {
		q = q.PageState(req.GetPageToken())
	}

	iter := q.Iter()
	defer iter.Close()

	var users []*userv1.User
	var (
		id        gocql.UUID
		name      string
		aliasName string
		createdAt time.Time
		updatedAt time.Time
		email     string
	)

	for iter.Scan(&id, &name, &aliasName, &createdAt, &updatedAt, &email) {
		users = append(users, &userv1.User{
			Id:        id.String(),
			Name:      name,
			AliasName: aliasName,
			CreatedAt: timestamppb.New(createdAt),
			UpdatedAt: timestamppb.New(updatedAt),
			Email:     email,
		})
	}

	nextPage := iter.PageState()
	if err := iter.Close(); err != nil {
		c.observeError(op, "cassandra")
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	c.observeDuration(op, "cassandra", start)
	return &userv1.ListUsersResponse{
		Users:     users,
		PageToken: nextPage, // requires proto update
	}, nil
}

// --- metrics helpers ---
func (c *UserController) observeDuration(op, db string, start time.Time) {
	c.M.Duration.WithLabelValues(op, db).Observe(time.Since(start).Seconds())
}

func (c *UserController) observeError(op, db string) {
	c.M.Errors.WithLabelValues(op, db).Inc()
}
