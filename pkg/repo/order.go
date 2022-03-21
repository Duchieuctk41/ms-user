package repo

import (
	"context"
	"encoding/json"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/utils"
	"finan/ms-order-management/pkg/valid"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm/clause"

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

func (r *RepoPG) CreateOrderV2(ctx context.Context, order *model.Order, tx *gorm.DB) error {
	log := logger.WithCtx(ctx, "RepoPG.CreateOrderV1")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err := tx.Create(&order).Error; err != nil {
		log.WithError(err).Error("error_500: create order in CreateOrderV1 - RepoPG")
		return ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	return nil
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
	log := logger.WithCtx(ctx, "RepoPG.GetOneOrder")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if len(id) == 9 {
		if err = tx.Model(&model.Order{}).Where("order_number = ?", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_item.created_at ASC")
		}).Preload("PaymentOrderHistory", func(db *gorm.DB) *gorm.DB {
			return db.Table("payment_order_history").Order("payment_order_history.created_at DESC")
		}).First(&rs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.WithError(err).Error("error_404: record not found in GetOneOrder - RepoPG")
				return rs, ginext.NewError(http.StatusNotFound, utils.MessageError()[http.StatusNotFound])
			} else {
				log.WithError(err).Error("error_500: get one order in GetOneOrder - RepoPG")
				return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
			}
		}
	} else {
		if err = tx.Model(&model.Order{}).Where("id = ?", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Order("order_item.created_at ASC")
		}).Preload("PaymentOrderHistory", func(db *gorm.DB) *gorm.DB {
			return db.Table("payment_order_history").Order("payment_order_history.created_at DESC")
		}).First(&rs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.WithError(err).Error("error_404: record not found in GetOneOrder - RepoPG")
				return rs, ginext.NewError(http.StatusNotFound, utils.MessageError()[http.StatusNotFound])
			} else {
				log.WithError(err).Error("error_500: get one order in GetOneOrder - RepoPG")
				return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
			}
		}
	}

	return rs, nil
}

func (r *RepoPG) GetStateOrderEcom(ctx context.Context, id string, tx *gorm.DB) (rs model.EcomOrderState, err error) {
	log := logger.WithCtx(ctx, "RepoPG.GetOneOrder")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if len(id) == 9 {
		if err = tx.Table("ecom_order").Select("id, state").Where("order_number = ?", id).First(&rs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.WithError(err).Error("error_404: record not found in GetOneOrderEcom - RepoPG")
				return rs, ginext.NewError(http.StatusNotFound, utils.MessageError()[http.StatusNotFound])
			} else {
				log.WithError(err).Error("error_500: get one order in GetOneOrderEcom - RepoPG")
				return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
			}
		}
	} else {
		if err = tx.Table("ecom_order").Select("id, state").Where("id = ?", id).First(&rs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.WithError(err).Error("error_404: record not found in GetOneOrderEcom - RepoPG")
				return rs, ginext.NewError(http.StatusNotFound, utils.MessageError()[http.StatusNotFound])
			} else {
				log.WithError(err).Error("error_500: get one order in GetOneOrderEcom - RepoPG")
				return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
			}
		}
	}

	return rs, nil
}

func (r *RepoPG) GetOneOrderBuyer(ctx context.Context, id string, tx *gorm.DB) (rs model.OrderBuyerResponse, err error) {
	log := logger.WithCtx(ctx, "RepoPG.GetOneOrderBuyer")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if len(id) == 9 {
		if err = tx.Table("orders").Where("order_number = ?", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Table("order_item").Order("order_item.created_at ASC")
		}).Preload("PaymentOrderHistory", func(db *gorm.DB) *gorm.DB {
			return db.Table("payment_order_history").Order("payment_order_history.created_at DESC")
		}).First(&rs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.WithError(err).Error("error_404: record not found in GetOnePaymentSource - RepoPG")
				return rs, ginext.NewError(http.StatusNotFound, utils.MessageError()[http.StatusNotFound])
			} else {
				log.WithError(err).Error("error_500: get one order in GetOneOrderBuyer - RepoPG")
				return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
			}
		}
	} else {
		if err = tx.Table("orders").Where("id = ?", id).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
			return db.Table("order_item").Order("order_item.created_at ASC")
		}).Preload("PaymentOrderHistory", func(db *gorm.DB) *gorm.DB {
			return db.Table("payment_order_history").Order("payment_order_history.created_at DESC")
		}).First(&rs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.WithError(err).Error("error_404: record not found in GetOnePaymentSource - RepoPG")
				return rs, ginext.NewError(http.StatusNotFound, utils.MessageError()[http.StatusNotFound])
			} else {
				log.WithError(err).Error("error_500: get one order in GetOneOrderBuyer - RepoPG")
				return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
			}
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
		return db.Order("order_item.created_at ASC")
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

	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Save(&order).Error; err != nil {
		return model.Order{}, err
	}

	if err = tx.Model(&model.Order{}).Where("orders.id = ?", order.ID).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_item.created_at ASC")
	}).First(&rs).Error; err != nil {
		return model.Order{}, err
	}

	tx.Commit()
	return order, nil
}

// 17/02/2022 - hieucn - just update order, don't get order, preload order_item in update func anymore
func (r *RepoPG) UpdateOrderV2(ctx context.Context, order model.Order, tx *gorm.DB) (rs model.Order, err error) {
	log := logger.WithCtx(ctx, "OrderHandlers.UpdateOrderV2")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Save(&order).Error; err != nil {
		log.WithError(err).Error("error_500: Error when UpdateOrderV2 - RepoPG")
		return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

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
	pageSize := r.GetPageSize(req.Size)

	if req.PageSize > 0 {
		pageSize = r.GetPageSize(req.PageSize)
	}

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
		return db.Order("order_item.created_at asc")
	}).Limit(pageSize).Offset(r.GetOffset(page, pageSize)).Find(&rs.Data)

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), page, pageSize); err != nil {
		return rs, err
	}

	if rs.Meta["total_pages"].(int) > page {
		rs.Meta["next_page"] = page + 1
	} else {
		rs.Meta["next_page"] = 0
	}

	return rs, nil
}

func (r *RepoPG) GetlistOrderV2(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ListOrderResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	page := r.GetPage(req.Page)
	pageSize := r.GetPageSize(req.Size)

	if req.PageSize > 0 {
		pageSize = r.GetPageSize(req.PageSize)
	}

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
		return db.Order("order_item.created_at asc")
	}).Preload("PaymentOrderHistory", func(db *gorm.DB) *gorm.DB {
		return db.Table("payment_order_history").Order("payment_order_history.created_at DESC")
	}).Limit(pageSize).Offset(r.GetOffset(page, pageSize)).Find(&rs.Data)

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), page, pageSize); err != nil {
		return rs, err
	}

	if rs.Meta["total_pages"].(int) > page {
		rs.Meta["next_page"] = page + 1
	} else {
		rs.Meta["next_page"] = 0
	}

	return rs, nil
}

func (r *RepoPG) GetCompleteOrders(ctx context.Context, contactID uuid.UUID, tx *gorm.DB) (res model.GetCompleteOrdersResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	err = tx.Model(model.Order{}).
		Select("count(*) as count, sum(grand_total) as sum_amount").
		Where("contact_id = ?", contactID).
		Where("state = ?", "complete").
		Find(&res).Error
	return
}

// 15/02/2022 - hieucn - fix multi request  call in one time
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
			tNow := gorm.DeletedAt{
				Time: time.Now(),
			}
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
			orderItem.CreatorID = order.UpdaterID
			//time.Sleep(3 * time.Second)
			if err = tx.FirstOrCreate(&orderItem, model.OrderItem{OrderID: order.ID, SkuID: orderItem.SkuID}).Error; err != nil {
				log.WithError(err).Error("error_500: create if exists order_item in UpdateDetailOrder - RepoPG")
				return model.Order{}, nil, err
			}

			// log history order item
			go func() {
				desc := utils.ACTION_CREATE_OR_SELECT_ORDER_ITEM + " in UpdateDetailOrder func - OrderService"
				history, _ := utils.PackHistoryModel(context.Background(), orderItem.UpdaterID, order.UpdaterID.String(), orderItem.ID, utils.TABLE_ORDER_ITEM, utils.ACTION_CREATE_OR_SELECT_ORDER_ITEM, desc, orderItem, mapItem)
				r.LogHistory(context.Background(), history, nil)
			}()
		} else {
			if orderItem.DeletedAt != nil {
				orderItem.UpdaterID = order.UpdaterID
				if err = tx.Where("id = ?", orderItem.ID).Delete(&orderItem).Error; err != nil {
					log.WithError(err).Error("error_500: delete order_item in UpdateDetailOrder - RepoPG")
					return model.Order{}, nil, err
				}

				// log history order item
				go func() {
					desc := utils.ACTION_DELETE_ORDER_ITEM + " in UpdateDetailOrder func - OrderService"
					history, _ := utils.PackHistoryModel(context.Background(), orderItem.UpdaterID, order.UpdaterID.String(), orderItem.ID, utils.TABLE_ORDER_ITEM, utils.ACTION_DELETE_ORDER_ITEM, desc, orderItem, mapItem)
					r.LogHistory(context.Background(), history, nil)
				}()
			} else {
				orderItem.UpdaterID = order.UpdaterID
				if err = tx.Model(&model.OrderItem{}).Where("id = ?", orderItem.ID).Save(&orderItem).Error; err != nil {
					log.WithError(err).Error("error_500: update order_item in UpdateDetailOrder - RepoPG")
					return model.Order{}, nil, err
				}
				// log history order item
				go func() {
					desc := utils.ACTION_UPDATE_ORDER_ITEM + " in UpdateDetailOrder func - OrderService"
					history, _ := utils.PackHistoryModel(context.Background(), order.UpdaterID, order.UpdaterID.String(), orderItem.ID, utils.TABLE_ORDER_ITEM, utils.ACTION_UPDATE_ORDER_ITEM, desc, orderItem, mapItem)
					r.LogHistory(context.Background(), history, nil)
				}()
			}
		}
	}

	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Save(&order).Error; err != nil {
		log.WithError(err).Error("error_500: update order in UpdateDetailOrder - RepoPG")
		return model.Order{}, nil, err
	}

	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_item.created_at ASC")
	}).Preload("PaymentOrderHistory", func(db *gorm.DB) *gorm.DB {
		return db.Table("payment_order_history").Order("payment_order_history.created_at DESC")
	}).First(&rs).Error; err != nil {
		return model.Order{}, nil, err
	}

	tx.Commit()
	return rs, stocks, nil
}

// version 1 - UpdateDetailOrder
//func (r *RepoPG) UpdateDetailOrder(ctx context.Context, order model.Order, mapItem map[string]model.OrderItem, tx *gorm.DB) (rs model.Order, stocks []model.StockRequest, err error) {
//	log := logger.WithCtx(ctx, "RepoPG.UpdateDetailOrder")
//
//	var cancel context.CancelFunc
//	if tx == nil {
//		tx, cancel = r.DBWithTimeout(ctx)
//		tx = tx.Begin()
//		defer func() {
//			tx.Rollback()
//			cancel()
//		}()
//	}
//
//	tMap := make(map[string]model.OrderItem)
//	for i, _ := range order.OrderItem {
//		if v, ok := mapItem[order.OrderItem[i].SkuID.String()]; ok {
//			// If exists in map -> update quantity and money.
//			// Cập nhật lại delivering quantity cho stock này
//			if v.Quantity-order.OrderItem[i].Quantity != 0 {
//				stocks = append(stocks, model.StockRequest{
//					SkuID:          order.OrderItem[i].SkuID,
//					QuantityChange: v.Quantity - order.OrderItem[i].Quantity,
//				})
//			}
//			order.OrderItem[i].Quantity = v.Quantity
//			order.OrderItem[i].TotalAmount = v.TotalAmount
//			order.OrderItem[i].SkuName = v.SkuName
//			order.OrderItem[i].ProductName = v.ProductName
//			order.OrderItem[i].ProductNormalPrice = v.ProductNormalPrice
//			order.OrderItem[i].ProductSellingPrice = v.ProductSellingPrice
//			order.OrderItem[i].ProductImages = v.ProductImages
//			order.OrderItem[i].SkuCode = v.SkuCode
//			order.OrderItem[i].UOM = v.UOM
//			order.OrderItem[i].Price = v.Price
//			order.OrderItem[i].HistoricalCost = v.HistoricalCost
//		} else {
//			// If not exist in map -> delete
//			tNow := gorm.DeletedAt{
//				Time: time.Now(),
//			}
//			order.OrderItem[i].DeletedAt = &tNow
//			// Giảm số lượng khách đang đặt cho SKU này (- delivering quantity)
//			stocks = append(stocks, model.StockRequest{
//				SkuID:          order.OrderItem[i].SkuID,
//				QuantityChange: -order.OrderItem[i].Quantity,
//			})
//		}
//		tMap[order.OrderItem[i].SkuID.String()] = order.OrderItem[i]
//	}
//
//	for skuID, item := range mapItem {
//		if _, ok := tMap[skuID]; !ok {
//			item.OrderID = order.ID
//			order.OrderItem = append(order.OrderItem, item)
//			// Thêm số lượng khách đang đặt bên stock (+ quantity)
//			stocks = append(stocks, model.StockRequest{
//				SkuID:          item.SkuID,
//				QuantityChange: item.Quantity,
//			})
//		}
//	}
//
//	for _, orderItem := range order.OrderItem {
//		if orderItem.ID == uuid.Nil {
//			if err = tx.Model(&model.OrderItem{}).Create(&orderItem).Error; err != nil {
//				return model.Order{}, nil, err
//			}
//
//			// log history order_item
//			go func() {
//				history := model.History{
//					BaseModel: model.BaseModel{
//						CreatorID: orderItem.UpdaterID,
//					},
//					ObjectID:    orderItem.ID,
//					ObjectTable: utils.TABLE_ORDER_ITEM,
//					Action:      utils.ACTION_UPDATE_ORDER_ITEM,
//					Description: utils.ACTION_UPDATE_ORDER_ITEM + " in UpdateDetailOrder func - OrderService",
//					Worker:      orderItem.CreatorID.String(),
//				}
//
//				tmpData, err := json.Marshal(orderItem)
//				if err != nil {
//					log.WithError(err).Error("Error when parse order in UpdateDetailOrder func - OrderService")
//					return
//				}
//				history.Data = tmpData
//
//				requestData, err := json.Marshal(mapItem)
//				if err != nil {
//					log.WithError(err).Error("Error when parse order request in UpdateDetailOrder - OrderService")
//					return
//				}
//				history.DataRequest = requestData
//
//				r.LogHistory(context.Background(), history, nil)
//			}()
//		} else {
//			if err = tx.Model(&model.OrderItem{}).Where("id = ?", orderItem.ID).Updates(&orderItem).Error; err != nil {
//				return model.Order{}, nil, err
//			}
//
//			// log history order_item
//			go func() {
//				history := model.History{
//					BaseModel: model.BaseModel{
//						CreatorID: orderItem.UpdaterID,
//					},
//					ObjectID:    orderItem.ID,
//					ObjectTable: utils.TABLE_ORDER_ITEM,
//					Action:      utils.ACTION_UPDATE_ORDER_ITEM,
//					Description: utils.ACTION_UPDATE_ORDER_ITEM + " in UpdateDetailOrder func - OrderService",
//					Worker:      orderItem.UpdaterID.String(),
//				}
//
//				tmpData, err := json.Marshal(orderItem)
//				if err != nil {
//					log.WithError(err).Error("Error when parse orderItem in UpdateDetailOrder func - OrderService")
//					return
//				}
//				history.Data = tmpData
//
//				requestData, err := json.Marshal(mapItem)
//				if err != nil {
//					log.WithError(err).Error("Error when parse order_item request in UpdateDetailOrder - OrderService")
//					return
//				}
//				history.DataRequest = requestData
//
//				r.LogHistory(context.Background(), history, nil)
//			}()
//		}
//	}
//
//	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Save(&order).Error; err != nil {
//		return model.Order{}, nil, err
//	}
//
//	if err = tx.Model(&model.Order{}).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
//		return db.Where("deleted_at IS NULL").Order("order_item.created_at ASC")
//	}).Where("id = ?", order.ID).First(&order).Error; err != nil {
//		return model.Order{}, nil, err
//	}
//
//	tx.Commit()
//	return order, stocks, nil
//}

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
		" 		WHERE business_id = ? "
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
		tx = tx.Where("business_id = ? ", req.BusinessID)
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

	tx = tx.Order("created_at desc").Limit(pageSize).Offset(r.GetOffset(page, pageSize)).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_item.created_at ASC")
	}).Find(&rs.Data)

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

func (r *RepoPG) GetTotalContactDelivery(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.TotalContactDelivery, err error) {
	log := logger.WithCtx(ctx, "RepoPG.GetTotalContactDelivery").WithField("req", req)

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	query := `
		 SELECT count(contact_id) as count FROM (
		 SELECT case when contact_id is not null THEN 1 END AS contact_id FROM "orders" 
		 WHERE business_id = ?
		`

	if req.State != "" {
		t := strings.Split(req.State, ",")
		query += " AND state IN (?) GROUP BY contact_id) tmp"
		if err = tx.Raw(query, req.BusinessID, t).Scan(&rs).Error; err != nil {
			log.WithError(err).Error("error_400: Error when GetTotalContactDelivery with state")
			return rs, ginext.NewError(http.StatusBadRequest, "Error when GetTotalContactDelivery")
		}
	} else {
		query += " GROUP BY contact_id ) tmp"
		if err = tx.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			log.WithError(err).Error("error_400: Error when GetTotalContactDelivery")
			return rs, ginext.NewError(http.StatusBadRequest, "Error when GetTotalContactDelivery")
		}
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

func (r *RepoPG) GetCountQuantityInOrderEcom(ctx context.Context, req model.CountQuantityInOrderRequest, tx *gorm.DB) (rs model.CountQuantityInOrderResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}
	query := `
		SELECT sum(quantity) AS SUM
		FROM ecom_order_item a
		LEFT JOIN ecom_orders b ON b.id = a.order_id
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

func (r *RepoPG) UpdateMultiOrderEcom(ctx context.Context, rs []model.OrderEcom, tx *gorm.DB) {
	start := time.Now()
	log := logger.WithCtx(ctx, "UpdateMultiOrderEcom")
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	eg := errgroup.Group{}

	for _, v := range rs {
		eg.Go(func() error {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				UpdateAll: true,
			}).Create(&v).Error; err != nil {
				log.WithError(err).WithField("order ecom ID ", v.ID).Error("error_500 : Error when create or update order ecom")
			}

			return nil
		})
	}

	_ = eg.Wait()

	// log time
	elapsed := time.Since(start)
	log.Printf("%s took %s for %s orders", "Storage order ecom", elapsed, strconv.Itoa(len(rs)))
}

func (r *RepoPG) UpdateMultiEcomOrder(ctx context.Context, rs []model.EcomOrder, tx *gorm.DB) {
	start := time.Now()
	log := logger.WithCtx(ctx, "UpdateMultiEcomOrder")
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	eg := errgroup.Group{}
	mapEcomOrder := make(map[string]model.EcomOrder)
	for _, v := range rs {
		tmp := v
		mapEcomOrder[v.ID.String()] = v
		eg.Go(func() error {
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				UpdateAll: true,
				//DoNothing: true,
			}).Omit("EcomOrderItem").Create(&tmp).Error; err != nil {
				//}).Create(&tmp).Error; err != nil {
				log.WithError(err).WithField("order ecom ID ", v.ID).Error("error_500 : Error when create or update order ecom")
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				UpdateAll: true,
			}).Create(&v.EcomOrderItem).Error; err != nil {
				log.WithError(err).WithField("order ecom ID ", v.ID).Error("error_500 : Error when create or update order item ecom")
			}
			return nil

		})

	}

	_ = eg.Wait()

	// log time
	elapsed := time.Since(start)
	log.Printf("%s took %s for %s orders", "Storage order ecom", elapsed, strconv.Itoa(len(rs)))
}

func PushConsumer(ctx context.Context, value interface{}, topic string) {
	log := logger.WithCtx(ctx, "PushConsumer")

	s, _ := json.Marshal(value)
	_, err := utils.PushConsumer(utils.ConsumerRequest{
		Topic: topic,
		Body:  string(s),
	})
	log.WithError(err).Error("PushConsumer topic: " + topic + " body: " + string(s))
	if err != nil {
		log.WithError(err).Error("Fail to push consumer " + topic + ": %")
	}
}

func (r *RepoPG) CountOrder(ctx context.Context, req model.OrverviewRequest, tx *gorm.DB) (model.OrderTotal, error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}
	rs := model.OrderTotal{}
	query := ""
	query += `SELECT COALESCE ( MAX ( TEMP.revenue_total ), 0 ) AS revenue_total,
						COALESCE ( MAX ( TEMP.order_complete_total ), 0 ) AS order_complete_total,
						COALESCE ( MAX ( TEMP.order_cancel_total ), 0 ) AS order_cancel_total,
						COALESCE ( MAX ( TEMP.order_delivering_total ), 0 ) AS order_delivering_total,
						COALESCE ( MAX ( TEMP.order_waiting_confirm_total ), 0 ) AS order_waiting_confirm_total 
					FROM
						(
						SELECT
						CASE WHEN STATE= 'complete' THEN
									SUM ( grand_total ) 
									END AS revenue_total,
							CASE WHEN STATE = 'complete' THEN
									COUNT ( * ) 
								END AS order_complete_total,
							CASE WHEN STATE = 'cancel' THEN
									COUNT ( * ) 
								END AS order_cancel_total,
							CASE WHEN STATE = 'delivering' THEN
									COUNT ( * ) 
								END AS order_delivering_total,
							CASE WHEN STATE = 'waiting_confirm' THEN
									COUNT ( * ) 
								END AS order_waiting_confirm_total 
							FROM
								orders o 
						WHERE
						business_id = ?`

	if req.StartTime != nil && req.EndTime != nil {
		query += ` AND updated_at BETWEEN ? AND ? `
	}
	query += `group by state ) as TEMP`

	if req.StartTime != nil && req.EndTime != nil {
		if err := tx.Raw(utils.RemoveSpace(query), req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return model.OrderTotal{}, nil
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.OrderTotal{}, err
		}
	}

	return rs, nil
}

func (r *RepoPG) GetOrderItemRevenueAnalytics(ctx context.Context, input model.GetOrderRevenueAnalyticsParam, tx *gorm.DB) (rs model.ListOrderRevenueAnalyticsResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}
	page := r.GetPage(input.Page)
	pageSize := r.GetPageSize(input.PageSize)

	if input.StartTime != nil && input.EndTime != nil {
		tx = tx.Where("orders.business_id = ? and order_item.deleted_at is null and order_item.updated_at >= ? and order_item.updated_at <= ? and orders.state ='complete'", input.BusinessID, input.StartTime, input.EndTime)
	}
	switch input.Sort {
	case "revenue":
		tx = tx.Model(model.OrderItem{}).Select("sku_id , product_name, sku_name, sum(total_amount) as total_amount, sum(quantity) as total_quantity").Joins("INNER JOIN orders ON orders.id = order_item.order_id").
			Group("sku_id, product_name, sku_name").Having("sum(total_amount) > 0").
			Order("sum(total_amount) desc")
	case "quantity":
		tx = tx.Model(model.OrderItem{}).Select("sku_id , product_name, sku_name, sum(total_amount) as total_amount,sum(quantity) as total_quantity").Joins("INNER JOIN orders ON orders.id = order_item.order_id").
			Group("sku_id, product_name, sku_name").Having("sum(quantity) > 0").
			Order("sum(quantity) desc")
	default:
		return rs, err
	}
	var total int64 = 0
	tx = tx.Count(&total).Limit(pageSize).Offset(r.GetOffset(page, pageSize))

	if err = tx.Find(&rs.Data).Error; err != nil {
		return rs, err
	}

	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), page, pageSize); err != nil {
		return rs, err
	}
	return rs, nil
}

// 01/03/2022 - hieucn - multi product line
func (r *RepoPG) UpdateDetailOrderSellerV2(ctx context.Context, order model.Order, lstItem []model.OrderItem, tx *gorm.DB) (rs model.Order, stocks []model.StockRequest, err error) {
	log := logger.WithCtx(ctx, "RepoPG.UpdateDetailOrderSellerV2")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		tx = tx.Begin()
		defer func() {
			tx.Rollback()
			cancel()
		}()
	}

	for _, item := range lstItem {
		item.OrderID = order.ID
		order.OrderItem = append(order.OrderItem, item)
		// Thêm số lượng khách đang đặt bên stock (+ quantity)
		stocks = append(stocks, model.StockRequest{
			SkuID:          item.SkuID,
			QuantityChange: item.Quantity,
		})
	}

	for _, orderItem := range order.OrderItem {
		if orderItem.ID == uuid.Nil {
			orderItem.CreatorID = order.UpdaterID
			//time.Sleep(3 * time.Second) - test multi request call in one time
			// 01/03/2022 - hieucn - change FirstAndCreate to Create
			if err = tx.Create(&orderItem).Error; err != nil {
				log.WithError(err).Error("error_500: create if exists order_item in UpdateDetailOrder - RepoPG")
				return model.Order{}, nil, err
			}

			// log history order item
			go func() {
				desc := utils.ACTION_CREATE_OR_SELECT_ORDER_ITEM + " in UpdateDetailOrderSellerV2 func - OrderService"
				history, _ := utils.PackHistoryModel(context.Background(), orderItem.UpdaterID, order.UpdaterID.String(), orderItem.ID, utils.TABLE_ORDER_ITEM, utils.ACTION_CREATE_OR_SELECT_ORDER_ITEM, desc, orderItem, nil)
				r.LogHistory(context.Background(), history, nil)
			}()
		} else {
			// 01/03/2022 - hieucn - delete old item, create new item
			orderItem.UpdaterID = order.UpdaterID
			if err = tx.Where("id = ?", orderItem.ID).Delete(&orderItem).Error; err != nil {
				log.WithError(err).Error("error_500: delete order_item in UpdateDetailOrderSellerV2 - RepoPG")
				return model.Order{}, nil, err
			}

			// log history order item
			go func() {
				desc := utils.ACTION_DELETE_ORDER_ITEM + " in UpdateDetailOrderSellerV2 func - OrderService"
				history, _ := utils.PackHistoryModel(context.Background(), orderItem.UpdaterID, order.UpdaterID.String(), orderItem.ID, utils.TABLE_ORDER_ITEM, utils.ACTION_DELETE_ORDER_ITEM, desc, orderItem, nil)
				r.LogHistory(context.Background(), history, nil)
			}()
		}
	}

	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Save(&order).Error; err != nil {
		log.WithError(err).Error("error_500: update order in UpdateDetailOrderSellerV2 - RepoPG")
		return model.Order{}, nil, err
	}

	if err = tx.Model(&model.Order{}).Where("id = ?", order.ID).Preload("OrderItem", func(db *gorm.DB) *gorm.DB {
		return db.Order("order_item.created_at ASC")
	}).Preload("PaymentOrderHistory", func(db *gorm.DB) *gorm.DB {
		return db.Table("payment_order_history").Order("payment_order_history.created_at DESC")
	}).First(&rs).Error; err != nil {
		return model.Order{}, nil, err
	}

	tx.Commit()
	return rs, stocks, nil
}

func (r *RepoPG) OrderDataDell(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.DataSell, error) {
	rs := model.DataSell{}
	querySellOrder := ""
	querySellOrder += `select
					temp.business_id,
					coalesce(sum(temp.offline_sell), 0) as offline_sell,
					coalesce(sum(temp.online_sell), 0) as online_sell
				from
					(
					select
						o.business_id ,
						case
							when create_method = 'seller' then sum(grand_total)
						end as offline_sell,
						case
							when create_method = 'buyer' then sum(grand_total)
						end as online_sell
					from
						orders o
					where
						business_id = ?`
	if req.StartTime != nil && req.EndTime != nil {
		querySellOrder += ` AND updated_at BETWEEN ? AND ? `
	}
	querySellOrder += `group by business_id ,create_method ) 
		as temp group by business_id`
	if req.StartTime != nil && req.EndTime != nil {
		if err := r.DB.Raw(querySellOrder, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return model.DataSell{}, nil
		}
	} else {
		if err := r.DB.Raw(querySellOrder, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.DataSell{}, nil
		}
	}
	if req.Ecom == utils.SHOPEE {
		queryEcommerceOrder := `select
									coalesce(sum(grand_total), 0) as ecommerce
								from
									ecom_order
								where
									business_id = ?`
		if req.StartTime != nil && req.EndTime != nil {
			queryEcommerceOrder += ` AND updated_at BETWEEN ? AND ? `
		}
		if req.StartTime != nil && req.EndTime != nil {
			if err := r.DB.Raw(queryEcommerceOrder, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
				return model.DataSell{}, nil
			}
		} else {
			if err := r.DB.Raw(queryEcommerceOrder, req.BusinessID).Scan(&rs).Error; err != nil {
				return model.DataSell{}, nil
			}

		}
	}

	return rs, nil
}

func (r *RepoPG) CountOrderAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.CountOrderAnalytics, error) {
	rs := model.CountOrderAnalytics{}
	query := ""
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_WEEK || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query += `select temp.business_id,
					COALESCE(MAX(temp.total_revenue),0) AS total_revenue,
					COALESCE(MAX(temp.total_cancel),0) AS total_cancel,
					COALESCE(MAX(temp.count_revenue),0) AS count_revenue,
					COALESCE(MAX(temp.count_cancel),0) AS count_cancel
   	from
   		(select o.business_id ,
   			case WHEN  state ='complete' then sum(grand_total) end as total_revenue ,
   			case WHEN  state ='cancel' then sum(grand_total) end as total_cancel ,
   			case WHEN  state ='complete' then count(*) end as count_revenue,
   			case WHEN  state ='cancel' then count(*) end as count_cancel 
  		 from orders o
  			where business_id = ?`

		if req.StartTime != nil && req.EndTime != nil {
			query += ` AND updated_at BETWEEN ? AND ? `
		}
		query += `group by business_id ,state ) 
		as temp group by business_id`
	} else {
		query += `select temp.business_id,
					COALESCE(sum(temp.total_revenue),0) AS total_revenue,
					COALESCE(sum(temp.last_period_total_revenue),0) AS last_period_total_revenue,
					COALESCE(sum(temp.total_cancel),0) AS total_cancel,
					COALESCE(sum(temp.last_period_total_cancel),0) AS last_period_total_cancel,
					COALESCE(sum(temp.count_revenue),0) AS count_revenue,
					COALESCE(sum(temp.last_period_count_revenue),0) AS last_period_count_revenue,
					COALESCE(sum(temp.count_cancel),0) AS count_cancel,
					COALESCE(sum(temp.last_period_count_cancel),0) AS last_period_count_cancel
				from
					  (select o.business_id ,
						   case WHEN state ='complete' and updated_at  > 'first_date'
							   then sum(grand_total) end as total_revenue,
						   case WHEN  state ='complete' and updated_at  < 'per_last_date'
							   then sum(grand_total) end as last_period_total_revenue,
						   case WHEN  state ='cancel' and updated_at  > 'first_date' 
							   then sum(grand_total) end as total_cancel ,
						   case WHEN  state ='cancel' and updated_at  < 'per_last_date' 
							  then sum(grand_total) end as last_period_total_cancel ,
						   case WHEN  state ='complete' and updated_at > 'first_date'
							  then count(*) end as count_revenue,
						   case WHEN  state ='complete' and updated_at < 'per_last_date'
							   then count(*) end as last_period_count_revenue,
						   case WHEN  state ='cancel' and updated_at  > 'first_date'
							   then count(*) end as count_cancel,
						   case WHEN  state ='cancel' and updated_at < 'per_last_date' 
							   then sum(grand_total) end as last_period_count_cancel
				   from orders o where business_id = ?
						  and ((updated_at) > 'per_first_date' AND (updated_at) < 'last_date')
				  group by business_id ,updated_at,state) as temp group by business_id;`
		now := time.Now().Add(time.Duration(-7) * time.Hour)
		switch req.Type {
		case utils.OPTION_FILTER_TODAY:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_WEEK:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_MONTH:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Add(time.Duration(-7)*time.Hour).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		default:
			return model.CountOrderAnalytics{}, nil
		}
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_WEEK || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		if req.StartTime != nil && req.EndTime != nil {
			if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
				return model.CountOrderAnalytics{}, nil
			}
		} else {
			return model.CountOrderAnalytics{}, nil
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.CountOrderAnalytics{}, nil
		}
	}

	revenue, err := r.CountBuyer(ctx, model.GetOrderAnalyticsRequest{
		BusinessID: req.BusinessID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Type:       req.Type,
	})
	if err != nil {
		return model.CountOrderAnalytics{}, err
	}

	buyerNew, err := r.CountBuyerNew(ctx, model.GetOrderAnalyticsRequest{
		BusinessID: req.BusinessID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Type:       req.Type,
	})
	if err != nil {
		return model.CountOrderAnalytics{}, err
	}
	rs.TotalBuyerNew = buyerNew.TotalBuyerNew
	rs.LastPeriodTotalBuyerNew = buyerNew.LastPeriodTotalBuyerNew
	rs.TotalBuyer = revenue.TotalBuyer
	rs.LastPeriodTotalBuyer = revenue.LastPeriodTotalBuyer
	return rs, nil
}

func (r *RepoPG) CountOrderEcomAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.CountOrderAnalytics, error) {
	rs := model.CountOrderAnalytics{}
	query := ""
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_WEEK || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query += `select temp.business_id,
					COALESCE(MAX(temp.total_revenue),0) AS total_revenue,
					COALESCE(MAX(temp.total_cancel),0) AS total_cancel,
					COALESCE(MAX(temp.count_revenue),0) AS count_revenue,
					COALESCE(MAX(temp.count_cancel),0) AS count_cancel
   	from
   		(select o.business_id ,
   			case WHEN  state ='complete' then sum(grand_total) end as total_revenue ,
   			case WHEN  state ='cancel' then sum(grand_total) end as total_cancel ,
   			case WHEN  state ='complete' then count(*) end as count_revenue,
   			case WHEN  state ='cancel' then count(*) end as count_cancel 
  		 from ecom_orders o
  			where business_id = ?`

		if req.StartTime != nil && req.EndTime != nil {
			query += ` AND updated_at BETWEEN ? AND ? `
		}
		query += `group by business_id ,state ) 
		as temp group by business_id`
	} else {
		query += `select temp.business_id,
					COALESCE(sum(temp.total_revenue),0) AS total_revenue,
					COALESCE(sum(temp.last_period_total_revenue),0) AS last_period_total_revenue,
					COALESCE(sum(temp.total_cancel),0) AS total_cancel,
					COALESCE(sum(temp.last_period_total_cancel),0) AS last_period_total_cancel,
					COALESCE(sum(temp.count_revenue),0) AS count_revenue,
					COALESCE(sum(temp.last_period_count_revenue),0) AS last_period_count_revenue,
					COALESCE(sum(temp.count_cancel),0) AS count_cancel,
					COALESCE(sum(temp.last_period_count_cancel),0) AS last_period_count_cancel
				from
					  (select o.business_id ,
						   case WHEN state ='complete' and updated_at  > 'first_date'
							   then sum(grand_total) end as total_revenue,
						   case WHEN  state ='complete' and updated_at  < 'per_last_date'
							   then sum(grand_total) end as last_period_total_revenue,
						   case WHEN  state ='cancel' and updated_at  > 'first_date' 
							   then sum(grand_total) end as total_cancel ,
						   case WHEN  state ='cancel' and updated_at  < 'per_last_date' 
							  then sum(grand_total) end as last_period_total_cancel ,
						   case WHEN  state ='complete' and updated_at > 'first_date'
							  then count(*) end as count_revenue,
						   case WHEN  state ='complete' and updated_at < 'per_last_date'
							   then count(*) end as last_period_count_revenue,
						   case WHEN  state ='cancel' and updated_at  > 'first_date'
							   then count(*) end as count_cancel,
						   case WHEN  state ='cancel' and updated_at < 'per_last_date' 
							   then sum(grand_total) end as last_period_count_cancel
				   from ecom_order o where business_id = ?
						  and ((updated_at) > 'per_first_date' AND (updated_at) < 'last_date')
				  group by business_id ,updated_at,state) as temp group by business_id;`
		now := time.Now().Add(time.Duration(-7) * time.Hour)
		switch req.Type {
		case utils.OPTION_FILTER_TODAY:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_WEEK:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_MONTH:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Add(time.Duration(-7)*time.Hour).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		default:
			return model.CountOrderAnalytics{}, nil
		}
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_WEEK || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		if req.StartTime != nil && req.EndTime != nil {
			if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
				return model.CountOrderAnalytics{}, nil
			}
		} else {
			return model.CountOrderAnalytics{}, nil
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.CountOrderAnalytics{}, nil
		}
	}

	revenue, err := r.CountEcomBuyer(ctx, model.GetOrderAnalyticsRequest{
		BusinessID: req.BusinessID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Type:       req.Type,
	})
	if err != nil {
		return model.CountOrderAnalytics{}, err
	}

	buyerNew, err := r.CountBuyerEcomNew(ctx, model.GetOrderAnalyticsRequest{
		BusinessID: req.BusinessID,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
		Type:       req.Type,
	})
	if err != nil {
		return model.CountOrderAnalytics{}, err
	}
	rs.TotalBuyerNew = buyerNew.TotalBuyerNew
	rs.LastPeriodTotalBuyerNew = buyerNew.LastPeriodTotalBuyerNew
	rs.TotalBuyer = revenue.TotalBuyer
	rs.LastPeriodTotalBuyer = revenue.LastPeriodTotalBuyer
	return rs, nil
}

func (r *RepoPG) CountBuyer(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.CountBuyer, error) {
	query := ""

	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		query += `select count(*) total_buyer from (select contact_id  from orders o
				where business_id = ?`
		if req.StartTime != nil && req.EndTime != nil {
			query += " AND created_at BETWEEN ? AND ? "
		}
		query += `group by contact_id ) as temp`
	} else {

		query += `select
					COALESCE(count(total_buyer),0) AS total_buyer,
					COALESCE(count(last_period_total_buyer),0) AS last_period_total_buyer
				   	from
					  (select distinct
						   case WHEN created_at  > 'first_date'
							   then contact_id end as total_buyer,
						   case WHEN  created_at  < 'per_last_date'
							   then contact_id end as last_period_total_buyer						  
				   	from orders o where business_id = ?
						  and (created_at > 'per_first_date' AND created_at < 'last_date')
				  	group by contact_id,created_at) as temp;`
		now := time.Now().Add(time.Duration(-7) * time.Hour)
		switch req.Type {
		case utils.OPTION_FILTER_TODAY:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_WEEK:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_MONTH:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Add(time.Duration(-7)*time.Hour).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		default:
			return model.CountBuyer{}, nil
		}
	}
	rs := model.CountBuyer{}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		if req.StartTime != nil && req.EndTime != nil {
			if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
				return model.CountBuyer{}, nil
			}
		} else {
			return model.CountBuyer{}, nil
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.CountBuyer{}, nil
		}
	}
	return rs, nil
}

func (r *RepoPG) CountEcomBuyer(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.CountBuyer, error) {
	query := ""

	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		query += `select count(*) total_buyer from (select contact_id  from ecom_order o
				where business_id = ?`
		if req.StartTime != nil && req.EndTime != nil {
			query += " AND created_at BETWEEN ? AND ? "
		}
		query += `group by contact_id ) as temp`
	} else {

		query += `select
					COALESCE(count(total_buyer),0) AS total_buyer,
					COALESCE(count(last_period_total_buyer),0) AS last_period_total_buyer
				   	from
					  (select distinct
						   case WHEN created_at  > 'first_date'
							   then contact_id end as total_buyer,
						   case WHEN  created_at  < 'per_last_date'
							   then contact_id end as last_period_total_buyer						  
				   	from ecom_order o where business_id = ?
						  and (created_at > 'per_first_date' AND created_at < 'last_date')
				  	group by contact_id,created_at) as temp;`
		now := time.Now().Add(time.Duration(-7) * time.Hour)
		switch req.Type {
		case utils.OPTION_FILTER_TODAY:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_WEEK:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_MONTH:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Add(time.Duration(-7)*time.Hour).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		default:
			return model.CountBuyer{}, nil
		}
	}
	rs := model.CountBuyer{}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		if req.StartTime != nil && req.EndTime != nil {
			if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
				return model.CountBuyer{}, nil
			}
		} else {
			return model.CountBuyer{}, nil
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&rs).Error; err != nil {
			return model.CountBuyer{}, nil
		}
	}
	return rs, nil
}

func (r *RepoPG) CountBuyerNew(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.CountBuyerNew, error) {
	query := ""

	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		query += `select count (*) as total_buyer_new, null as last_period_total_buyer_new from 
					(select distinct contact_id as total_buyer_new
				from orders o where business_id = 'business_id_req'
					and (created_at > ? AND created_at < ? )
		 		group by contact_id) temp  
					where total_buyer_new not IN (
		 		select distinct
				   contact_id  as last_period_total_buyer
			   from orders o where business_id = 'business_id_req'
					and created_at < ? and contact_id is not null 
	 	 		group by contact_id)`

		query = strings.ReplaceAll(query, "business_id_req", valid.String(req.BusinessID))
	} else {

		query += `select total_buyer_new, last_period_total_buyer_new
					from ( select generate_series(0, 0 ) as index, count (*) as total_buyer_new 
						from ( select distinct contact_id as total_buyer_new
							from orders o where business_id = 'req_business_id'
								and (created_at) > 'first_date'
								and (created_at) < 'last_date' 
							group by
								contact_id) temp
						where total_buyer_new not in (
							select distinct contact_id as last_period_total_buyer
							from orders o
							where
								business_id = 'req_business_id'
								and (created_at) < 'first_date'
									and contact_id is not null
								group by
									contact_id)
						) temp_order
					inner join (
						select generate_series(0, 0 ) as index, count (*) as last_period_total_buyer_new
						from ( select distinct contact_id as total_buyer_new
							from orders o where business_id = 'req_business_id'
								and (created_at) > 'per_first_date'
								and (created_at) < 'per_last_date' 
							group by
								contact_id) temp
						where total_buyer_new not in ( select distinct contact_id as last_period_total_buyer
							from orders o
							where business_id = 'req_business_id'
								and (created_at) < 'per_first_date'
									and contact_id is not null
								group by
									contact_id)) last_temp_order on
						temp_order.index = last_temp_order.index`

		query = strings.ReplaceAll(query, "req_business_id", valid.String(req.BusinessID))
		now := time.Now().Add(time.Duration(-7) * time.Hour)
		switch req.Type {
		case utils.OPTION_FILTER_TODAY:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_WEEK:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_MONTH:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Add(time.Duration(-7)*time.Hour).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		default:
			return model.CountBuyerNew{}, nil
		}
	}
	rs := model.CountBuyerNew{}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		if req.StartTime != nil && req.EndTime != nil {
			if err := r.DB.Raw(query, req.StartTime, req.EndTime, req.StartTime).Scan(&rs).Error; err != nil {
				return model.CountBuyerNew{}, nil
			}
		} else {
			return model.CountBuyerNew{}, nil
		}
	} else {
		if err := r.DB.Raw(query).Scan(&rs).Error; err != nil {
			return model.CountBuyerNew{}, nil
		}
	}
	return rs, nil
}

func (r *RepoPG) CountBuyerEcomNew(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.CountBuyerNew, error) {
	query := ""

	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		query += `select count (*) as total_buyer_new, null as last_period_total_buyer_new from 
					(select distinct contact_id as total_buyer_new
				from ecom_order o where business_id = 'business_id_req'
					and (created_at > ? AND created_at < ? )
		 		group by contact_id) temp  
					where total_buyer_new not IN (
		 		select distinct
				   contact_id  as last_period_total_buyer
			   from ecom_order o where business_id = 'business_id_req'
					and created_at < ? and contact_id is not null 
	 	 		group by contact_id)`

		query = strings.ReplaceAll(query, "business_id_req", valid.String(req.BusinessID))
	} else {

		query += `select total_buyer_new, last_period_total_buyer_new
					from ( select generate_series(0, 0 ) as index, count (*) as total_buyer_new 
						from ( select distinct contact_id as total_buyer_new
							from ecom_order o where business_id = 'req_business_id'
								and (created_at) > 'first_date'
								and (created_at) < 'last_date' 
							group by
								contact_id) temp
						where total_buyer_new not in (
							select distinct contact_id as last_period_total_buyer
							from ecom_order o
							where
								business_id = 'req_business_id'
								and (created_at) < 'first_date'
									and contact_id is not null
								group by
									contact_id)
						) temp_order
					inner join (
						select generate_series(0, 0 ) as index, count (*) as last_period_total_buyer_new
						from ( select distinct contact_id as total_buyer_new
							from ecom_order o where business_id = 'req_business_id'
								and (created_at) > 'per_first_date'
								and (created_at) < 'per_last_date' 
							group by
								contact_id) temp
						where total_buyer_new not in ( select distinct contact_id as last_period_total_buyer
							from ecom_order o
							where business_id = 'req_business_id'
								and (created_at) < 'per_first_date'
									and contact_id is not null
								group by
									contact_id)) last_temp_order on
						temp_order.index = last_temp_order.index`

		query = strings.ReplaceAll(query, "req_business_id", valid.String(req.BusinessID))
		now := time.Now().Add(time.Duration(-7) * time.Hour)
		switch req.Type {
		case utils.OPTION_FILTER_TODAY:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_WEEK:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, 0, -7).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		case utils.OPTION_FILTER_THIS_MONTH:
			query = strings.ReplaceAll(query, "per_first_date", req.StartTime.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Add(time.Duration(-7)*time.Hour).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "per_last_date", now.UTC().Add(7*time.Hour).AddDate(0, -1, 0).Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "first_date", req.StartTime.Format("2006-01-02 15:04:05"))
			query = strings.ReplaceAll(query, "last_date", now.UTC().Add(7*time.Hour).Format("2006-01-02 15:04:05"))
		default:
			return model.CountBuyerNew{}, nil
		}
	}
	rs := model.CountBuyerNew{}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE || req.Type == utils.OPTION_FILTER_LAST_MONTH || req.Type == utils.OPTION_FILTER_LAST_WEEK {
		if req.StartTime != nil && req.EndTime != nil {
			if err := r.DB.Raw(query, req.StartTime, req.EndTime, req.StartTime).Scan(&rs).Error; err != nil {
				return model.CountBuyerNew{}, nil
			}
		} else {
			return model.CountBuyerNew{}, nil
		}
	} else {
		if err := r.DB.Raw(query).Scan(&rs).Error; err != nil {
			return model.CountBuyerNew{}, nil
		}
	}
	return rs, nil
}

func (r *RepoPG) GetDetailChartAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
				select
					temp_order.index,
					to_char(temp_order.time, 'HH24:MI:SS') AS time,
					temp_order.value as value,
					temp_order_last.value as per_value
					
				from
					(
					select
							row_number() over (
						order by
							temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				left join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last
							
							on
					temp_order_last.index = temp_order.index
				order by
					temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
					select
						temp_order.index,
						to_char(temp_order.time, 'YYYY-MM-DD') as time,
						temp_order.value as value,
						temp_order_last.value as per_value
					from      
						(
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value as value
						from	
							(
							select
								date_trunc('day',  'latest_end_time'::date ) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from orders o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order
					left join (
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value
						from
							(
							select
								date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from orders o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order_last on
						temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
				select
					case
						when ('latest_end_time'::date - 'first_start_time'::date>'latest_last_end_time'::date - 'first_last_start_time'::date) then temp_order.index
						else temp_order_last.index
					end as index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value,
					temp_order_last.value as per_value
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('day', 'latest_end_time'::date) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				full outer join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('day', 'latest_last_end_time'::date ) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							and deleted_at is null
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last on
					temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		select
			row_number() over (
			order by temp.time) as index,
			to_char(temp.time, 'YYYY-MM-DD') as time,
			null as per_value,
			temp1.value as value
		from
			(
			select
				date_trunc('day',  'end_time'::date ) - interval '1 day' * n as time
			from
				generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
		left join (
			select
				DATE_TRUNC('day', updated_at + '7 hour') as time,
				business_id ,
				sum(grand_total) as value
				from orders o
			where
				business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
			group by
				DATE_TRUNC('day', updated_at + '7 hour'),
				business_id
			order by
				DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
			temp.time = temp1.time
		order by
			temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *RepoPG) GetOrderChartAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
				select
					temp_order.index,
					to_char(temp_order.time, 'HH24:MI:SS') AS time,
					temp_order.value as value,
					temp_order_last.value as per_value
					
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							orders o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				left join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							orders o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last
							
							on
					temp_order_last.index = temp_order.index
				order by
					temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
						select
						temp_order.index,
						to_char(temp_order.time, 'YYYY-MM-DD') as time,
						temp_order.value as value,
						temp_order_last.value as per_value
					from      
						(
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value as value
						from
							(
							select
								date_trunc('day',  'latest_end_time'::date ) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								count(*) as value
								from orders o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order
					left join (
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value
						from
							(
							select
								date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								count(*) as value
								from orders o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order_last on
						temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
				select
					case
						when ('latest_end_time'::date - 'first_start_time'::date>'latest_last_end_time'::date - 'first_last_start_time'::date) then temp_order.index
						else temp_order_last.index
					end as index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value,
					temp_order_last.value as per_value
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('day', 'latest_end_time'::date) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							orders o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				full outer join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('day', 'latest_last_end_time'::date ) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							orders o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							and deleted_at is null
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last on
					temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		select
			row_number() over (
			order by temp.time) as index,
			to_char(temp.time, 'YYYY-MM-DD') as time,
			null as per_value,
			temp1.value as value
		from
			(
			select
				date_trunc('day',  'end_time'::date ) - interval '1 day' * n as time
			from
				generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
		left join (
			select
				DATE_TRUNC('day', updated_at + '7 hour') as time,
				business_id ,
				count(*) as value
				from orders o
			where
				business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
			group by
				DATE_TRUNC('day', updated_at + '7 hour'),
				business_id
			order by
				DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
			temp.time = temp1.time
		order by
			temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *RepoPG) GetCustomerChartAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
			select
				temp_order.index,
				to_char(temp_order.time, 'HH24:MI:SS') as time,
				temp_order.value as value,
				temp_order_last.value as per_value
			from
				(
				select
					row_number() over (
				order by
					temp.time) as index,
					temp.time,
					sum(temp1.value) as value
				from
					(
					select
						date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
					from
						generate_series(0, 23 ) as i(n)) temp
				left join (
					select
						time,
						count(contact_id) as value
					from
						(
						select
							contact_id,
							DATE_TRUNC('hour', created_at +'7 hour' ) as time
						from
							orders o
						where
							business_id = ? and deleted_at is null
							and (created_at between ? and ?)
						group by
							contact_id,
							DATE_TRUNC('hour', created_at + '7 hour')) as tem
					group by
						time)temp1 on
					temp.time = temp1.time
				group by
					temp.time) temp_order
			left join (
				select
					row_number() over (
				order by
					temp.time) as index,
					temp.time,
					sum(temp1.value) as value
				from
					(
					select
						date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
					from
						generate_series(0, 23 ) as i(n)) temp
				left join (
					select
						time,
						count(contact_id) as value
					from
						(
						select
							contact_id,
							DATE_TRUNC('hour', created_at +'7 hour' ) as time
						from
							orders o
						where
							business_id = ? and deleted_at is null
							and (created_at between ? and ?)
						group by
							contact_id,
							DATE_TRUNC('hour', created_at + '7 hour')) as tem
					group by
						time)temp1 on
					temp.time = temp1.time
				group by
					temp.time) temp_order_last on temp_order_last.index = temp_order.index
				order by
				temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
			select
					temp_order.index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value as value,
					temp_order_last.value as per_value
			from
					(
				select
						row_number() over (
				order by
						temp.time) as index,
						temp.time,
						sum(temp1.value) as value
				from
						(
					select
							date_trunc('day', 'latest_end_time'::date ) - interval '1 day' * n as time
					from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
				left join (
					select
							time,
							count(contact_id) as value
					from
							(
						select
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour' ) as time
						from
								orders o
						where
							business_id = ?
							and deleted_at is null
							and (created_at between ? and ?)
						group by
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour')) as tem
					group by
							time)temp1 on
						temp.time = temp1.time
				group by
						temp.time) temp_order
			left join (
				select
						row_number() over (
				order by
						temp.time) as index,
						temp.time,
						sum(temp1.value) as value
				from
						(
					select
							date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
					from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
				left join (
					select
							time,
							count(contact_id) as value
					from
							(
						select
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour' ) as time
						from
								orders o
						where
							business_id = ? and deleted_at is null
							and (created_at between ? and ?)
						group by
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour')) as tem
					group by
							time)temp1 on
						temp.time = temp1.time
				group by
						temp.time) temp_order_last on
				temp_order_last.index = temp_order.index
			order by
					temp_order.time`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
			select
				case
					when (
						'latest_end_time' :: date - 'first_start_time' :: date > 'latest_last_end_time' :: date - 'first_last_start_time' :: date
					) then temp_order.index
					else temp_order_last.index
				end as index,
				to_char(temp_order.time, 'YYYY-MM-DD') as time,
				temp_order.value as value,
				temp_order_last.value as per_value
			from
				(
					select
						row_number() over (
							order by
								temp.time
						) as index,
						temp.time,
						sum(temp1.value) as value
					from
						(
							select
								date_trunc('day', 'latest_end_time' :: date) - interval '1 day' * n as time
							from
								generate_series(
									0,
									'latest_end_time' :: date - 'first_start_time' :: date
								) as i(n)
						) temp
						left join (
							select
								time,
								count(contact_id) as value
							from
								(
									select
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour') as time
									from
										orders o
									where
										business_id = ?
										and deleted_at is null
										and (
											created_at between ?
											and ?
										)
									group by
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour')
								) as tem
							group by
								time
						) temp1 on temp.time = temp1.time
					group by
						temp.time
				) temp_order full
				outer join (
					select
						row_number() over (
							order by
								temp.time
						) as index,
						temp.time,
						sum(temp1.value) as value
					from
						(
							select
								date_trunc('day', 'latest_last_end_time' :: date) - interval '1 day' * n as time
							from
								generate_series(
									0,
									'latest_last_end_time' :: date - 'first_last_start_time' :: date
								) as i(n)
						) temp
						left join (
							select
								time,
								count(contact_id) as value
							from
								(
									select
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour') as time
									from
										orders o
									where
										business_id = ?
										and (
											created_at between ?
											and ?
										)
									group by
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour')
								) as tem
							group by
								time
						) temp1 on temp.time = temp1.time
					group by
						temp.time
				) temp_order_last on temp_order_last.index = temp_order.index
			order by
				temp_order.time	`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}

	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		 		select
						row_number() over (
				order by
						temp.time) as index,
						temp.time,
						sum(temp1.value) as value,
						null as per_value
				from
						(
					select
							date_trunc('day', 'end_time'::date ) - interval '1 day' * n as time
					from
							generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
				left join (
					select
							time,
							count(contact_id) as value
					from
							(
						select
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour' ) as time
						from
								orders o
						where
							business_id = ?
							and deleted_at is null
							and (created_at between ? and ?)
						group by
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour')) as tem
					group by
							time)temp1 on
						temp.time = temp1.time
				group by
						temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *RepoPG) GetCancelChartAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
				select
					temp_order.index,
					to_char(temp_order.time, 'HH24:MI:SS') AS time,
					temp_order.value as value,
					temp_order_last.value as per_value
					
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				left join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last
							
							on
					temp_order_last.index = temp_order.index
				order by
					temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
						select
						temp_order.index,
						to_char(temp_order.time, 'YYYY-MM-DD') as time,
						temp_order.value as value,
						temp_order_last.value as per_value
					from      
						(
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value as value
						from
							(
							select
								date_trunc('day',  'latest_end_time'::date ) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from orders o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order
					left join (
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value
						from
							(
							select
								date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from orders o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order_last on
						temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
				select
					case
						when ('latest_end_time'::date - 'first_start_time'::date>'latest_last_end_time'::date - 'first_last_start_time'::date) then temp_order.index
						else temp_order_last.index
					end as index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value,
					temp_order_last.value as per_value
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('day', 'latest_end_time'::date) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				full outer join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('day', 'latest_last_end_time'::date ) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							orders o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
							and deleted_at is null
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last on
					temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		select
			row_number() over (
			order by temp.time) as index,
			to_char(temp.time, 'YYYY-MM-DD') as time,
			null as per_value,
			temp1.value as value
		from
			(
			select
				date_trunc('day',  'end_time'::date ) - interval '1 day' * n as time
			from
				generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
		left join (
			select
				DATE_TRUNC('day', updated_at + '7 hour') as time,
				business_id ,
				sum(grand_total) as value
				from orders o
			where
				business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
			group by
				DATE_TRUNC('day', updated_at + '7 hour'),
				business_id
			order by
				DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
			temp.time = temp1.time
		order by
			temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *RepoPG) GetDetailChartEcomAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
				select
					temp_order.index,
					to_char(temp_order.time, 'HH24:MI:SS') AS time,
					temp_order.value as value_ecom,
					temp_order_last.value as per_value_ecom
					
				from
					(
					select
							row_number() over (
						order by
							temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				left join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last
							
							on
					temp_order_last.index = temp_order.index
				order by
					temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
					select
						temp_order.index,
						to_char(temp_order.time, 'YYYY-MM-DD') as time,
						temp_order.value as value_ecom,
						temp_order_last.value as per_value_ecom
					from      
						(
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value as value
						from	
							(
							select
								date_trunc('day',  'latest_end_time'::date ) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from ecom_order o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order
					left join (
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value
						from
							(
							select
								date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from ecom_order o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order_last on
						temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
				select
					case
						when ('latest_end_time'::date - 'first_start_time'::date>'latest_last_end_time'::date - 'first_last_start_time'::date) then temp_order.index
						else temp_order_last.index
					end as index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value as value_ecom,
					temp_order_last.value as per_value_ecom
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('day', 'latest_end_time'::date) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				full outer join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('day', 'latest_last_end_time'::date ) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							and deleted_at is null
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last on
					temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		select
			row_number() over (
			order by temp.time) as index,
			to_char(temp.time, 'YYYY-MM-DD') as time,
			null as per_value_ecom,
			temp1.value as value_ecom
		from
			(
			select
				date_trunc('day',  'end_time'::date ) - interval '1 day' * n as time
			from
				generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
		left join (
			select
				DATE_TRUNC('day', updated_at + '7 hour') as time,
				business_id ,
				sum(grand_total) as value
				from ecom_order o
			where
				business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
			group by
				DATE_TRUNC('day', updated_at + '7 hour'),
				business_id
			order by
				DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
			temp.time = temp1.time
		order by
			temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *RepoPG) GetOrderChartEcomAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
				select
					temp_order.index,
					to_char(temp_order.time, 'HH24:MI:SS') AS time,
					temp_order.value as value_ecom,
					temp_order_last.value as per_value_ecom
					
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				left join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last
							
							on
					temp_order_last.index = temp_order.index
				order by
					temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
						select
						temp_order.index,
						to_char(temp_order.time, 'YYYY-MM-DD') as time,
						temp_order.value as value_ecom,
						temp_order_last.value as per_value_ecom
					from      
						(
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value as value
						from
							(
							select
								date_trunc('day',  'latest_end_time'::date ) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								count(*) as value
								from ecom_order o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order
					left join (
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value
						from
							(
							select
								date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								count(*) as value
								from ecom_order o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order_last on
						temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
				select
					case
						when ('latest_end_time'::date - 'first_start_time'::date>'latest_last_end_time'::date - 'first_last_start_time'::date) then temp_order.index
						else temp_order_last.index
					end as index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value as value_ecom,
					temp_order_last.value as per_value_ecom
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('day', 'latest_end_time'::date) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							ecom_order o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				full outer join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('day', 'latest_last_end_time'::date ) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							count(*) as value
						from
							ecom_order o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='complete'
							and deleted_at is null
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last on
					temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		select
			row_number() over (
			order by temp.time) as index,
			to_char(temp.time, 'YYYY-MM-DD') as time,
			null as per_value_ecom,
			temp1.value as value_ecom
		from
			(
			select
				date_trunc('day',  'end_time'::date ) - interval '1 day' * n as time
			from
				generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
		left join (
			select
				DATE_TRUNC('day', updated_at + '7 hour') as time,
				business_id ,
				count(*) as value
				from ecom_order o
			where
				business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='complete'
			group by
				DATE_TRUNC('day', updated_at + '7 hour'),
				business_id
			order by
				DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
			temp.time = temp1.time
		order by
			temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *RepoPG) GetCustomerChartEcomAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
			select
				temp_order.index,
				to_char(temp_order.time, 'HH24:MI:SS') as time,
				temp_order.value as value_ecom,
				temp_order_last.value as per_value_ecom
			from
				(
				select
					row_number() over (
				order by
					temp.time) as index,
					temp.time,
					sum(temp1.value) as value
				from
					(
					select
						date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
					from
						generate_series(0, 23 ) as i(n)) temp
				left join (
					select
						time,
						count(contact_id) as value
					from
						(
						select
							contact_id,
							DATE_TRUNC('hour', created_at +'7 hour' ) as time
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null
							and (created_at between ? and ?)
						group by
							contact_id,
							DATE_TRUNC('hour', created_at + '7 hour')) as tem
					group by
						time)temp1 on
					temp.time = temp1.time
				group by
					temp.time) temp_order
			left join (
				select
					row_number() over (
				order by
					temp.time) as index,
					temp.time,
					sum(temp1.value) as value
				from
					(
					select
						date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
					from
						generate_series(0, 23 ) as i(n)) temp
				left join (
					select
						time,
						count(contact_id) as value
					from
						(
						select
							contact_id,
							DATE_TRUNC('hour', created_at +'7 hour' ) as time
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null
							and (created_at between ? and ?)
						group by
							contact_id,
							DATE_TRUNC('hour', created_at + '7 hour')) as tem
					group by
						time)temp1 on
					temp.time = temp1.time
				group by
					temp.time) temp_order_last on temp_order_last.index = temp_order.index
				order by
				temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
			select
					temp_order.index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value as value_ecom,
					temp_order_last.value as per_value_ecom
			from
					(
				select
						row_number() over (
				order by
						temp.time) as index,
						temp.time,
						sum(temp1.value) as value
				from
						(
					select
							date_trunc('day', 'latest_end_time'::date ) - interval '1 day' * n as time
					from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
				left join (
					select
							time,
							count(contact_id) as value
					from
							(
						select
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour' ) as time
						from
								ecom_order o
						where
							business_id = ?
							and deleted_at is null
							and (created_at between ? and ?)
						group by
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour')) as tem
					group by
							time)temp1 on
						temp.time = temp1.time
				group by
						temp.time) temp_order
			left join (
				select
						row_number() over (
				order by
						temp.time) as index,
						temp.time,
						sum(temp1.value) as value
				from
						(
					select
							date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
					from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
				left join (
					select
							time,
							count(contact_id) as value
					from
							(
						select
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour' ) as time
						from
								ecom_order o
						where
							business_id = ? and deleted_at is null
							and (created_at between ? and ?)
						group by
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour')) as tem
					group by
							time)temp1 on
						temp.time = temp1.time
				group by
						temp.time) temp_order_last on
				temp_order_last.index = temp_order.index
			order by
					temp_order.time`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
			select
				case
					when (
						'latest_end_time' :: date - 'first_start_time' :: date > 'latest_last_end_time' :: date - 'first_last_start_time' :: date
					) then temp_order.index
					else temp_order_last.index
				end as index,
				to_char(temp_order.time, 'YYYY-MM-DD') as time,
				temp_order.value as value_ecom,
				temp_order_last.value as per_value_ecom
			from
				(
					select
						row_number() over (
							order by
								temp.time
						) as index,
						temp.time,
						sum(temp1.value) as value
					from
						(
							select
								date_trunc('day', 'latest_end_time' :: date) - interval '1 day' * n as time
							from
								generate_series(
									0,
									'latest_end_time' :: date - 'first_start_time' :: date
								) as i(n)
						) temp
						left join (
							select
								time,
								count(contact_id) as value
							from
								(
									select
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour') as time
									from
										ecom_order o
									where
										business_id = ?
										and deleted_at is null
										and (
											created_at between ?
											and ?
										)
									group by
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour')
								) as tem
							group by
								time
						) temp1 on temp.time = temp1.time
					group by
						temp.time
				) temp_order full
				outer join (
					select
						row_number() over (
							order by
								temp.time
						) as index,
						temp.time,
						sum(temp1.value) as value
					from
						(
							select
								date_trunc('day', 'latest_last_end_time' :: date) - interval '1 day' * n as time
							from
								generate_series(
									0,
									'latest_last_end_time' :: date - 'first_last_start_time' :: date
								) as i(n)
						) temp
						left join (
							select
								time,
								count(contact_id) as value
							from
								(
									select
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour') as time
									from
										ecom_order o
									where
										business_id = ?
										and (
											created_at between ?
											and ?
										)
									group by
										contact_id,
										DATE_TRUNC('day', created_at + '7 hour')
								) as tem
							group by
								time
						) temp1 on temp.time = temp1.time
					group by
						temp.time
				) temp_order_last on temp_order_last.index = temp_order.index
			order by
				temp_order.time	`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}

	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		 		select
						row_number() over (
				order by
						temp.time) as index,
						temp.time,
						sum(temp1.value) as value_ecom,
						null as per_value_ecom
				from
						(
					select
							date_trunc('day', 'end_time'::date ) - interval '1 day' * n as time
					from
							generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
				left join (
					select
							time,
							count(contact_id) as value
					from
							(
						select
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour' ) as time
						from
								ecom_order o
						where
							business_id = ?
							and deleted_at is null
							and (created_at between ? and ?)
						group by
								contact_id,
								DATE_TRUNC('day', created_at + '7 hour')) as tem
					group by
							time)temp1 on
						temp.time = temp1.time
				group by
						temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}

func (r *RepoPG) GetCancelChartEcomAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (rs []model.ChartDataDetail, err error) {
	query := ""

	if utils.OPTION_FILTER_TODAY == req.Type {
		query = `
				select
					temp_order.index,
					to_char(temp_order.time, 'HH24:MI:SS') AS time,
					temp_order.value as value_ecom,
					temp_order_last.value as per_value_ecom
					
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('hour', 'day_to_day'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				left join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('hour', 'day_yesterday'::date) + interval '1 hour' * n as time
						from
							generate_series(0, 23 ) as i(n)) temp
					left join (
						select
							DATE_TRUNC('hour', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
						group by
							DATE_TRUNC('hour', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('hour', updated_at + '7 hour')) as temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last
							
							on
					temp_order_last.index = temp_order.index
				order by
					temp_order.time`

		query = strings.ReplaceAll(query, "day_to_day", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "day_yesterday", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if utils.OPTION_FILTER_THIS_WEEK == req.Type || utils.OPTION_FILTER_LAST_WEEK == req.Type {
		query = `
						select
						temp_order.index,
						to_char(temp_order.time, 'YYYY-MM-DD') as time,
						temp_order.value as value_ecom,
						temp_order_last.value as per_value_ecom
					from      
						(
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value as value
						from
							(
							select
								date_trunc('day',  'latest_end_time'::date ) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_end_time'::date - 'first_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from ecom_order o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order
					left join (
						select
							row_number() over (
							order by temp.time) as index,
							temp.time as time,
							temp1.business_id,
							temp1.value
						from
							(
							select
								date_trunc('day', 'latest_last_end_time'::date) - interval '1 day' * n as time
							from
								generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date ) as i(n)) temp
						left join (
							select
								DATE_TRUNC('day', updated_at + '7 hour') as time,
								business_id ,
								sum(grand_total) as value
								from ecom_order o
							where
								business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
							group by
								DATE_TRUNC('day', updated_at + '7 hour'),
								business_id
							order by
								DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
							temp.time = temp1.time
						order by
							temp.time) temp_order_last on
						temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfWeek(req.StartTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))

	}
	if req.Type == utils.OPTION_FILTER_THIS_MONTH || req.Type == utils.OPTION_FILTER_LAST_MONTH {
		query = `
				select
					case
						when ('latest_end_time'::date - 'first_start_time'::date>'latest_last_end_time'::date - 'first_last_start_time'::date) then temp_order.index
						else temp_order_last.index
					end as index,
					to_char(temp_order.time, 'YYYY-MM-DD') as time,
					temp_order.value as value_ecom,
					temp_order_last.value as per_value_ecom
				from
					(
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value as value
					from
						(
						select
							date_trunc('day', 'latest_end_time'::date) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_end_time'::date - 'first_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order
				full outer join (
					select
						row_number() over (
					order by
						temp.time) as index,
						temp.time as time,
						temp1.business_id,
						temp1.value
					from
						(
						select
							date_trunc('day', 'latest_last_end_time'::date ) - interval '1 day' * n as time
						from
							generate_series(0, 'latest_last_end_time'::date - 'first_last_start_time'::date) as i(n)) temp
					left join (
						select
							DATE_TRUNC('day', updated_at + '7 hour') as time,
							business_id ,
							sum(grand_total) as value
						from
							ecom_order o
						where
							business_id = ?  and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
							and deleted_at is null
						group by
							DATE_TRUNC('day', updated_at + '7 hour'),
							business_id
						order by
							DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
						temp.time = temp1.time
					order by
						temp.time) temp_order_last on
					temp_order.index = temp_order_last.index`

		query = strings.ReplaceAll(query, "first_start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_end_time", utils.EndOfMonth(req.EndTime.UTC().Add(7*time.Hour)).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "first_last_start_time", req.StartTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "latest_last_end_time", req.EndTimeSamePeriod.UTC().Add(7*time.Hour).Format("2006-01-02"))
	}
	if req.Type == utils.OPTION_FILTER_CUSTOM_RANGE {
		query = `
		select
			row_number() over (
			order by temp.time) as index,
			to_char(temp.time, 'YYYY-MM-DD') as time,
			null as per_value_ecom,
			temp1.value as value_ecom
		from
			(
			select
				date_trunc('day',  'end_time'::date ) - interval '1 day' * n as time
			from
				generate_series(0, 'end_time'::date - 'start_time'::date ) as i(n)) temp
		left join (
			select
				DATE_TRUNC('day', updated_at + '7 hour') as time,
				business_id ,
				sum(grand_total) as value
				from ecom_order o
			where
				business_id = ? and deleted_at is null and (updated_at between ? and ?) and state ='cancel'
			group by
				DATE_TRUNC('day', updated_at + '7 hour'),
				business_id
			order by
				DATE_TRUNC('day', updated_at + '7 hour')) temp1 on
			temp.time = temp1.time
		order by
			temp.time`

		query = strings.ReplaceAll(query, "end_time", req.EndTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		query = strings.ReplaceAll(query, "start_time", req.StartTime.UTC().Add(7*time.Hour).Format("2006-01-02"))
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&rs).Error; err != nil {
			return nil, err
		}
		return rs, nil
	}
	if err := r.DB.Raw(query, req.BusinessID, req.StartTimeSamePeriod, req.EndTime, req.BusinessID, req.StartTimeSamePeriod, req.EndTime).Scan(&rs).Error; err != nil {
		return nil, err
	}
	return rs, nil
}
