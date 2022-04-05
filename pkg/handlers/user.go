package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/praslar/lib/common"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"ms-user/pkg/model"
	"ms-user/pkg/service"
	"ms-user/pkg/utils"
	"ms-user/pkg/valid"
	"net/http"
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

// 31/3/2022 - hieucn - login user with email & password
func (h *UserHandlers) Login(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "UserHandlers.Login")

	// Check valid request
	req := model.CreateUserReq{}
	r.MustBind(&req)
	if err := common.CheckRequireValid(req); err != nil {
		log.WithError(err).Error("error_400: Invalid input")
		return nil, ginext.NewError(http.StatusBadRequest, "Invalid input: "+err.Error())
	}

	rs, err := h.service.Login(r.Context(), req)
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

// 05/04/2022 - hieucn - VerifyTokenHandler
func (h *UserHandlers) VerifyTokenHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		log := logger.WithCtx(ctx, "UserHandlers.VerifyTokenHandler")

		req := model.OAuthVerifyRequest{}
		ctx.ShouldBind(&req)
		if err := common.CheckRequireValid(req); err != nil {
			log.WithError(err).Error("error_400: Invalid input")
			ginext.NewError(http.StatusBadRequest, "Invalid input: "+err.Error())
		}

		// do validate scopes latter
		claims, err := h.service.ParseAccessToken(req.Token)
		if err != nil {
			log.WithField("error", err).Error("parse token error")
			ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusUnauthorized])
		}

		// load user
		userID, err := utils.ExtractUserID(claims.Subject)
		if err != nil {
			log.WithField("error", err).Error("invalid userID in token payload")
			ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusUnauthorized])
		}

		//if _, err = h.service.GetOneUserByID(r.Context(), userID); err != nil {
		//	log.WithField("error", err).Error("failed to load user")
		//	ctx.Error(ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusUnauthorized]))
		//}

		// check roles if the request requires
		//if req.SecurityRoles > 0 {
		//	if uint64(user.UserRoles)&req.SecurityRoles == 0 {
		//		c.Ctx.Output.SetStatus(http.StatusForbidden)
		//		return
		//	}
		//}

		//rs := &model.OAuthVerifyResponseData{UserID: claims.Subject}
		ctx.Set("x-user-id", userID.String())
		ctx.Next()
	}
}

// 05/04/2022 - hieucn - get one user by id
func (h *UserHandlers) GetOneUserByID(r *ginext.Request) (*ginext.Response, error) {
	log := logger.WithCtx(r.GinCtx, "UserHandlers.GetOneUserByID")

	// parse ID from URI
	id := utils.ParseIDFromUri(r.GinCtx)
	if id == nil {
		log.Error("error_400: ID giao dịch không đúng định dạng")
		return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: ID giao dịch không đúng định dạng")
	}

	rs, err := h.service.GetOneUserByID(r.Context(), valid.UUID(id))
	if err != nil {
		return nil, err
	}

	return &ginext.Response{Code: http.StatusOK, GeneralBody: &ginext.GeneralBody{
		Data: rs,
	}}, nil
}
