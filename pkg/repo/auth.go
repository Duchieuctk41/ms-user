package repo

import (
	"context"
	"github.com/google/uuid"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"gorm.io/gorm"
	"ms-user/pkg/model"
	"ms-user/pkg/utils"
	"net/http"
)

func (r *RepoPG) DeleteRefreshToken(ctx context.Context, userID uuid.UUID, tx *gorm.DB) error {
	log := logger.WithCtx(ctx, "RepoPG.DeleteListBusiness")
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := tx.Debug().Where("user_id = ?", userID).Delete(&model.RefreshToken{}).Error; err != nil {
		log.WithError(err).Error("error_500 when call func DeleteRefreshToken")
		return ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	return nil
}

func (r *RepoPG) CreateRefreshToken(ctx context.Context, req *model.RefreshToken, tx *gorm.DB) error {
	log := logger.WithCtx(ctx, "RepoPG.CreateRefreshToken")
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := tx.Debug().Create(&req).Error; err != nil {
		log.WithError(err).Error("error_500 when call func CreateRefreshToken")
		return ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	return nil
}
