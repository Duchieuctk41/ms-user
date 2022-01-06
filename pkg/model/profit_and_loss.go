package model

import (
	"time"

	"github.com/google/uuid"
)

type ProfitAndLossRequest struct {
	UserRole    string     `json:"user_role"`
	UserCallAPI uuid.UUID  `json:"user_call_api"`
	StartTime   *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" form:"end_time"`
	BusinessID  *string    `json:"business_id,omitempty" form:"business_id" valid:"Required"`
	Page        int        `json:"page" form:"page"`
	PageSize    int        `json:"page_size" form:"page_size"`
	Sort        string     `json:"sort" form:"sort"`
}

type ProfitAndLossResponse struct {
	BusinessID          uuid.UUID `json:"business_id"`
	SkuID               uuid.UUID `json:"sku_id"`
	ProductName         string    `json:"product_name"`
	SkuName             string    `json:"sku_name"`
	TotalQuantity       float64   `json:"total_quantity"`
	TotalPrice          float64   `json:"total_price"`
	TotalHistoricalCost float64   `json:"total_historical_cost"`
	Profit              float64   `json:"profit"`
}

type GetListProfitAndLossResponse struct {
	Data []ProfitAndLossResponse `json:"data"`
	Meta map[string]interface{}  `json:"meta"`
}

type TotalProfitAndLossResponse struct {
	TotalProfit   float64 `json:"total_profit"`
	TotalQuantity float64 `json:"total_quantity"`
}

type OverviewPandLResponse struct {
	SumGrandTotal float64     `json:"sum_grand_total"`
	CostTotal     float64     `json:"cost_total"`
	ProfitTotal   float64     `json:"profit_total"`
	DetailSales   DetailSales `json:"detail_sales"`
}

type DetailSales struct {
	SumGrandTotal        float64 `json:"sum_grand_total"`
	SumOrderedGrandTotal float64 `json:"sum_ordered_grand_total"`
	SumPromotionDiscount float64 `json:"sum_promotion_discount"`
	SumDeliveryFee       float64 `json:"sum_delivery_fee"`
	SumOtherDiscount     float64 `json:"sum_other_discount"`
}
