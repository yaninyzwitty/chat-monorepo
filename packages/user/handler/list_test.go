package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/require"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	"github.com/yaninyzwitty/chat/packages/user/handler"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func setupTest(t *testing.T) *handler.UserHandler {
	ctx := context.Background()
	dbSess, err := GetConnection(ctx)
	require.NoError(t, err)
	require.NotNil(t, dbSess)
	return handler.NewUserHandler(dbSess)
}

func TestUserHandler(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	h := setupTest(t)
	require.NotNil(t, h)

	// --- Table-driven test cases ---
	testCases := []struct {
		name    string
		run     func(t *testing.T)
		wantErr bool
	}{
		{
			name: "CreateUser success",
			run: func(t *testing.T) {
				id := gocql.TimeUUID().String()
				user := &userv1.User{
					Id:        id,
					Name:      "test-user",
					AliasName: "alias",
					Email:     "test@example.com",
					CreatedAt: timestamppb.Now(),
					UpdatedAt: timestamppb.Now(),
				}
				err := h.CreateUser(ctx, user, "password123")
				require.NoError(t, err)

				got, err := h.GetUser(ctx, id)
				require.NoError(t, err)
				require.Equal(t, user.Name, got.Name)
				require.Equal(t, user.Email, got.Email)
			},
		},
		{
			name: "CreateUser with empty ID",
			run: func(t *testing.T) {
				user := &userv1.User{
					Id:        "",
					Name:      "test-user",
					AliasName: "alias",
					Email:     "noid@example.com",
					CreatedAt: timestamppb.Now(),
					UpdatedAt: timestamppb.Now(),
				}
				err := h.CreateUser(ctx, user, "password")
				require.Error(t, err)
			},
			wantErr: true,
		},
		{
			name: "GetUser not found",
			run: func(t *testing.T) {
				fakeUUID := gocql.TimeUUID().String()
				got, err := h.GetUser(ctx, fakeUUID)
				require.Error(t, err)
				require.Nil(t, got)
			},
			wantErr: true,
		},
		{
			name: "ListUsers",
			run: func(t *testing.T) {
				resp, err := h.ListUsers(ctx, 10, nil)
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.GreaterOrEqual(t, len(resp.Users), 1)
				for _, u := range resp.Users {
					require.NotEmpty(t, u.Id)
					require.NotEmpty(t, u.Email)
					require.NotNil(t, u.CreatedAt)
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.wantErr {
				require.NotPanics(t, func() { tc.run(t) })
			} else {
				require.NotPanics(t, func() { tc.run(t) })
			}
		})
	}
}

// Optional cleanup after all tests
func TestCleanup(t *testing.T) {
	Cleanup()
	time.Sleep(1 * time.Second)
}
