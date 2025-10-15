package handler_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/require"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	"github.com/yaninyzwitty/chat/packages/user/handler"
)

func TestCreateUser(t *testing.T) {
	ctx := context.Background()
	db, err := getConn()
	require.NoError(t, err)

	testCases := []struct {
		name  string
		setup func(ctx context.Context, db *gocql.Session) error
		input struct {
			user   *userv1.User
			passwd string
		}
		errors bool
	}{
		{
			name: "success:create_user",
			setup: func(ctx context.Context, db *gocql.Session) error {
				return db.Query("TRUNCATE init_sh_keyspace.users").Exec()
			},
			input: struct {
				user   *userv1.User
				passwd string
			}{
				user: &userv1.User{
					Id:        gocql.TimeUUID().String(),
					Name:      "Alice",
					AliasName: "Ali",
					Email:     "alice@example.com",
				},
				passwd: "secure-pass",
			},
			errors: false,
		},
		{
			name: "error:missing_id",
			setup: func(ctx context.Context, db *gocql.Session) error {
				return nil
			},
			input: struct {
				user   *userv1.User
				passwd string
			}{
				user: &userv1.User{
					Id:        "", // invalid UUID
					Name:      "Bob",
					AliasName: "Bobby",
					Email:     "bob@example.com",
				},
				passwd: "pass123",
			},
			errors: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.setup(ctx, db))

			h := handler.NewUserHandler(db)
			err := h.CreateUser(ctx, tc.input.user, tc.input.passwd)

			if tc.errors {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	ctx := context.Background()
	db, err := getConn()
	require.NoError(t, err)

	validID := gocql.TimeUUID()

	testCases := []struct {
		name   string
		setup  func(ctx context.Context, db *gocql.Session) error
		input  string
		errors bool
	}{
		{
			name: "success:get_existing_user",
			setup: func(ctx context.Context, db *gocql.Session) error {
				return db.Query(`INSERT INTO init_sh_keyspace.users (id, name, email, alias_name, created_at, updated_at, password)
					VALUES (?, ?, ?, ?, toTimestamp(now()), toTimestamp(now()), ?)`,
					validID, "Alice", "alice@example.com", "Ali", "pwd").Exec()
			},
			input:  validID.String(),
			errors: false,
		},
		{
			name:   "error:user_not_found",
			setup:  func(ctx context.Context, db *gocql.Session) error { return nil },
			input:  gocql.TimeUUID().String(), // random UUID
			errors: true,
		},
		{
			name:   "error:invalid_uuid",
			setup:  func(ctx context.Context, db *gocql.Session) error { return nil },
			input:  "not-a-uuid",
			errors: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.setup(ctx, db))

			h := handler.NewUserHandler(db)
			_, err := h.GetUser(ctx, tc.input)

			if tc.errors {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestListUsers(t *testing.T) {
	ctx := context.Background()
	db, err := getConn()
	require.NoError(t, err)

	testCases := []struct {
		name      string
		setup     func(ctx context.Context, db *gocql.Session) error
		limit     int32
		pageToken []byte
		expectLen int
		errors    bool
	}{
		{
			name: "success:list_first_page",
			setup: func(ctx context.Context, db *gocql.Session) error {
				_ = db.Query("TRUNCATE init_sh_keyspace.users").Exec()
				for i := 0; i < 5; i++ {
					_ = db.Query(`INSERT INTO init_sh_keyspace.users (id, name, email, alias_name, created_at, updated_at, password)
						VALUES (?, ?, ?, ?, toTimestamp(now()), toTimestamp(now()), ?)`,
						gocql.TimeUUID(), fmt.Sprintf("User%d", i), fmt.Sprintf("u%d@example.com", i), "Alias", "pwd").Exec()
				}
				return nil
			},
			limit:     3,
			pageToken: nil,
			expectLen: 3,
			errors:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, tc.setup(ctx, db))

			h := handler.NewUserHandler(db)
			res, err := h.ListUsers(ctx, tc.limit, tc.pageToken)

			if tc.errors {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Len(t, res.Users, tc.expectLen)
				require.NotNil(t, res.PageToken)
			}
		})
	}
}
