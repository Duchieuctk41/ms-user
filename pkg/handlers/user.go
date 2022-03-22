package handlers

import (
	"ms-user/pkg/service"
	"net/http"

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
