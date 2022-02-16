package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"gorm.io/gorm"
)

func (r *RepoPG) LogHistory(ctx context.Context, history model.History, tx *gorm.DB) (rs model.History, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Create(&history).Error; err != nil {
		return rs, err
	}

	return history, nil
}
