package repo

import (
	"finan/ms-order-management/pkg/model"
	"context"
)

func (r *RepoPG) CreateOrderTracking(ctx context.Context, orderTracking model.OrderTracking) (err error) {

	if err := r.DB.Create(&orderTracking).Error; err != nil {
		return err
	}
	return nil
}

