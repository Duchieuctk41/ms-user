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

func (r *RepoPG) GetOrderTracking(ctx context.Context, req model.OrderTrackingRequest, tx *gorm.DB) (rs model.OrderTrackingResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	tx = tx.Where("order_id", []string{"order_id", "=", "?"})

	var total int64 = 0
	tx.Count(&total)
	if req.Page != 0 && req.PageSize != 0 {
		tx = tx.Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize)
	}

	if err = tx.Order("sort").Find(rs.Data).Error; err != nil {
		return rs, err
	}

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), req.Page, req.PageSize); err != nil {
		return rs, err
	}

	return rs, nil
}
