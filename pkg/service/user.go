package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"github.com/praslar/lib/common"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"ms-user/conf"
	"ms-user/pkg/model"
	"ms-user/pkg/repo"
	"ms-user/pkg/utils"
	"ms-user/pkg/valid"
	"net/http"
	"strings"
	"time"
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
	Login(ctx context.Context, req model.CreateUserReq) (rs model.ConfirmLoginResponse, err error)
	ParseAccessToken(str string) (*model.AccessTokenClaims, error)
	GetOneUserByID(ctx context.Context, userID uuid.UUID) (res model.User, er error)
}

type AccessTokenClaims struct {
	DeviceID       string `json:"device_id"`
	PermissionKeys string `json:"permission_keys"`
	jwt.StandardClaims
}

type RefreshTokenClaims struct {
	DeviceID       string `json:"did"`
	BusinessID     string `json:"business_id"`
	PermissionKeys string `json:"permission_keys"`
	jwt.StandardClaims
}

func (s *UserService) TestMsUser(ctx context.Context) error {
	log := logger.WithCtx(ctx, "UserService.TestMsUser")

	if err := s.repo.TestMsUser(ctx); err != nil {
		return err
	}

	log.Info("TestMsUser: Test ms-user success")

	return nil
}

func (s *UserService) CreateUser(ctx context.Context, req model.CreateUserReq) (rs model.User, err error) {
	log := logger.WithCtx(ctx, "UserService.CreateUser")

	//get email
	email := strings.Trim(valid.String(req.Email), "")

	// validate email
	if ok := utils.ValidateEmail(email); !ok {
		log.Error("error_400: Email invalid")
		return rs, ginext.NewError(http.StatusBadRequest, "Email invalid")
	}

	user, err := s.repo.GetOneUserByEmail(ctx, email, nil)
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
		log.Error("error_400: Password invalid in CreateUser - UserService")
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

func (s *UserService) Login(ctx context.Context, req model.CreateUserReq) (rs model.ConfirmLoginResponse, err error) {
	log := logger.WithCtx(ctx, "UserService.Login")

	//get email
	email := strings.Trim(valid.String(req.Email), " ")
	user, err := s.repo.GetOneUserByEmail(ctx, email, nil)
	if err != nil {
		return rs, err
	}

	// check password
	if err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(valid.String(req.Password))); err != nil {
		log.Error("error_400: Password incorrect in Login - UserService")
		return rs, ginext.NewError(http.StatusUnauthorized, "account or password incorrect")
	}

	// create refresh_token
	rs.RefreshToken, err = s.CreateRefreshToken(ctx, user.ID, "")
	if err != nil {
		return rs, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	// create access_token
	rs.Token, err = utils.CreateToken(model.CreateTokenRequest{
		UserID:  user.ID.String(),
		NumHour: conf.LoadEnv().NumHourExpToken,
		Extra:   "",
	})
	if err != nil {
		return rs, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	return rs, nil
}

// CreateRefreshToken makes a new refresh token, store information in DB then return the token string
func (s *UserService) CreateRefreshToken(ctx context.Context, userID uuid.UUID, extra string) (string, error) {
	now := time.Now().Unix()
	expiresAt := now + int64((time.Duration(conf.LoadEnv().RefreshTokenTTLInDays) * time.Hour * 24).Seconds())
	claims := &RefreshTokenClaims{
		StandardClaims: jwt.StandardClaims{
			Audience:  extra,
			IssuedAt:  now,
			ExpiresAt: expiresAt,
			Issuer:    "ms-user",
			Subject:   userID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(conf.LoadEnv().JWTSecret))
	if err != nil {
		return "", err
	}

	parts := strings.Split(signed, ".")
	dto := model.RefreshToken{
		Sign:   parts[2],
		UserID: userID,
	}
	err = s.repo.Transaction(context.Background(), func(rp repo.PGInterface) error {
		// delete existing refresh token in this device
		if err = s.repo.DeleteRefreshToken(ctx, userID, nil); err != nil {
			return err
		}

		// create new refresh token
		if err = s.repo.CreateRefreshToken(ctx, &dto, nil); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	return signed, nil
}

func (s *UserService) keyFunc(token *jwt.Token) (interface{}, error) {
	if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
		return nil, fmt.Errorf("unexpected signing method: %s", token.Header["alg"])
	}
	return []byte(conf.LoadEnv().JWTSecret), nil
}

// ParseRefreshToken ...
func (s *UserService) ParseRefreshToken(str string) (*RefreshTokenClaims, error) {
	token, err := jwt.ParseWithClaims(str, &RefreshTokenClaims{}, s.keyFunc)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, _ := token.Claims.(*RefreshTokenClaims)
	return claims, nil
}

// Parse Access token ...
func (s *UserService) ParseAccessToken(str string) (*model.AccessTokenClaims, error) {
	token, err := jwt.ParseWithClaims(str, &model.AccessTokenClaims{}, s.keyFunc)
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, _ := token.Claims.(*model.AccessTokenClaims)
	return claims, nil
}

// Get one user by id
func (s *UserService) GetOneUserByID(ctx context.Context, ID uuid.UUID) (res model.User, err error) {
	return s.repo.GetOneUserByID(ctx, ID, nil)
}
