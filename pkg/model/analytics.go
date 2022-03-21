package model

import (
	"time"
)

type GetDailyVisitAnalyticsParam struct {
	BusinessID *string `json:"business_id" form:"business_id"`
}

type GetDailyReportResponse struct {
	Domain         string      `json:"domain"`
	EventDate      string      `json:"event_date"`
	UserOnairCount int         `json:"user_onair_count"`
	UserOnairList  []string    `json:"user_onair_list"`
	TotalUserCount interface{} `json:"total_user_count"`
	Chart          interface{} `json:"chart"`
	CustomerOnline []User      `json:"customer_online"`
}

type GetOrderAnalyticsRequest struct {
	BusinessID             *string    `json:"business_id" form:"business_id" valid:"Required"`
	StartTime              *time.Time `json:"start_time" form:"start_time" valid:"Required"`
	EndTime                *time.Time `json:"end_time" form:"end_time" valid:"Required"`
	Type                   string     `json:"type" form:"type"`
	Option                 string     `json:"option" form:"option"`
	Ecom                   string     `json:"ecom" form:"ecom"`
	StartTimeSamePeriod    *time.Time `json:"start_time_same_period,omitempty"`
	EndTimeSamePeriod      *time.Time `json:"end_time_same_period,omitempty"`
	EndTimeTotalSamePeriod *time.Time
	Domain                 string `json:"domain" form:"domain"`
}

type ChartDataDetail struct {
	Index        *int     `json:"index"`
	Time         *string  `json:"time"`
	Value        *float64 `json:"value"`
	PerValue     *float64 `json:"per_value"`
	ValueEcom    *float64 `json:"value_ecom"`
	PerValueEcom *float64 `json:"per_value_ecom"`
}

type CountOrderAnalytics struct {
	TotalRevenue            float64 `json:"total_revenue"`
	LastPeriodTotalRevenue  float64 `json:"last_period_total_revenue"`
	TotalCancel             float64 `json:"total_cancel"`
	LastPeriodTotalCancel   float64 `json:"last_period_total_cancel"`
	CountRevenue            int     `json:"count_revenue"`
	LastPeriodCountRevenue  int     `json:"last_period_count_revenue"`
	CountCancel             int     `json:"count_cancel"`
	LastPeriodCountCancel   int     `json:"last_period_count_cancel"`
	TotalBuyer              int     `json:"total_buyer"`
	LastPeriodTotalBuyer    int     `json:"last_period_total_buyer"`
	TotalBuyerNew           int     `json:"total_buyer_new"`
	LastPeriodTotalBuyerNew int     `json:"last_period_total_buyer_new"`
}

type CountBuyer struct {
	TotalBuyer           int `json:"total_buyer"`
	LastPeriodTotalBuyer int `json:"last_period_total_buyer"`
}

type CountBuyerNew struct {
	TotalBuyerNew           int `json:"total_buyer_new"`
	LastPeriodTotalBuyerNew int `json:"last_period_total_buyer_new"`
}
