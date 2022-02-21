package handlers

import (
	"encoding/json"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/service"
	"finan/ms-order-management/pkg/utils"
	"finan/ms-order-management/pkg/valid"
	"github.com/praslar/lib/common"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"net/http"
)

type PaymentOrderHistoryHandlers struct {
	service service.PaymentOrderHistoryInterface
}

func NewPaymentOrderHistoryHandlers(service service.PaymentOrderHistoryInterface) *PaymentOrderHistoryHandlers {
	return &PaymentOrderHistoryHandlers{service: service}
}

func (h *PaymentOrderHistoryHandlers) CreatePaymentOrderHistory(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.CreatePaymentOrderHistory")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.PaymentOrderHistoryRequest{}
	r.MustBind(&req)

	// log request information
	field, err := json.Marshal(req)
	if err != nil {
		log.WithError(err).Error("error_400: Cannot marshal request in CreatePaymentOrderHistory")
		return nil, ginext.NewError(http.StatusBadRequest, err.Error())
	}
	log.WithField("req", string(field)).Info("PaymentOrderHistoryHandlers.CreatePaymentOrderHistory")

	if err = common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("error_400: Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input:"+err.Error())
	}

	if valid.Float64(req.Amount) <= 0 {
		log.WithError(err).Error("error_400: Số tiền thanh toán phải lớn hơn hoặc bằng 0")
		return nil, ginext.NewError(http.StatusBadRequest, "Số tiền thanh toán phải lớn hơn hoặc bằng 0")
	}

	// check permission
	if err = utils.CheckPermissionV4(r.GinCtx, userID.String(), req.BusinessID.String()); err != nil {
		return nil, err
	}

	// Get one order
	rs, err := h.service.CreatePaymentOrderHistory(r.Context(), req, userID)
	if err != nil {
		log.WithError(err).Error("Fail to CreatePaymentOrderHistory")
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}

func (h *PaymentOrderHistoryHandlers) GetListPaymentOrderHistory(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetListPaymentOrderHistory")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.PaymentOrderHistoryParam{}
	r.MustBind(&req)

	// log request information
	field, err := json.Marshal(req)
	if err != nil {
		log.WithError(err).Error("error_400: Cannot marshal request in GetListPaymentOrderHistory")
		return nil, ginext.NewError(http.StatusBadRequest, err.Error())
	}
	log.WithField("req", string(field)).Info("PaymentOrderHistoryHandlers.GetListPaymentOrderHistory")

	if err = common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input:"+err.Error())
	}

	// check permission
	if err = utils.CheckPermissionV4(r.GinCtx, userID.String(), req.BusinessID); err != nil {
		return nil, err
	}

	// Get one order
	rs, err := h.service.GetListPaymentOrderHistory(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("Fail to GetListPaymentOrderHistory")
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}
