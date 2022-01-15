package repo

import (
	"context"
	"encoding/json"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/utils"
	"finan/ms-order-management/pkg/valid"
	"fmt"
	"gitlab.com/goxp/cloud0/logger"
	"strings"
	"time"

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

func (r *RepoPG) CountOneStateOrder(ctx context.Context, businessId uuid.UUID, state string, tx *gorm.DB) int {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

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

func (r *RepoPG) RevenueBusiness(ctx context.Context, req model.RevenueBusinessParam, tx *gorm.DB) (model.RevenueBusiness, error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

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

func (r *RepoPG) GetContactHaveOrder(ctx context.Context, businessId uuid.UUID, tx *gorm.DB) (string, int, error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

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

func (r *RepoPG) GetOneOrder(ctx context.Context, id string, tx *gorm.DB) (rs model.Order, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if len(id) == 9 {
		if err = r.DB.Model(&model.Order{}).Where("order_number = ? AND deleted_at IS NULL", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("order_item.created_at ASC")
		}).First(&rs).Error; err != nil {
			return model.Order{}, err
		}
	} else {
		if err = r.DB.Model(&model.Order{}).Where("id = ? AND deleted_at IS NULL", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("order_item.created_at ASC")
		}).First(&rs).Error; err != nil {
			return model.Order{}, err
		}
	}

	return rs, nil
}

func (r *RepoPG) GetOneOrderRecent(ctx context.Context, buyerID string, tx *gorm.DB) (rs model.Order, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Model(&model.Order{}).Where("buyer_id = ?", buyerID).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return tx.Order("order_item.created_at ASC")
	}).Order("orders.created_at DESC").First(&rs).Error; err != nil {
		return model.Order{}, err
	}

	return rs, nil
}

func (r *RepoPG) UpdateOrder(ctx context.Context, order model.Order, tx *gorm.DB) (rs model.Order, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := tx.Model(&model.Order{}).Where("id = ?", order.ID).Save(&order).Error; err != nil {
		return model.Order{}, err
	}

	if err = tx.Model(&model.Order{}).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_item.created_at ASC")
	}).Where("id = ?", order.ID).First(&order).Error; err != nil {
		return model.Order{}, err
	}

	tx.Commit()
	return order, nil
}

func (r *RepoPG) OverviewSales(ctx context.Context, req model.OrverviewPandLRequest, tx *gorm.DB) (model.OverviewPandLResponse, error) {

	query := ""
	query += "SELECT SUM(grand_total) AS sum_grand_total, SUM(ordered_grand_total) as sum_ordered_grand_total, " +
		" SUM(promotion_discount) as sum_promotion_discount, " +
		" SUM(delivery_fee) as sum_delivery_fee, SUM(other_discount) as sum_other_discount " +
		" FROM orders " +
		" WHERE business_id = ? " +
		"  AND state = 'complete' "

	// 14/01/2021 - hieucn - fix compare nil time
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		query += " AND updated_at BETWEEN ? AND ? "
	}
	detailSales := model.DetailSales{}
	rs := model.OverviewPandLResponse{}
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&detailSales).Error; err != nil {
			return rs, err
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&detailSales).Error; err != nil {
			return rs, err
		}
	}
	rs.SumGrandTotal = detailSales.SumGrandTotal
	rs.DetailSales = detailSales
	return rs, nil
}

func (r *RepoPG) DetailSales(ctx context.Context, req model.OrverviewPandLRequest, tx *gorm.DB) (model.OverviewPandLResponse, error) {
	query := ""
	query += "SELECT SUM(grand_total) AS sum_grand_total " +
		" FROM orders " +
		" WHERE business_id = ? " +
		"  AND state = 'complete' "
	// 14/01/2021 - hieucn - fix compare nil time
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		query += " AND updated_at BETWEEN ? AND ? "
	}
	rs := model.OverviewPandLResponse{}
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return rs, err
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return rs, err
		}
	}
	return rs, nil
}

func (r *RepoPG) GetListOrderEcom(ctx context.Context, req model.OrderEcomRequest, tx *gorm.DB) (rs model.ListOrderEcomResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	var total int64 = 0
	tx = tx.Model(&model.OrderEcom{}).Where("business_id = ?", req.BusinessID)

	if req.Search != "" {
		tx = tx.Where("order_number ilike ? ", "%"+req.Search+"%")
	}

	// 14/01/2021 - hieucn - fix compare nil time
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		tx = tx.Where("created_time between ? and ?", req.StartTime, req.EndTime)
	}

	tx = tx.Count(&total)
	if req.Page != 0 && req.PageSize != 0 {
		tx = tx.Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize)
	}

	if req.Sort != "" {
		tx = tx.Order(req.Sort)
	} else {
		tx = tx.Order("created_time desc")
	}

	if err := tx.Find(&rs.Data).Error; err != nil {
		return rs, err
	}

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), req.Page, req.PageSize); err != nil {
		return rs, err
	}
	return rs, nil
}

func (r *RepoPG) GetAllOrder(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ListOrderResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	page := r.GetPage(req.Page)
	pageSize := r.GetPageSize(req.PageSize)

	tx = tx.Model(&model.Order{})

	if req.BusinessID != "" && req.SellerID == "" {
		tx = tx.Where("business_id = ? ", req.BusinessID)
	}

	if req.ContactID != "" {
		tx = tx.Where("contact_id = ? ", req.ContactID)
	}

	if req.OrderNumber != "" {
		tx = tx.Where("order_number = ?", req.OrderNumber)
	}

	if req.BuyerID != "" {
		tx = tx.Where("buyer_id = ? ", req.BuyerID)
	}

	if req.DeliveryMethod != nil {
		tx = tx.Where("delivery_method = ?", req.DeliveryMethod)
	}

	if req.SellerID != "" {
		// Get ra business_id tương ứng của thằng 1 rồi cho thằng 2 làm buyer_id và ngược lại
		if uhb1, err := utils.GetUserHasBusiness(req.SellerID, ""); err != nil {
			return rs, err
		} else if len(uhb1) == 0 {
			return rs, fmt.Errorf("Data business empty with user_id: %v", req.SellerID)
		} else {
			tx = tx.Where("business_id = ?", uhb1[0].BusinessID)
		}
	}

	if req.State != "" {
		req.State = strings.ReplaceAll(req.State, " ", "")
		stateArr := strings.Split(req.State, ",")
		tx = tx.Where("state IN (?) ", stateArr)
	} else {
		tx = tx.Where("state IN (?) ", []string{utils.ORDER_STATE_DELIVERING, utils.ORDER_STATE_COMPLETE, utils.ORDER_STATE_WAITING_CONFIRM, utils.ORDER_STATE_CANCEL})
	}

	if req.Search != "" {
		tx = tx.Where("order_number ilike ? OR unaccent(buyer_info->>'name') ilike ? OR buyer_info->>'phone_number' ilike ? OR (CONCAT('0', substring(buyer_info->>'phone_number' from 4))  ilike ?)", "%"+req.Search+"%", "%"+utils.TransformString(req.Search, false)+"%", "%"+req.Search+"%", "%"+req.Search+"%")
	}

	if req.DateFrom != nil && req.DateTo != nil {
		tx = tx.Where(" created_at BETWEEN ? AND ? ", req.DateFrom, req.DateTo)
	} else if req.DateFrom != nil && req.DateTo == nil {
		_, dateToStr := utils.ConvertTimestampVN(req.DateFrom, req.DateFrom)
		tx = tx.Where(" created_at BETWEEN ? AND ? ", req.DateFrom, dateToStr)
	}

	if req.IsPrinted != nil {
		tx = tx.Where("is_printed = ?", req.IsPrinted)
	}

	var total int64 = 0
	tx = tx.Count(&total)

	tx = tx.Order(req.Sort).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Where("order_item.deleted_at is null").Order("order_item.created_at asc")
	}).Limit(pageSize).Offset(r.GetOffset(page, pageSize)).Find(&rs.Data)

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), page, pageSize); err != nil {
		return rs, err
	}

	return rs, nil
}

func (r *RepoPG) GetCompleteOrders(ctx context.Context, contactID uuid.UUID, tx *gorm.DB) (res model.GetCompleteOrdersResponse, err error) {
	err = tx.Table("orders").
		Select("count(*) as count, sum(grand_total) as sum_amount").
		Where("contact_id = ?", contactID).Where("deleted_at is null").
		Where("state = ?", "complete").
		Find(&res).Error
	return
}

func (r *RepoPG) UpdateDetailOrder(ctx context.Context, order model.Order, mapItem map[string]model.OrderItem, tx *gorm.DB) (rs model.Order, stocks []model.StockRequest, err error) {
	log := logger.WithCtx(ctx, "RepoPG.UpdateDetailOrder")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		tx = tx.Begin()
		defer func() {
			tx.Rollback()
			cancel()
		}()
	}

	tMap := make(map[string]model.OrderItem)
	for i, _ := range order.OrderItem {
		if v, ok := mapItem[order.OrderItem[i].SkuID.String()]; ok {
			// If exists in map -> update quantity and money.
			// Cập nhật lại delivering quantity cho stock này
			if v.Quantity-order.OrderItem[i].Quantity != 0 {
				stocks = append(stocks, model.StockRequest{
					SkuID:          order.OrderItem[i].SkuID,
					QuantityChange: v.Quantity - order.OrderItem[i].Quantity,
				})
			}
			order.OrderItem[i].Quantity = v.Quantity
			order.OrderItem[i].TotalAmount = v.TotalAmount
			order.OrderItem[i].SkuName = v.SkuName
			order.OrderItem[i].ProductName = v.ProductName
			order.OrderItem[i].ProductNormalPrice = v.ProductNormalPrice
			order.OrderItem[i].ProductSellingPrice = v.ProductSellingPrice
			order.OrderItem[i].ProductImages = v.ProductImages
			order.OrderItem[i].SkuCode = v.SkuCode
			order.OrderItem[i].UOM = v.UOM
			order.OrderItem[i].Price = v.Price
			order.OrderItem[i].HistoricalCost = v.HistoricalCost
		} else {
			// If not exist in map -> delete
			tNow := time.Now()
			order.OrderItem[i].DeletedAt = &tNow
			// Giảm số lượng khách đang đặt cho SKU này (- delivering quantity)
			stocks = append(stocks, model.StockRequest{
				SkuID:          order.OrderItem[i].SkuID,
				QuantityChange: -order.OrderItem[i].Quantity,
			})
		}
		tMap[order.OrderItem[i].SkuID.String()] = order.OrderItem[i]
	}

	for skuID, item := range mapItem {
		if _, ok := tMap[skuID]; !ok {
			item.OrderID = order.ID
			order.OrderItem = append(order.OrderItem, item)
			// Thêm số lượng khách đang đặt bên stock (+ quantity)
			stocks = append(stocks, model.StockRequest{
				SkuID:          item.SkuID,
				QuantityChange: item.Quantity,
			})
		}
	}

	for _, orderItem := range order.OrderItem {
		if orderItem.ID == uuid.Nil {
			if err = tx.Model(&model.OrderItem{}).Create(&orderItem).Error; err != nil {
				return model.Order{}, nil, err
			}

			// log history order detail ver1
			go func() {
				history := model.History{
					BaseModel: model.BaseModel{
						CreatorID: orderItem.UpdaterID,
					},
					ObjectID:    orderItem.ID,
					ObjectTable: utils.TABLE_ORDER_ITEM,
					Action:      utils.ACTION_UPDATE_DETAIL_ORDER,
					Description: "Create order_item in UpdateDetailOrder ver1",
					Worker:      orderItem.CreatorID.String(),
				}

				dataOrder, err := json.Marshal(order)
				if err != nil {
					log.WithError(err).Error("Error when parse order in OrderDetail ver1")
					return
				}
				history.Data.RawMessage = dataOrder
				r.LogHistory(context.Background(), history, nil)
			}()
		} else {
			if err = tx.Model(&model.OrderItem{}).Where("id = ?", orderItem.ID).Updates(&orderItem).Error; err != nil {
				return model.Order{}, nil, err
			}

			// log history order detail ver1
			go func() {
				history := model.History{
					BaseModel: model.BaseModel{
						CreatorID: orderItem.UpdaterID,
					},
					ObjectID:    orderItem.ID,
					ObjectTable: utils.TABLE_ORDER_ITEM,
					Action:      utils.ACTION_UPDATE_DETAIL_ORDER,
					Description: "Update order_item in UpdateDetailOrder ver1",
					Worker:      orderItem.UpdaterID.String(),
				}

				dataOrder, err := json.Marshal(order)
				if err != nil {
					log.WithError(err).Error("Error when parse order in OrderDetail ver1")
					return
				}
				history.Data.RawMessage = dataOrder
				r.LogHistory(context.Background(), history, nil)
			}()
		}
	}

	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Save(&order).Error; err != nil {
		return model.Order{}, nil, err
	}

	if err = tx.Model(&model.Order{}).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Where("deleted_at IS NULL").Order("order_item.created_at ASC")
	}).Where("id = ?", order.ID).First(&order).Error; err != nil {
		return model.Order{}, nil, err
	}

	tx.Commit()
	return order, stocks, nil
}

func (r *RepoPG) CountOrderState(ctx context.Context, req model.RevenueBusinessParam, tx *gorm.DB) (res model.CountOrderState, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	query := ""
	query += "SELECT  " +
		"            COALESCE(MAX(DAT.count_waiting_confirm),0) AS count_waiting_confirm, " +
		"            COALESCE(MAX(DAT.count_delivering),0) AS count_delivering, " +
		"            COALESCE(MAX(DAT.count_complete),0) AS count_complete, " +
		"            COALESCE(MAX(DAT.count_cancel),0) AS count_cancel, " +
		"            DAT.business_id " +
		"     FROM (" +
		"        	SELECT " +
		"              business_id," +
		"              CASE WHEN state = 'create' THEN COUNT(*) END AS count_create," +
		"              CASE WHEN state = 'waiting_confirm' THEN COUNT(*) END AS count_waiting_confirm," +
		"              CASE WHEN state = 'readily_delivery' THEN COUNT(*) END AS count_readily_delivery, " +
		"              CASE WHEN state = 'delivering' THEN COUNT(*) END AS count_delivering, " +
		"              CASE WHEN state = 'complete' THEN COUNT(*) END AS count_complete, " +
		"              CASE WHEN state = 'cancel' THEN COUNT(*) END AS count_cancel " +
		" 		FROM orders " +
		" 		WHERE business_id = ? AND deleted_at IS NULL"
	if req.DateFrom != nil && req.DateTo != nil {
		query += " AND updated_at BETWEEN ? AND ? "
	}

	query += "  GROUP BY business_id, state " +
		" ) DAT " +
		" GROUP BY " +
		" DAT.business_id "

	rs := model.CountOrderState{}

	if req.DateFrom != nil && req.DateTo != nil {
		if err := tx.Raw(query, req.BusinessID, req.DateFrom, req.DateTo).Scan(&rs).Error; err != nil {
			return model.CountOrderState{}, err
		}
	} else {
		if err := tx.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.CountOrderState{}, err
		}
	}

	revenue, err := r.RevenueBusiness(ctx, model.RevenueBusinessParam{
		BusinessID: req.BusinessID,
		DateFrom:   req.DateFrom,
		DateTo:     req.DateTo,
	}, nil)
	if err != nil {
		return model.CountOrderState{}, err
	}

	orverviewPandLRequest := model.OrverviewPandLRequest{
		StartTime:  req.DateFrom,
		EndTime:    req.DateTo,
		BusinessID: &req.BusinessID,
	}
	var overviewPandL model.OverviewPandLResponse
	overviewPandL, err = r.OverviewCost(ctx, orverviewPandLRequest, overviewPandL, nil)
	rs.Profit = revenue.SumGrandTotal - overviewPandL.CostTotal
	rs.Revenue = revenue.SumGrandTotal
	return rs, nil
}

func (r *RepoPG) GetOrderByContact(ctx context.Context, req model.OrderByContactParam, tx *gorm.DB) (rs model.ListOrderResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	page := r.GetPage(req.Page)
	pageSize := r.GetPageSize(req.PageSize)

	tx = tx.Model(&model.Order{})

	if req.BusinessID != "" {
		tx = tx.Where("deleted_at IS NULL AND business_id = ? ", req.BusinessID)
	}

	if req.ContactID != "" {
		tx = tx.Where("contact_id = ? ", req.ContactID)
	}

	//if req.StartTime != nil && req.EndTime != nil {
	//	tx = tx.Where(" created_at BETWEEN ? AND ? ", req.StartTime, req.EndTime)
	//}

	// 14/01/2021 - hieucn - fix compare nil time
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		tx = tx.Where(" created_at BETWEEN ? AND ? ", req.StartTime, req.EndTime)
	}

	var total int64 = 0
	tx = tx.Count(&total)

	tx = tx.Order("created_at desc").Limit(pageSize).Offset(r.GetOffset(page, pageSize)).Preload("OrderItem").Find(&rs.Data)

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), page, pageSize); err != nil {
		return rs, err
	}

	return rs, nil
}

func (r *RepoPG) GetAllOrderForExport(ctx context.Context, req model.ExportOrderReportRequest, tx *gorm.DB) (orders []model.Order, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	tx = tx.Model(&model.Order{}).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_item.created_at ASC")
	}).Where("business_id = ?", req.BusinessID)
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		tx = tx.Where("created_at >= ? AND created_at <= ?", req.StartTime, req.EndTime)
	}
	if req.PaymentMethod != nil {
		tx = tx.Where("payment_method = ?", req.PaymentMethod)
	}
	if req.DeliveryMethod != nil {
		tx = tx.Where("delivery_method = ?", req.DeliveryMethod)
	}
	if req.State != nil {
		tx = tx.Where("state = ?", req.State)
	}
	err = tx.Order("created_at desc").Find(&orders).Error

	return
}

func (r *RepoPG) GetContactDelivering(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ContactDeliveringResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	page := r.GetPage(req.Page)
	pageSize := r.GetPageSize(req.PageSize)

	tx = tx.Model(model.Order{}).Select("contact_id, count(*) as count, max(created_at) as created_at, max(updated_at) as updated_at").Where("business_id = ?", req.BusinessID)

	if req.State != "" {
		t := strings.Split(req.State, ",")
		tx = tx.Where("state IN(?)", t)
	}

	var total int64 = 0
	tx = tx.Group("contact_id").Count(&total).Limit(pageSize).Offset(r.GetOffset(page, pageSize))

	if req.Sort != "" {
		tx = tx.Order(req.Sort)
	}

	if err = tx.Find(&rs.Data).Error; err != nil {
		return rs, err
	}

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), page, pageSize); err != nil {
		return rs, err
	}

	return rs, nil
}

func (r *RepoPG) GetCountQuantityInOrder(ctx context.Context, req model.CountQuantityInOrderRequest, tx *gorm.DB) (rs model.CountQuantityInOrderResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}
	query := `
		SELECT sum(quantity) AS SUM
		FROM order_item a
		LEFT JOIN orders b ON b.id = a.order_id
		WHERE a.sku_id = ?
		  AND a.deleted_at IS NULL
		  AND b.deleted_at IS NULL
		  AND b.business_id = ?
		  AND b.state IN (?)
		`

	if err := tx.Raw(query, req.SkuID, req.BusinessID, req.States).Scan(&rs).Error; err != nil {
		return rs, err
	}

	return rs, nil
}

func (r *RepoPG) CountOrderForTutorial(ctx context.Context, creatorID uuid.UUID, tx *gorm.DB) (count int, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	var total int64 = 0
	if err = tx.Model(model.Order{}).Where("creator_id = ?", creatorID).Unscoped().Count(&total).Error; err != nil {
		return 0, err
	}

	return int(total), nil
}
func (r *RepoPG) GetSumOrderCompleteContact(ctx context.Context, req model.GetTotalOrderByBusinessRequest, tx *gorm.DB) (rs []model.GetTotalOrderByBusinessResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}
	query := ""
	query += `select contact_id,
					count(*) as total_quantity_order,
					sum(grand_total) as total_amount_order
				from orders o
				where contact_id = ?
				and business_id = ?
				and state = 'complete'`
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		query += " AND updated_at BETWEEN ? AND ? "
	}
	query += "group by contact_id"
	if !valid.DayTime(req.StartTime).IsZero() && !valid.DayTime(req.EndTime).IsZero() {
		if err := tx.Raw(query, req.ContactID, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return rs, nil
		}
	} else {
		if err := tx.Raw(query, req.ContactID, req.BusinessID).Scan(&rs).Error; err != nil {
			return rs, nil
		}
	}
	return rs, nil
}
