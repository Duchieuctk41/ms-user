package handlers

import (
	"ms-user/pkg/model"
	"ms-user/pkg/service"
	"net/http"

	"github.com/praslar/lib/common"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
)

type UserHandlers struct {
	service service.UserInterface
}

func NewUserHandlers(service service.UserInterface) *UserHandlers {
	return &UserHandlers{service: service}
}

// TestMsUser - hieucn - 22/03/2022
func (h *UserHandlers) TestMsUser(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "UserHandlers.TestMsUser")

	if err := h.service.TestMsUser(r.Context()); err != nil {
		return nil, ginext.NewError(http.StatusBadRequest, "Fail to Test ms-user")
	}
	log.Info("UserHandlers: Test ms-user success")

	return &ginext.Response{
		Code: http.StatusOK,
		GeneralBody: &ginext.GeneralBody{
			Data: "test ms-user success",
		},
	}, nil
}

// 30/3/2022 - hieucn - register user with email, password
func (h *UserHandlers) CreateUser(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "UserHandlers.CreateUser")

	// Check valid request
	req := model.CreateUserReq{}
	r.MustBind(&req)
	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("error_400: Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input: "+err.Error())
	}

	rs, err := h.service.CreateUser(r.Context(), req)
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
