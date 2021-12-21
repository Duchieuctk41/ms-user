package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"gorm.io/gorm"
)

func (r *RepoPG) CreateOrderItem(ctx context.Context, orderItem model.OrderItem, tx *gorm.DB) (rs model.OrderItem, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := tx.Create(&orderItem).Error; err != nil {
		return model.OrderItem{}, err
	}

	return orderItem, nil
}
