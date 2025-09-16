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
	"github.com/yaninyzwitty/chat/packages/shared/util"
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

	c.Db = database.DbConnect(ctx, cfg, token)
	return c
}

// CreateUser inserts a new user into Cassandra
func (c *UserController) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	if req.Name == "" || req.AliasName == "" || req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "the user's name and alias name are required")
	}

	userID := gocql.TimeUUID()
	now := time.Now()

	err := c.Db.Query(
		`INSERT INTO chat.users (id, name, alias_name, created_at, updated_at, email) VALUES (?, ?, ?, ?, ?, ?)`,
		userID, req.Name, req.AliasName, now, now, req.Email,
	).Exec()
	if err != nil {
		util.Fail(err, "failed to add user to db %v", err)
		return nil, status.Errorf(codes.Internal, "failed to insert user: %v", err)
	}

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

// GetUser retrieves a user by UUID
func (c *UserController) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
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
		if err == gocql.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to query user: %v", err)
	}

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

// ListUsers retrieves paginated users
func (c *UserController) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	pageSize := int(req.GetPageLimit())
	if pageSize <= 0 {
		pageSize = 10 // default
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
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	return &userv1.ListUsersResponse{
		Users:     users,
		PageToken: nextPage, // requires proto update
	}, nil
}
