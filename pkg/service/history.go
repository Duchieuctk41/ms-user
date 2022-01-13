package service

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/repo"
	"gitlab.com/goxp/cloud0/logger"
)

type HistoryService struct {
	repo repo.PGInterface
}

func NewHistoryService(repo repo.PGInterface) HistoryServiceInterface {
	return &HistoryService{repo: repo}
}

type HistoryServiceInterface interface {
	LogHistory(ctx context.Context, req model.History)
}

func (s *HistoryService) LogHistory(ctx context.Context, req model.History) {
	log := logger.WithCtx(ctx, "HistoryService.LogHistory")

	_, err := s.repo.LogHistory(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("Fail to log history")
		return
	}
	return
}
