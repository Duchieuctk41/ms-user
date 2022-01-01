package service

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/repo"
	"finan/ms-order-management/pkg/utils"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"net/http"
)

type OrderTrackingService struct {
	repo repo.PGInterface
}

func NewOrderTrackingService(repo repo.PGInterface) OrderTrackingServiceInterface {
	return &OrderTrackingService{repo: repo}
}

type OrderTrackingServiceInterface interface {
	GetOrderTracking(ctx context.Context, req model.OrderTrackingRequest) (res model.OrderTrackingResponse, err error)
}

func (s *OrderTrackingService) GetOrderTracking(ctx context.Context, req model.OrderTrackingRequest) (res model.OrderTrackingResponse, err error) {
	log := logger.WithCtx(ctx, "OrderTrackingService.GetOrderTracking")

	lstOrderTracking, err := s.repo.GetOrderTracking(ctx, req, nil)
	if len(lstOrderTracking.Data) > 0 {
		// Get Order info
		order, err := s.repo.GetOneOrder(ctx, lstOrderTracking.Data[0].OrderID.String(), nil)
		if err != nil {
			log.WithError(err).Error("Error when GetOneOrder")
			return res, ginext.NewError(http.StatusBadRequest, "Error when get order")
		}

		// check permission
		if err = utils.CheckPermission(ctx, req.UserID.String(), order.BusinessID.String(), req.UserRole); err != nil {
			return res, err
		}
	}
	return lstOrderTracking, nil
}
