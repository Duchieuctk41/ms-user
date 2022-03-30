package service

import (
	"context"
	"fmt"
	"github.com/praslar/lib/common"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"ms-user/pkg/model"
	"ms-user/pkg/repo"
	"ms-user/pkg/utils"
	"ms-user/pkg/valid"
	"net/http"
	"strings"
)

type UserService struct {
	repo repo.PGInterface
}

func NewUserService(repo repo.PGInterface) UserInterface {
	return &UserService{repo: repo}
}

type UserInterface interface {
	TestMsUser(ctx context.Context) error
	CreateUser(ctx context.Context, req model.CreateUserReq) (rs model.User, err error)
}

func (s *UserService) TestMsUser(ctx context.Context) error {
	log := logger.WithCtx(ctx, "UserService.TestMsUser")

	if err := s.repo.TestMsUser(ctx); err != nil {
		return err
	}

	log.Info("UserService: Test ms-user success")

	return nil
}

func (s *UserService) CreateUser(ctx context.Context, req model.CreateUserReq) (rs model.User, err error) {
	log := logger.WithCtx(ctx, "UserService.TestMsUser")

	//get email
	email := strings.Trim(valid.String(req.Email), " ")

	// validate email
	if ok := utils.ValidateEmail(email); !ok {
		log.Error("error_400: Email invalid")
		return rs, ginext.NewError(http.StatusBadRequest, "Email invalid")
	}

	user, err := s.repo.GetOneUserByID(ctx, email, nil)
	if err != nil && err != gorm.ErrRecordNotFound {
		return rs, err
	}
	if user.Email == email {
		log.Error("error_400: This account has been existed")
		return rs, ginext.NewError(http.StatusBadRequest, "This account has been existed")
	}
	common.Sync(req, &rs)

	// verify password
	if err = utils.VerifyPassword(valid.String(req.Password)); err != nil {
		return rs, ginext.NewError(http.StatusBadRequest, fmt.Sprintf("Password invalid: %v", err.Error()))
	}

	// hash password
	hashPass, err := bcrypt.GenerateFromPassword([]byte(valid.String(req.Password)), bcrypt.DefaultCost)
	if err != nil {
		return rs, ginext.NewError(http.StatusBadRequest, "Cannot encode password")
	} else {
		rs.Password = string(hashPass)
	}

	if err = s.repo.CreateUser(ctx, &rs, nil); err != nil {
		return rs, err
	}

	return rs, nil
}
