package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"gorm.io/gorm"
)

func (r *RepoPG) CreateOrderTracking(ctx context.Context, orderTracking model.OrderTracking, tx *gorm.DB) (err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := r.DB.Create(&orderTracking).Error; err != nil {
		return err
	}
	return nil
}
