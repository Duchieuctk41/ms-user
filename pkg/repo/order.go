package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (r *RepoPG) CreateOrder(ctx context.Context, order model.Order, tx *gorm.DB) (rs model.Order, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := tx.Create(&order).Error; err != nil {
		return model.Order{}, err
	}

	return order, nil
}

func (r *RepoPG) CountOneStateOrder(businessId uuid.UUID, state string) int {
	query := ""
	query += "SELECT COUNT(*) count_state " +
		" FROM orders " +
		" WHERE business_id = ? " +
		" AND state = ? " +
		" GROUP BY state "

	var data struct {
		CountState int `json:"count_state"`
	}
	if err := r.DB.Raw(query, businessId, state).Scan(&data).Error; err != nil {
		return 0
	}

	return data.CountState
}

func (r *RepoPG) RevenueBusiness(ctx context.Context, req model.RevenueBusinessParam) (model.RevenueBusiness, error) {
	query := ""
	query += "SELECT SUM(grand_total) AS sum_grand_total " +
		" FROM orders " +
		" WHERE business_id = ? " +
		"  AND state = 'complete' "
	if req.DateFrom != nil && req.DateTo != nil {
		query += " AND updated_at BETWEEN ? AND ? "
	}
	rs := model.RevenueBusiness{}
	if req.DateFrom != nil && req.DateTo != nil {
		//dateFromStr, dateToStr := utils.ConvertTimestampVN(req.DateFrom, req.DateTo)

		if err := r.DB.Raw(query, req.BusinessID, req.DateFrom, req.DateTo).Scan(&rs).Error; err != nil {
			return model.RevenueBusiness{}, err
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.RevenueBusiness{}, err
		}
	}
	
	return rs, nil
}

func (r *RepoPG) GetContactHaveOrder(ctx context.Context, businessId uuid.UUID) (string, int, error) {
	query := ""
	query += "SELECT contact_id " +
		" FROM orders " +
		" WHERE business_id = ? " +
		" GROUP BY contact_id "

	type data struct {
		ContactId uuid.UUID `json:"contact_id"`
	}
	lstData := []data{}
	if err := r.DB.Raw(query, businessId).Scan(&lstData).Error; err != nil {
		return "", -1, err
	}
	contactIds := ""

	for i, _ := range lstData {
		if i == 0 {
			contactIds += lstData[i].ContactId.String()
		} else {
			contactIds += "," + lstData[i].ContactId.String()
		}
	}

	return contactIds, len(lstData), nil
}

func (r *RepoPG) GetOneOrder(ctx context.Context, id string) (rs model.Order, err error) {
	if len(id) == 9 {
		if err = r.DB.Model(&model.Order{}).Where("order_number = ?", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_item.created_at ASC")
		}).First(&rs).Error; err != nil {
			return model.Order{}, err
		}
	} else {
		if err = r.DB.Model(&model.Order{}).Where("id = ?", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_item.created_at ASC")
		}).First(&rs).Error; err != nil {
			return model.Order{}, err
		}
	}

	return rs, nil
}
