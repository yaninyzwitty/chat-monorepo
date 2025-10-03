package handler

import (
	"context"
	"time"

	"github.com/gocql/gocql"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type UserHandler struct {
	Db *gocql.Session
}

func NewUserHandler(db *gocql.Session) *UserHandler {
	return &UserHandler{Db: db}
}

// --- DB INSERT ---
func (h *UserHandler) CreateUser(ctx context.Context, user *userv1.User, userPassword string) error {
	now := time.Now()

	if err := h.Db.Query(
		`INSERT INTO chat.users (id, name, alias_name, created_at, updated_at, email, password) 
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.Id, user.Name, user.AliasName, now, now, user.Email, userPassword,
	).Exec(); err != nil {
		return status.Errorf(codes.Internal, "failed to insert user: %v", err)
	}
	return nil
}

// --- DB SELECT ---
func (h *UserHandler) GetUser(ctx context.Context, id string) (*userv1.User, error) {
	userID, err := gocql.ParseUUID(id)
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

	if err := h.Db.Query(
		`SELECT name, alias_name, created_at, updated_at, email 
		 FROM chat.users WHERE id = ?`,
		userID,
	).Consistency(gocql.One).Scan(&name, &aliasName, &createdAt, &updatedAt, &email); err != nil {
		if err == gocql.ErrNotFound {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to query user: %v", err)
	}

	return &userv1.User{
		Id:        id,
		Name:      name,
		AliasName: aliasName,
		Email:     email,
		CreatedAt: timestamppb.New(createdAt),
		UpdatedAt: timestamppb.New(updatedAt),
	}, nil
}

// --- DB LIST ---
func (h *UserHandler) ListUsers(ctx context.Context, pageLimit int32, pageToken []byte) (*userv1.ListUsersResponse, error) {
	pageSize := int(pageLimit)
	if pageSize <= 0 {
		pageSize = 10
	}

	q := h.Db.Query(
		`SELECT id, name, alias_name, created_at, updated_at, email FROM chat.users`,
	).PageSize(pageSize)

	if len(pageToken) > 0 {
		q = q.PageState(pageToken)
	}

	iter := q.Iter()
	if err := iter.Close(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to close iter: %v", err)

	}

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

	if err := iter.Close(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list users: %v", err)
	}

	return &userv1.ListUsersResponse{
		Users:     users,
		PageToken: iter.PageState(),
	}, nil
}
