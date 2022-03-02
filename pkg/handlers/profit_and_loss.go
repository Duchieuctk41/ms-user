package handlers

import (
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/service"
	"finan/ms-order-management/pkg/utils"
	"finan/ms-order-management/pkg/valid"

	"net/http"

	"github.com/google/uuid"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
)

type ProfitAndLossHandlers struct {
	service service.ProfitAndLossServiceInterface
}

func NewProfitAndLossHandlers(service service.ProfitAndLossServiceInterface) *ProfitAndLossHandlers {
	return &ProfitAndLossHandlers{service: service}
}

func (h *ProfitAndLossHandlers) GetOverviewPandL(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "GetOverviewPandL When get overview")
	owner, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	var req model.OrverviewPandLRequest
	r.MustBind(&req)

	//Check Permission
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err := utils.CheckPermission(r.GinCtx, owner.String(), valid.String(req.BusinessID), role); err != nil {
		return nil, err
	}

	rs, err := h.service.OverviewPandL(r.GinCtx, req)
	if err != nil {
		return nil, err
	}

	return ginext.NewResponseData(http.StatusOK, rs), nil
}

func (h *ProfitAndLossHandlers) GetListProfitAndLoss(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "CreateOrUpdateStocks When Create SKU")
	owner, err := utils.CurrentUser(r.GinCtx.Request)
	if err != nil {
		log.WithError(err).Error("Error when get current user")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	var req model.ProfitAndLossRequest
	r.MustBind(&req)

	// Check Permission
	role := r.GinCtx.Request.Header.Get("x-user-roles")
	if err := utils.CheckPermission(r.GinCtx, owner.String(), valid.String(req.BusinessID), role); err != nil {
		return nil, err
	}

	rs, err := h.service.GetListProfitAndLoss(r.GinCtx, uuid.UUID{}, req)
	if err != nil {
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
