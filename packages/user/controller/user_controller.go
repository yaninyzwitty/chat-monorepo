package controller

import (
	"context"

	"github.com/gocql/gocql"
	"github.com/prometheus/client_golang/prometheus"
	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
	database "github.com/yaninyzwitty/chat/packages/db"
	"github.com/yaninyzwitty/chat/packages/shared/config"
	"github.com/yaninyzwitty/chat/packages/shared/monitoring"
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

func (c *UserController) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	return nil, nil
}

func (c *UserController) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.GetUserResponse, error) {
	return nil, nil
}

func (c *UserController) ListUsers(ctx context.Context, req *userv1.ListUsersRequest) (*userv1.ListUsersResponse, error) {
	return nil, nil
}
