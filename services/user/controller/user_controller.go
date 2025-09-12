package controller

import (
	"context"

	userv1 "github.com/yaninyzwitty/chat/gen/user/v1"
)

type UserController struct {
	userv1.UnimplementedUserServiceServer
}

func NewUserController() *UserController {
	return &UserController{}
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
