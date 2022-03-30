package repo

import (
	"context"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"gorm.io/gorm"
	"ms-user/pkg/model"
	"net/http"
)

func (r *RepoPG) TestMsUser(ctx context.Context) (err error) {
	log := logger.WithCtx(ctx, "RepoPG.TestMsUser")

	log.Info("RepoPG: Test ms-user success")

	return nil
}

func (r *RepoPG) GetOneUserByID(ctx context.Context, email string, tx *gorm.DB) (rs model.User, err error) {
	log := logger.WithCtx(ctx, "RepoPG.GetOneUserByID")
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Model(&model.User{}).Where("email = ?", email).First(&rs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.WithError(err).Error("error_404: record not found in GetOneUserByID - RepoPG")
			return rs, err
		}
		log.WithError(err).Error("error_500: error GetOneUserByID - RepoPG")
		return rs, ginext.NewError(http.StatusInternalServerError, err.Error())
	}
	return rs, nil
}

func (r *RepoPG) CreateUser(ctx context.Context, req *model.User, tx *gorm.DB) error {
	log := logger.WithCtx(ctx, "RepoPG.CreateUser")
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := tx.Model(&model.User{}).Create(&req).Error; err != nil {
		log.WithError(err).Error("error_500: error CreateUser - RepoPG")
		return ginext.NewError(http.StatusInternalServerError, err.Error())
	}
	req.Password = ""

	return nil
}
