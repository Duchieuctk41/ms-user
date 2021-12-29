package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/utils"
	"strings"

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

func (r *RepoPG) OverviewCost(ctx context.Context, req model.OrverviewPandLRequest, overviewPandL model.OverviewPandLResponse, tx *gorm.DB) (model.OverviewPandLResponse, error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}
	query := ""
	query += `select
					sum(oi.historical_cost * oi.quantity) as cost_total
				from
					order_item oi
				inner join orders o on
					oi.order_id = o.id
					and o.state = 'complete'
					and o.business_id = ? `
	if req.StartTime != nil && req.EndTime != nil {
		query += " AND updated_at BETWEEN ? AND ? "
	}
	//rs := model.OverviewPandLResponse{}
	if req.StartTime != nil && req.EndTime != nil {
		if err := r.DB.Raw(query, req.BusinessID, req.StartTime, req.EndTime).Scan(&overviewPandL).Error; err != nil {
			return model.OverviewPandLResponse{}, err
		}
	} else {
		if err := r.DB.Raw(query, req.BusinessID).Scan(&overviewPandL).Error; err != nil {
			return model.OverviewPandLResponse{}, err
		}
	}

	return overviewPandL, nil
}

func (r *RepoPG) GetListProfitAndLoss(ctx context.Context, req model.ProfitAndLossRequest, tx *gorm.DB) (rs model.GetListProfitAndLossResponse, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}
	page := r.GetPage(req.Page)
	pageSize := r.GetPageSize(req.PageSize)

	rs = model.GetListProfitAndLossResponse{}
	if req.BusinessID != nil {
		tx = tx.Select(utils.RemoveSpace(`orders.business_id,
						order_item.sku_id ,
						order_item.product_name ,
						order_item.sku_name,
						sum(order_item.quantity) total_quantity,
						sum(order_item.price * order_item.quantity) as price,
						sum(order_item.historical_cost * order_item.quantity) as historical_cost,
						sum(order_item.price * order_item.quantity)-sum(order_item.historical_cost * order_item.quantity) as profit`)).
			Table("order_item").
			Joins(`inner join orders on
						order_item.order_id = orders.id
						and orders.state = 'complete'
						and orders.business_id = ? `, req.BusinessID).
			Group("business_id, sku_id ,product_name ,sku_name")
	}

	var total int64

	switch req.Sort {
	case "profit desc":
		tx = tx.Order("profit desc")
	case "profit asc":
		tx = tx.Order("profit asc")
	case "quantity desc":
		tx = tx.Order("quantity desc")
	case "quantity asc":
		tx = tx.Order("quantity asc")
	default:
		tx = tx.Order("profit desc")
	}

	if err := tx.Limit(pageSize).Offset(r.GetOffset(page, pageSize)).Find(&rs.Data).Error; err != nil {
		return rs, err
	}
	countQuery := "SELECT count(*) FROM order_item inner join orders on order_item.order_id = orders.id and orders.state = 'complete' and orders.business_id = 'BusinessID'  GROUP BY business_id, sku_id ,product_name ,sku_name"
	// TODO: Pagination
	countQuery = strings.ReplaceAll(countQuery, "BusinessID", *req.BusinessID)
	if rs.Meta, err = r.GetPaginationInfo(countQuery, tx, int(total), page, pageSize); err != nil {
		return rs, err
	}

	queryTotal := ""
	queryTotal += utils.RemoveSpace(`select
				sum(oi.quantity) as total_quantity,
					sum(oi.price * oi.quantity) - sum(oi.historical_cost * oi.quantity) as total_profit
				from
					order_item oi
				inner join orders o on
					oi.order_id = o.id
					and o.state = 'complete'
					and o.business_id = ? `)
	if req.StartTime != nil && req.EndTime != nil {
		queryTotal += " AND updated_at BETWEEN ? AND ? "
	}

	totalProfit := model.TotalProfitAndLossResponse{}
	if req.StartTime != nil && req.EndTime != nil {
		//dateFromStr, dateToStr := utils.ConvertTimestampVN(req.DateFrom, req.DateTo)
		if err := r.DB.Raw(queryTotal, req.BusinessID, req.StartTime, req.EndTime).Scan(&totalProfit).Error; err != nil {
			return rs, err
		}
	} else {
		if err := r.DB.Raw(queryTotal, req.BusinessID).Scan(&totalProfit).Error; err != nil {
			return rs, err
		}
	}
	rs.Meta["total_profit"] = totalProfit.TotalProfit
	rs.Meta["total_quantity"] = totalProfit.TotalQuantity

	return rs, nil
}
