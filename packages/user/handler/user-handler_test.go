package handler_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gocql/gocql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/cassandra"
	"github.com/testcontainers/testcontainers-go/wait"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	database "github.com/yaninyzwitty/chat/packages/db"
	"github.com/yaninyzwitty/chat/packages/user/handler"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	testDB  *gocql.Session
	cleanup DBCleanupFunc
)

type DBCleanupFunc func(ctx context.Context) error

func TestMain(m *testing.M) {
	ctx := context.Background()

	session, cleanupFunc, err := CreateTestDB(ctx)
	if err != nil {
		panic(err)
	}

	testDB = session
	cleanup = cleanupFunc

	code := m.Run()

	// cleanup container
	if err := cleanup(ctx); err != nil {
		fmt.Println("failed cleanup:", err)
	}

	os.Exit(code)
}

func CreateTestDB(ctx context.Context) (*gocql.Session, DBCleanupFunc, error) {
	ctr, err := createTestContainer(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cassandra container: %w", err)
	}

	connectionHost, err := ctr.Host(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get cassandra container host: %w", err)
	}
	connectionPort, err := ctr.MappedPort(ctx, "9042")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get cassandra container port: %w", err)
	}

	session := database.ConnectLocal(connectionHost, connectionPort.Int())

	// cleanup
	cleanupFunc := func(ctx context.Context) error {
		session.Close()
		if err := ctr.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate cassandra container: %w", err)
		}
		return nil
	}

	return session, cleanupFunc, nil
}

func createTestContainer(ctx context.Context) (*cassandra.CassandraContainer, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	cqlScripts := workingDir + "/packages/db/testdata/init.cql"

	cassandraContainer, err := cassandra.Run(
		ctx,
		"cassandra:4.1.3",
		cassandra.WithInitScripts(cqlScripts),
		testcontainers.WithWaitStrategy(wait.ForLog("Starting server on port 9042")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cassandra container: %w", err)
	}
	return cassandraContainer, nil
}

func TestHandler_CreateAndGetUser(t *testing.T) {
	ctx := context.Background()
	h := handler.NewUserHandler(testDB)

	// create user via handler
	user := &userv1.User{
		Id:        gocql.TimeUUID().String(),
		Name:      "test-name",
		AliasName: "test-alias",
		Email:     "test-email@example.com",
		CreatedAt: timestamppb.Now(),
		UpdatedAt: timestamppb.Now(),
	}
	err := h.CreateUser(ctx, user, "supersecret")
	require.NoError(t, err, "expected user to be created successfully")

	// fetch user via handler
	got, err := h.GetUser(ctx, user.Id)
	require.NoError(t, err)
	assert.Equal(t, user.Name, got.Name)
	assert.Equal(t, user.AliasName, got.AliasName)
	assert.Equal(t, user.Email, got.Email)
}

func TestHandler_GetUser_NotFound(t *testing.T) {
	ctx := context.Background()
	h := handler.NewUserHandler(testDB)

	// random non-existent UUID
	nonExistentID := gocql.TimeUUID().String()
	got, err := h.GetUser(ctx, nonExistentID)

	assert.Nil(t, got)
	require.Error(t, err)

	st, ok := err.(interface{ GRPCStatus() *status.Status })
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.GRPCStatus().Code())
}

func TestHandler_ListUsers(t *testing.T) {
	ctx := context.Background()
	h := handler.NewUserHandler(testDB)

	// insert a few users via handler
	for i := 0; i < 3; i++ {
		u := &userv1.User{
			Id:        gocql.TimeUUID().String(),
			Name:      fmt.Sprintf("name-%d", i),
			AliasName: fmt.Sprintf("alias-%d", i),
			Email:     fmt.Sprintf("email-%d@test.com", i),
			CreatedAt: timestamppb.New(time.Now()),
			UpdatedAt: timestamppb.New(time.Now()),
		}
		require.NoError(t, h.CreateUser(ctx, u, "pwd"))
	}

	// list users with page size 2
	resp, err := h.ListUsers(ctx, 2, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Users)
	assert.LessOrEqual(t, len(resp.Users), 2)
}

func TestHandler_GetUser_InvalidUUID(t *testing.T) {
	ctx := context.Background()
	h := handler.NewUserHandler(testDB)

	got, err := h.GetUser(ctx, "not-a-uuid")
	assert.Nil(t, got)
	require.Error(t, err)

	st, ok := err.(interface{ GRPCStatus() *status.Status })
	require.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.GRPCStatus().Code())
}
