package handlers

import (
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/service"
	"finan/ms-order-management/pkg/utils"
	"net/http"

	"github.com/praslar/lib/common"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
)

type OrderHandlers struct {
	service service.OrderServiceInterface
}

func NewPoCategoryHandlers(service service.OrderServiceInterface) *OrderHandlers {
	return &OrderHandlers{service: service}
}

func (h *OrderHandlers) GetOneOrder(r *ginext.Request) (*ginext.Response, error) {
	return ginext.NewResponseData(http.StatusOK, "hello world"), nil
}

// CreateOrderFast Create order for Web POS combine with create product fast
func (h *OrderHandlers) CreateOrderFast(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.CreateOrderFast")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.OrderBody{}
	r.MustBind(&req)
	req.UserId = userID
	if err := common.CheckRequireValid(req); err != nil {
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	// Check Permission
	if req.BusinessId == nil {
		return nil, ginext.NewError(http.StatusUnauthorized, "You need input your business ID")
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessId.String(), role); err != nil {
		return nil, err
	}

	// create order
	rs, err := h.service.CreateOrder(r.Context(), req)
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

// UpdateOrder Update order for web POS, taken from UpdateOrderV5 function in ms-order-management
func (h *OrderHandlers) UpdateOrder(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.CreateOrderFast")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.OrderUpdateBody{}
	r.MustBind(&req)

	if req.ID = utils.ParseIDFromUri(r.GinCtx); req.ID == nil {
		return nil, ginext.NewError(http.StatusForbidden, "Wrong ID")
	}

	req.UpdaterID = &userID
	if err := common.CheckRequireValid(req); err != nil {
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	// Check Permission
	if req.BusinessId == nil {
		return nil, ginext.NewError(http.StatusUnauthorized, "You need input your business ID")
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessId.String(), role); err != nil {
		return nil, err
	}

	// create order
	rs, err := h.service.UpdateOrder(r.Context(), req)
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

// ProcessConsumer Receive message from rabbitmq
func (h *OrderHandlers) ProcessConsumer(r *ginext.Request) (*ginext.Response, error) {
	req := model.ProcessConsumerRequest{}
	r.MustBind(&req)
	res, err := h.service.ProcessConsumer(r.Context(), req)
	if err != nil {
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}
	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: res,
		},
	}, nil
}
