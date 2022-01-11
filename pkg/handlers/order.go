package handlers

import (
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/service"
	"finan/ms-order-management/pkg/utils"
	"finan/ms-order-management/pkg/valid"
	"net/http"

	"github.com/google/uuid"
	"github.com/praslar/lib/common"
	"github.com/sirupsen/logrus"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
)

type OrderHandlers struct {
	service service.OrderServiceInterface
}

func NewOrderHandlers(service service.OrderServiceInterface) *OrderHandlers {
	return &OrderHandlers{service: service}
}

// GetOneOrder - convert from /api/v2/get-one-oder - version app 1.0.35.1.4
func (h *OrderHandlers) GetOneOrder(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetOneOrder")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.GetOneOrderRequest{}
	r.MustBind(&req)

	req.UserID = userID

	// check permission
	req.UserRole = r.GinCtx.Request.Header.Get("x-user-roles")

	req.ID = utils.ParseStringIDFromUri(r.GinCtx)
	if req.ID == nil {
		log.WithError(err).Error("Wrong orderNumber %v", err.Error())
		return nil, ginext.NewError(http.StatusForbidden, "Wrong orderNumber")
	}

	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input:"+err.Error())
	}

	// Get one order
	rs, err := h.service.GetOneOrder(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("Fail to GetOneOrder")
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}

// GetAllOrder - convert from /api/get-all-order - version app 1.0.35.1.4
func (h *OrderHandlers) GetAllOrder(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetAllOrder")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, "Unauthorized"+err.Error())
	}

	// Check valid request
	req := model.OrderParam{}
	r.MustBind(&req)

	// check permission
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err := utils.CheckPermissionV2(r.Context(), role, userID, req.BusinessID, req.BuyerID); err != nil {
		return nil, ginext.NewError(http.StatusUnauthorized, err.Error())
	}

	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input"+err.Error())
	}

	rs, err := h.service.GetAllOrder(r.Context(), req)
	if err != nil {
		logrus.Errorf("Fail to get all order: %v", err)
		return nil, ginext.NewError(http.StatusBadRequest, "Fail to get all order: "+err.Error())
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs.Data,
			Meta: rs.Meta,
		},
	}, nil
}

// CountOrderState - convert from /api/count-order-state - version app 1.0.35.1.4
func (h *OrderHandlers) CountOrderState(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.CountOrderState")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.RevenueBusinessParam{}
	r.MustBind(&req)

	// Check Permission
	if uuid.MustParse(req.BusinessID) == uuid.Nil {
		log.WithError(err).Error("Missing business ID")
		return nil, ginext.NewError(http.StatusUnauthorized, "You need input your business ID")
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessID, role); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// count order state
	res, err := h.service.CountOrderState(r.Context(), req)
	if err != nil {
		log.WithError(err).Errorf("Fail to CountOrderState: %v", err.Error())
		return nil, ginext.NewError(http.StatusUnauthorized, "Fail to CountOrderState: "+err.Error())
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: res,
		},
	}, nil
}

// GetOrderByContact - convert from /api/get-order-by-contact - version app 1.0.35.1.4
func (h *OrderHandlers) GetOrderByContact(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetOrderByContact")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, "Unauthorized"+err.Error())
	}

	// Check valid request
	req := model.OrderByContactParam{}
	r.MustBind(&req)

	// Check Permission
	if uuid.MustParse(req.BusinessID) == uuid.Nil {
		log.WithError(err).Error("Missing business ID")
		return nil, ginext.NewError(http.StatusUnauthorized, "You need input your business ID")
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessID, role); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// get order by contact
	rs, err := h.service.GetOrderByContact(r.Context(), req)
	if err != nil {
		log.WithError(err).Errorf("Fail to GetOrderByContact: %v", err.Error())
		return nil, ginext.NewError(http.StatusBadRequest, "Fail to GetOrderByContact"+err.Error())
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs.Data,
			Meta: rs.Meta,
		},
	}, nil
}

// GetContactDelivering - convert from /api/v2/get-contact-delivering - version app 1.0.35.1.4
func (h *OrderHandlers) GetContactDelivering(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetContactDelivering")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Fail to get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.OrderParam{}
	r.MustBind(&req)

	// Check Permission
	if req.BusinessID == "" {
		log.WithError(err).Error("Missing business ID")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessID, role); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Get contact delivering
	rs, err := h.service.GetContactDelivering(r.Context(), req)
	if err != nil {
		log.WithError(err).Errorf("Fail to get contact have order due to %v", err.Error())
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs.Data,
			Meta: rs.Meta,
		},
	}, nil
}

// CreateOrderFast Create order for Web POS combine with create product fast - version app 1.0.35.1.4
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
	req.UserID = userID
	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input:"+err.Error())
	}

	// Check Permission
	if req.BusinessID == nil {
		log.WithError(err).Error("Missing business ID")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessID.String(), role); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// create order
	rs, err := h.service.CreateOrder(r.Context(), req)
	if err != nil {
		log.WithError(err).Errorf("Fail to create order %v", err.Error())
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}

// UpdateOrder - convert from /api/v5/update-order/{id} - version app 1.0.35.1.4
// Update order for web POS, taken from UpdateOrderV5 function in ms-order-management
func (h *OrderHandlers) UpdateOrder(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.UpdateOrder")

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
		log.WithError(err).Error("Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input:"+err.Error())
	}

	// Check Permission
	role := r.GinCtx.Request.Header.Get("x-user-roles")

	// update order
	rs, err := h.service.UpdateOrder(r.Context(), req, role)
	if err != nil {
		log.WithError(err).Errorf("Fail to update order: %v", err.Error())
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}

// UpdateDetailOrder - convert from /api/v3/update-detail-order/{id} - version app 1.0.35.1.4
func (h *OrderHandlers) UpdateDetailOrder(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.UpdateDetailOrder")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.UpdateDetailOrderRequest{}
	r.MustBind(&req)
	req.UpdaterID = &userID
	// Check Permission
	role := r.GinCtx.Request.Header.Get("x-user-roles")

	// parse ID from URI
	if req.ID = utils.ParseIDFromUri(r.GinCtx); req.ID == nil {
		log.WithError(err).Error("Lỗi: ID đơn hàng không đúng định dạng")
		return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: ID đơn hàng không đúng định dạng")
	}

	// implement the business logic of UpdateDetailOrder
	rs, err := h.service.UpdateDetailOrder(r.Context(), req, role)
	if err != nil {
		log.WithError(err).Errorf("Fail to update detail order: %v", err.Error())
		return nil, err
	}
	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}

// ExportOrderReport - convert from /api/export-order-report - version app 1.0.35.1.4
func (h *OrderHandlers) ExportOrderReport(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.ExportOrderReport")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.ExportOrderReportRequest{}
	r.MustBind(&req)

	// Check Permission
	if valid.UUID(req.BusinessID) == uuid.Nil {
		log.WithError(err).Error("Missing business ID")
		return nil, ginext.NewError(http.StatusUnauthorized, "You need input your business ID")
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessID.String(), role); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	//  Get data order list
	res, err := h.service.ExportOrderReport(r.Context(), req)
	if err != nil {
		log.WithError(err).Errorf("Fail to ExportOrderReport: %v", err.Error())
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: res,
		},
	}, nil
}

// GetListOrderEcom - convert from /api/v1/order-ecom/get-list - version app 1.0.35.1.4
func (h *OrderHandlers) GetListOrderEcom(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetListOrderEcom")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.OrderEcomRequest{}
	r.MustBind(&req)
	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input"+err.Error())
	}

	// Check Permission
	if req.BusinessID == nil {
		log.WithError(err).Error("Missing business ID")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err = utils.CheckPermission(r.GinCtx, userID.String(), *req.BusinessID, role); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Get list order ecom
	rs, err := h.service.GetListOrderEcom(r.Context(), req)
	if err != nil {
		logrus.Errorf("Fail to GetListOrderEcom due to %v", err)
		return nil, ginext.NewError(http.StatusBadRequest, "Fail to GetListOrderEcom due to "+err.Error())
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs.Data,
			Meta: rs.Meta,
		},
	}, nil
}

// Author: Hieucn
// CreateOrderV2 Create order version 2 - update from CreateOrderFast - version app 1.0.35.1.4
// Check: buyer mustn't change state, buyer mustn't create-product-fast
// Check: permission of seller
func (h *OrderHandlers) CreateOrderV2(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.CreateOrderV2")

	// check x-user-id
	userID, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	// Check valid request
	req := model.OrderBody{}
	r.MustBind(&req)
	req.UserID = userID
	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input:"+err.Error())
	}

	// Check Permission
	if req.CreateMethod == utils.SELLER_CREATE_METHOD {
		if req.BusinessID == nil {
			log.WithError(err).Error("Missing business ID")
			return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
		}

		role := r.GinCtx.Request.Header.Get("x-user-roles")
		if err = utils.CheckPermission(r.GinCtx, userID.String(), req.BusinessID.String(), role); err != nil {
			log.WithError(err).Error("Unauthorized")
			return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
		}
	}

	// create order
	rs, err := h.service.CreateOrderV2(r.Context(), req)
	if err != nil {
		log.WithError(err).Errorf("Fail to create order %v", err.Error())
		return nil, err
	}

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: rs,
		},
	}, nil
}

// ProcessConsumer Receive message from rabbitmq - version app 1.0.35.1.4
func (h *OrderHandlers) ProcessConsumer(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.ProcessConsumer")

	req := model.ProcessConsumerRequest{}
	r.MustBind(&req)
	res, err := h.service.ProcessConsumer(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("Fail to ProcessConsumer")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}
	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: res,
		},
	}, nil
}

////Send email order
//func (h *OrderHandlers) SendEmailOrder(r *ginext.Request) (*ginext.Response, error) {
//	log := logger.WithCtx(r.GinCtx, "OrderHandlers.SendEmailOrder")
//
//	req := model.SendEmailRequest{}
//	r.MustBind(&req)
//
//	res, err := h.service.SendEmailOrder(r.Context(), req)
//	if err != nil {
//		log.WithError(err).Errorf("Fail to SendEmailOrder: %v", err.Error())
//		return nil, ginext.NewError(http.StatusBadRequest, "Fail to SendEmailOrder")
//	}
//	return &ginext.Response{
//		Code: http.StatusOK,
//		GeneralBody: &ginext.GeneralBody{
//			Data: res,
//		},
//	}, nil
//}

func (h *OrderHandlers) CountDeliveringQuantity(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.CountDeliveringQuantity")

	req := model.CountQuantityInOrderRequest{}
	r.MustBind(&req)
	res, err := h.service.CountDeliveringQuantity(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("Fail to CountDeliveringQuantity")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}
	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: res,
		},
	}, nil
}

func (h *OrderHandlers) GetSumOrderCompleteContact(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "OrderHandlers.GetSumOrderCompleteContact")

	req := model.GetTotalOrderByBusinessRequest{}
	r.MustBind(&req)
	res, err := h.service.GetSumOrderCompleteContact(r.Context(), req)
	if err != nil {
		log.WithError(err).Error("Fail to GetSumOrderCompleteContact")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}
	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: res,
		},
	}, nil
}
