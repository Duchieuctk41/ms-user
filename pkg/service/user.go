package service

import (
	"context"
	"gitlab.com/goxp/cloud0/logger"
	"ms-user/pkg/repo"
)

type UserService struct {
	repo repo.PGInterface
}

func NewUserService(repo repo.PGInterface) UserInterface {
	return &UserService{repo: repo}
}

type UserInterface interface {
	TestMsUser(ctx context.Context) error
}

func (s *UserService) TestMsUser(ctx context.Context) error {
	log := logger.WithCtx(ctx, "UserService.TestMsUser")

	if err := s.repo.TestMsUser(ctx); err != nil {
		return err
	}

	log.Info("UserService: Test ms-user success")

	return nil
}
