package handlers

import (
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/service"
	"finan/ms-order-management/pkg/utils"
	"github.com/praslar/lib/common"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"net/http"
)

type OrderTrackingHandlers struct {
	service service.OrderTrackingServiceInterface
}

func NewOrderTrackingHandlers(service service.OrderTrackingServiceInterface) *OrderTrackingHandlers {
	return &OrderTrackingHandlers{service: service}
}

func (h *OrderTrackingHandlers) GetOrderTracking(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetOrderTracking")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.OrderTrackingRequest{}
	r.MustBind(&req)
	req.UserID = userID
	req.UserRole = r.GinCtx.Request.Header.Get("x-user-roles")

	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Errorf("Invalid input: %v", err.Error())
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input: "+err.Error())
	}

	// get order tracking
	rs, err := h.service.GetOrderTracking(r.Context(), req)
	if err != nil {
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}
