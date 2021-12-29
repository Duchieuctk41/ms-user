package service

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/repo"

	"github.com/google/uuid"
	"gitlab.com/goxp/cloud0/logger"
)

type ProfitAndLossService struct {
	repo repo.PGInterface
}

func NewProfitAndLossService(repo repo.PGInterface) ProfitAndLossServiceInterface {
	return &ProfitAndLossService{repo: repo}
}

type ProfitAndLossServiceInterface interface {
	OverviewPandL(ctx context.Context, req model.OrverviewPandLRequest) (res model.OverviewPandLResponse, err error)
	GetListProfitAndLoss(ctx context.Context, currentUser uuid.UUID, req model.ProfitAndLossRequest) (res model.GetListProfitAndLossResponse, err error)
}

func (s *ProfitAndLossService) OverviewPandL(ctx context.Context, req model.OrverviewPandLRequest) (res model.OverviewPandLResponse, err error) {
	log := logger.WithCtx(ctx, "ProfitAndLossService.OverviewPandL").WithField("req", req)
	//

	overviewPandL, err := s.repo.OverviewSales(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("Get overview P&L sales error")
	}

	overviewPandL, err = s.repo.OverviewCost(ctx, req, overviewPandL, nil)
	if err != nil {
		log.WithError(err).Error("Get overview P&L cost error")
	}
	overviewPandL.ProfitTotal = overviewPandL.SumGrandTotal - overviewPandL.CostTotal

	return overviewPandL, nil
}

func (s *ProfitAndLossService) GetListProfitAndLoss(ctx context.Context, currentUser uuid.UUID, req model.ProfitAndLossRequest) (res model.GetListProfitAndLossResponse, err error) {
	log := logger.WithCtx(ctx, "ProfitAndLossService.GetListProfitAndLoss").WithField("req", req)

	res, err = s.repo.GetListProfitAndLoss(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("Get overview P&L cost error")
	}

	return res, nil
}
