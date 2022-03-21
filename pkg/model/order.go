package model

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Order struct {
	BaseModel
	BusinessID          uuid.UUID                     `json:"business_id" sql:"index" gorm:"column:business_id;not null;" valid:"Required" scheme:"business_id"`
	ContactID           uuid.UUID                     `json:"contact_id" sql:"index" gorm:"column:contact_id;not null;"`
	OrderNumber         string                        `json:"order_number" sql:"index" gorm:"column:order_number;not null;"`
	PromotionCode       string                        `json:"promotion_code" gorm:"column:promotion_code;null;"`
	OrderedGrandTotal   float64                       `json:"ordered_grand_total" gorm:"column:ordered_grand_total"`
	PromotionDiscount   float64                       `json:"promotion_discount" gorm:"promotion_discount"`
	DeliveryFee         float64                       `json:"delivery_fee" gorm:"column:delivery_fee"`
	GrandTotal          float64                       `json:"grand_total" gorm:"grand_total"`
	State               string                        `json:"state" sql:"index" gorm:"column:state;not null;"`
	PaymentMethod       string                        `json:"payment_method" sql:"index" gorm:"column:payment_method;"`
	Note                string                        `json:"note" gorm:"column:note;null;"`
	BuyerInfo           postgres.Jsonb                `json:"buyer_info" gorm:"null"`
	BuyerId             *uuid.UUID                    `json:"buyer_id" sql:"index" gorm:"column:buyer_id;type:uuid"`
	DeliveryMethod      string                        `json:"delivery_method" sql:"index" gorm:"column:delivery_method;"`
	OrderItem           []OrderItem                   `json:"order_item" gorm:"foreignkey:order_id;association_foreignkey:id;" `
	CreateMethod        string                        `json:"create_method" sql:"index" gorm:"create_method;default:'buyer'"`
	Email               string                        `json:"email" sql:"index" gorm:"type:varchar(500)"`
	OtherDiscount       float64                       `json:"other_discount" gorm:""`
	IsPrinted           bool                          `json:"is_printed" sql:"index" gorm:"column:is_printed;default:false"`
	DebtAmount          float64                       `json:"debt_amount" gorm:"column:debt_amount"`
	AmountPaid          float64                       `json:"amount_paid" gorm:"column:amount_paid;null"`
	PaymentOrderHistory []PaymentOrderHistoryResponse `json:"payment_order_history" gorm:"foreignkey:order_id;association_foreignkey:id"`
	Images              pq.StringArray                `json:"images" gorm:"type:varchar(500)[]"`
}

func GenerateRandomString(n int) string {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return ""
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret)
}

func (d *Order) GenRandomKey(tx *gorm.DB) string {
	res := GenerateRandomString(9)
	if err := tx.Model(&Order{}).Where("order_number = ?", res).First(&Order{}).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return res
		}
	}
	return d.GenRandomKey(tx)
}

func (u *Order) BeforeCreate(tx *gorm.DB) (err error) {
	u.OrderNumber = u.GenRandomKey(tx)
	return
}

func (Order) TableName() string {
	return "orders"
}

// Define your request body here
type OrderBody struct {
	UserID            uuid.UUID      `json:"user_id"`
	ContactID         *uuid.UUID     `json:"contact_id,omitempty"`
	BusinessID        *uuid.UUID     `json:"business_id" schema:"business_id"`
	PromotionCode     string         `json:"promotion_code"`
	PromotionDiscount float64        `json:"promotion_discount"`
	OrderedGrandTotal float64        `json:"ordered_grand_total"`
	GrandTotal        float64        `json:"grand_total"`
	State             string         `json:"state"`
	PaymentMethod     string         `json:"payment_method"`
	Note              string         `json:"note"`
	ListOrderItem     []OrderItem    `json:"list_order_item"`
	BuyerInfo         *BuyerInfo     `json:"buyer_info"`
	DeliveryFee       float64        `json:"delivery_fee"`
	DeliveryMethod    *string        `json:"delivery_method" valid:"Required" schema:"delivery_method"`
	PaymentSourceID   *uuid.UUID     `json:"payment_source_id"`
	PaymentSourceName *string        `json:"payment_source_name"`
	CreateMethod      string         `json:"create_method" valid:"Required"`
	OtherDiscount     float64        `json:"other_discount"`
	Email             string         `json:"email"`
	ListProductFast   []Product      `json:"list_product_fast"`
	Debit             *Debit         `json:"debit,omitempty"`
	BuyerReceived     bool           `json:"buyer_received"`
	Images            pq.StringArray `json:"images" gorm:"type:varchar(500)[]"`
	//BuyerId           *uuid.UUID  `json:"buyer_id"`

}

type BuyerInfo struct {
	PhoneNumber string  `json:"phone_number" valid:"Required"`
	Name        string  `json:"name" valid:"Required"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type Debit struct {
	BuyerPay *float64 `json:"buyer_pay"`
	Note     *string  `json:"note"`
}

type RevenueBusiness struct {
	SumGrandTotal float64 `json:"sum_grand_total"`
}

// Define your request param here
// Remember to user scheme tag
type OrderParam struct {
	BusinessID     string     `json:"business_id" form:"business_id"`
	ContactID      string     `json:"contact_id" form:"contact_id"`
	PromotionCode  string     `json:"promotion_code" form:"promotion_code"`
	State          string     `json:"state" form:"state"`
	OrderNumber    string     `json:"order_number" form:"order_number"`
	PaymentMethod  string     `json:"payment_method" form:"payment_method"`
	Note           string     `json:"note" form:"note"`
	PageSize       int        `json:"page_size" form:"page_size"`
	Size           int        `json:"size" form:"size"`
	Page           int        `json:"page" form:"page"`
	Sort           string     `json:"sort" form:"sort"`
	BuyerID        string     `json:"buyer_id" form:"buyer_id"`
	DateFrom       *time.Time `json:"date_from" form:"date_from"`
	DateTo         *time.Time `json:"date_to" form:"date_to"`
	Search         string     `json:"search" form:"search"`
	SellerID       string     `json:"seller_id" form:"seller_id"`
	UserRole       string     `json:"user_role" form:"user_role"`
	UserCallAPI    string     `json:"user_call_api" form:"user_call_api"`
	DeliveryMethod *string    `json:"delivery_method" form:"delivery_method"`
	IsPrinted      *bool      `json:"is_printed" form:"is_printed"`
}

type OrderUpdateBody struct {
	ID                *uuid.UUID      `json:"id"`
	BusinessID        *uuid.UUID      `json:"business_id" schema:"business_id"`
	PromotionCode     *string         `json:"promotion_code"`
	PromotionDiscount *float64        `json:"promotion_discount"`
	OrderedGrandTotal *float64        `json:"ordered_grand_total" gorm:"column:ordered_grand_total"`
	GrandTotal        *float64        `json:"grand_total" gorm:"grand_total"`
	State             *string         `json:"state"`
	PaymentMethod     *string         `json:"payment_method"`
	PaymentSourceID   *uuid.UUID      `json:"payment_source_id"`
	PaymentSourceName *string         `json:"payment_source_name"`
	Note              *string         `json:"note"`
	BuyerID           *uuid.UUID      `json:"buyer_id"`
	BuyerInfo         *BuyerInfo      `json:"buyer_info"`
	UpdaterID         *uuid.UUID      `json:"updater_id,omitempty"`
	OtherDiscount     *float64        `json:"other_discount"`
	Email             *string         `json:"email,omitempty"`
	ListOrderItem     []OrderItem     `json:"list_order_item,omitempty"`
	Debit             *Debit          `json:"debit,omitempty"`
	Images            *pq.StringArray `json:"images" gorm:"type:varchar(500)[]"`
}

type OrverviewPandLRequest struct {
	UserRole    string     `json:"user_role"`
	UserCallAPI uuid.UUID  `json:"user_call_api"`
	StartTime   *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" form:"end_time"`
	BusinessID  *string    `json:"business_id,omitempty" form:"business_id" valid:"Required"`
}

type OrverviewOrderRequest struct {
	UserRole    string     `json:"user_role"`
	UserCallAPI uuid.UUID  `json:"user_call_api"`
	StartTime   *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" form:"end_time"`
	BusinessID  *string    `json:"business_id,omitempty" form:"business_id" valid:"Required"`
}

type UpdateDetailOrderRequest struct {
	BusinessID        *uuid.UUID      `json:"business_id"`
	ID                *uuid.UUID      `json:"id"`
	PromotionDiscount *float64        `json:"promotion_discount,omitempty" valid:"Required"`
	OrderedGrandTotal *float64        `json:"ordered_grand_total,omitempty" valid:"Required"`
	GrandTotal        *float64        `json:"grand_total,omitempty" valid:"Required"`
	DeliveryFee       *float64        `json:"delivery_fee,omitempty"`    // set valid:"Required" when APP done new version store
	DeliveryMethod    *string         `json:"delivery_method,omitempty"` // set valid:"Required" when APP done new version store
	Note              *string         `json:"note"`
	UpdaterID         *uuid.UUID      `json:"updater_id,omitempty"`
	UserRole          *string         `json:"user_role"`
	OtherDiscount     *float64        `json:"other_discount,omitempty" valid:"Required"`
	ListOrderItem     []OrderItem     `json:"list_order_item,omitempty" valid:"Required"`
	BuyerInfo         *BuyerInfo      `json:"buyer_info"`
	Images            *pq.StringArray `json:"images" gorm:"type:varchar(500)[]"`
}

type ListOrderResponse struct {
	Data []Order                `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

type GetCompleteOrdersResponse struct {
	Count     int     `json:"count"`
	SumAmount float64 `json:"sum_amount"`
}

type RevenueBusinessParam struct {
	BusinessID string     `json:"business_id" form:"business_id"`
	ContactID  string     `json:"contact_id" form:"contact_id"`
	DateFrom   *time.Time `json:"date_from" form:"date_from"`
	DateTo     *time.Time `json:"date_to" form:"date_to"`
}

type CountOrderState struct {
	CountWaitingConfirm int     `json:"count_waiting_confirm"`
	CountDelivering     int     `json:"count_delivering"`
	CountComplete       int     `json:"count_complete"`
	CountCancel         int     `json:"count_cancel"`
	Revenue             float64 `json:"revenue"`
	Profit              float64 `json:"profit"`
}
type OrderByContactParam struct {
	PageSize   int        `json:"page_size" form:"page_size"`
	Page       int        `json:"page" form:"page"`
	StartTime  *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty" form:"end_time"`
	ContactID  string     `json:"contact_id" form:"contact_id"`
	BusinessID string     `json:"business_id" form:"business_id"`
}

type ExportOrderReportRequest struct {
	BusinessID     *uuid.UUID `json:"business_id" valid:"Required"`
	UserID         uuid.UUID  `json:"user_id"`
	UserRole       string     `json:"user_role"`
	StartTime      *time.Time `json:"start_time"`
	EndTime        *time.Time `json:"end_time"`
	State          *string    `json:"state"`
	PaymentMethod  *string    `json:"payment_method"`
	DeliveryMethod *string    `json:"delivery_method"`
}

type ContactDelivering struct {
	ContactID   uuid.UUID `json:"contact_id"`
	Count       int       `json:"count"`
	ContactInfo *Contact  `json:"contact_info"`
}

type TotalContactDelivery struct {
	Count int `json:"count"`
}

type ContactDeliveringResponse struct {
	Data []ContactDelivering    `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

type GetOneOrderRequest struct {
	ID         *string   `json:"id"`
	UserRole   string    `json:"user_role"`
	UserID     uuid.UUID `json:"user_id"`
	BusinessID string    `json:"business_id" form:"business_id"`
	BuyerID    string    `json:"buyer_id"  form:"buyer_id"`
}

type CountQuantityInOrderRequest struct {
	BusinessID uuid.UUID `json:"business_id" valid:"Required"`
	SkuID      uuid.UUID `json:"sku_id" valid:"Required"`
	States     []string  `json:"states" valid:"Required"`
}

type CountQuantityInOrderResponse struct {
	Sum float64 `json:"sum"`
}

type GetTotalOrderByBusinessRequest struct {
	BusinessID  string     `json:"business_id" form:"business_id"`
	ContactID   string     `json:"contact_id" form:"contact_id"`
	StartTime   *time.Time `json:"start_time" form:"start_time"`
	EndTime     *time.Time `json:"end_time" form:"end_time"`
	UserRole    string     `json:"user_role"`
	UserCallAPI uuid.UUID  `json:"user_call_api"`
}
type GetTotalOrderByBusinessResponse struct {
	ContactID          uuid.UUID `json:"contact_id" gorm:"null"`
	TotalQuantityOrder int       `json:"total_quantity_order" gorm:"null"`
	TotalAmountOrder   float64   `json:"total_amount_order" gorm:"null"`
}

type OrderBuyerResponse struct {
	//BaseModel
	ID                  uuid.UUID                     `json:"id"`
	BusinessID          uuid.UUID                     `json:"business_id"`
	ContactID           uuid.UUID                     `json:"contact_id"`
	OrderNumber         string                        `json:"order_number"`
	PromotionCode       string                        `json:"promotion_code"`
	OrderedGrandTotal   float64                       `json:"ordered_grand_total"`
	PromotionDiscount   float64                       `json:"promotion_discount"`
	DeliveryFee         float64                       `json:"delivery_fee"`
	GrandTotal          float64                       `json:"grand_total"`
	State               string                        `json:"state"`
	PaymentMethod       string                        `json:"payment_method"`
	Note                string                        `json:"note"`
	BuyerInfo           postgres.Jsonb                `json:"buyer_info"`
	BuyerId             *uuid.UUID                    `json:"buyer_id"`
	DeliveryMethod      string                        `json:"delivery_method"`
	OrderItem           []OrderItemBuyerResponse      `json:"order_item" gorm:"foreignkey:order_id;association_foreignkey:id;" `
	CreateMethod        string                        `json:"create_method"`
	Email               string                        `json:"email" sql:"index"`
	OtherDiscount       float64                       `json:"other_discount"`
	IsPrinted           bool                          `json:"is_printed"`
	DebtAmount          float64                       `json:"debt_amount"`
	AmountPaid          float64                       `json:"amount_paid"`
	PaymentOrderHistory []PaymentOrderHistoryResponse `json:"payment_order_history" gorm:"foreignkey:order_id;association_foreignkey:id;"`
}

type OrderTotal struct {
	RevenueTotal             float64 `json:"revenue_total"`
	OrderCompleteTotal       float64 `json:"order_complete_total"`
	OrderCancelTotal         float64 `json:"order_cancel_total"`
	OrderDeliveringTotal     float64 `json:"order_delivering_total"`
	OrderWaitingConfirmTotal float64 `json:"order_waiting_confirm_total"`
}

type CostTotal struct {
	CostTotal float64 `json:"cost_total"`
}

type OverviewOrderResponse struct {
	RevenueTotal             float64 `json:"revenue_total"`
	ProfitTotal              float64 `json:"profit_total"`
	OrderCompleteTotal       float64 `json:"order_complete_total"`
	OrderCancelTotal         float64 `json:"order_cancel_total"`
	OrderDeliveringTotal     float64 `json:"order_delivering_total"`
	OrderWaitingConfirmTotal float64 `json:"order_waiting_confirm_total"`
}

type OrverviewRequest struct {
	UserRole    string     `json:"user_role"`
	UserCallAPI uuid.UUID  `json:"user_call_api"`
	StartTime   *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty" form:"end_time"`
	BusinessID  *string    `json:"business_id,omitempty" form:"business_id" valid:"Required"`
}

type OrderItemAnalytic struct {
	SkuID         uuid.UUID `json:"sku_id"`
	ProductName   string    `json:"product_name"`
	SkuName       string    `json:"sku_name"`
	TotalAmount   float64   `json:"total_amount"`
	TotalQuantity float64   `json:"total_quantity"`
}

type GetOrderRevenueAnalyticsParam struct {
	BusinessID string     `json:"business_id" form:"business_id"`
	StartTime  *time.Time `json:"start_time,omitempty" form:"start_time"`
	EndTime    *time.Time `json:"end_time,omitempty" form:"end_time"`
	PageSize   int        `json:"page_size" form:"page_size"`
	Page       int        `json:"page" form:"page"`
	Sort       string     `json:"sort" form:"sort"`
}

type ListOrderRevenueAnalyticsResponse struct {
	Data []OrderItemAnalytic    `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

type DataSell struct {
	OnlineSell  float64 `json:"online_sell"`
	OfflineSell float64 `json:"offline_sell"`
	Ecommerce   float64 `json:"ecommerce,omitempty"`
}

type ResponseAnalytics struct {
	Id               int       `json:"id"`
	Type             string    `json:"type"`
	Amount           float64   `json:"amount"`
	LastPeriodAmount float64   `json:"last_period_amount"`
	Data             *DataSell `json:"data,omitempty"`
}

type AnalyticsResponse struct {
	Data  interface{} `json:"data"`
	Chart interface{} `json:"chart"`
}
type GetNumberDeliveringParam struct {
	BusinessID string `json:"business_id" form:"business_id" valid:"Required"`
	ContactIDs string `json:"contact_ids" form:"contact_ids" valid:"Required"`
	State      string `json:"state" form:"state"`
}
