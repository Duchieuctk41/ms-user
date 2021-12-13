package model

import (
	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
	"net/http"
	"time"
)

type Order struct {
	BaseModel
	BusinessId        uuid.UUID      `json:"business_id" sql:"index" gorm:"column:business_id;not null;" valid:"Required" scheme:"business_id"`
	ContactId         uuid.UUID      `json:"contact_id" sql:"index" gorm:"column:contact_id;not null;"`
	OrderNumber       string         `json:"order_number" sql:"index" gorm:"column:order_number;not null;"`
	PromotionCode     string         `json:"promotion_code" gorm:"column:promotion_code;null;"`
	OrderedGrandTotal float64        `json:"ordered_grand_total" gorm:"column:ordered_grand_total"`
	PromotionDiscount float64        `json:"promotion_discount" gorm:"promotion_discount"`
	DeliveryFee       float64        `json:"delivery_fee" gorm:"column:delivery_fee"`
	GrandTotal        float64        `json:"grand_total" gorm:"grand_total"`
	State             string         `json:"state" sql:"index" gorm:"column:state;not null;"`
	PaymentMethod     string         `json:"payment_method" sql:"index" gorm:"column:payment_method;"`
	Note              string         `json:"note" gorm:"column:note;null;"`
	BuyerInfo         postgres.Jsonb `json:"buyer_info" gorm:"null"`
	BuyerId           *uuid.UUID     `json:"buyer_id" sql:"index" gorm:"column:buyer_id;type:uuid"`
	DeliveryMethod    string         `json:"delivery_method" sql:"index" gorm:"column:delivery_method;"`
	OrderItem         []OrderItem    `json:"order_item" gorm:"foreignkey:order_id;association_foreignkey:id" `
	CreateMethod      string         `json:"create_method" sql:"index" gorm:"create_method;default:'buyer'"`
	Email             string         `json:"email" sql:"index" gorm:"type:varchar(500)"`
	OtherDiscount     float64        `json:"other_discount" gorm:""`
	IsPrinted         bool           `json:"is_printed" sql:"index" gorm:"column:is_printed;default:false"`
}

func (Order) TableName() string {
	return "orders"
}

// Define your request body here
type OrderBody struct {
	UserId            uuid.UUID   `json:"user_id"`
	ContactID         *uuid.UUID  `json:"contact_id,omitempty"`
	BusinessId        *uuid.UUID  `json:"business_id" schema:"business_id"`
	PromotionCode     string      `json:"promotion_code"`
	PromotionDiscount float64     `json:"promotion_discount"`
	OrderedGrandTotal float64     `json:"ordered_grand_total"`
	GrandTotal        float64     `json:"grand_total"`
	State             string      `json:"state"`
	PaymentMethod     string      `json:"payment_method"`
	Note              string      `json:"note"`
	ListOrderItem     []OrderItem `json:"list_order_item" valid:"Required"`
	BuyerInfo         *BuyerInfo  `json:"buyer_info"`
	DeliveryFee       float64     `json:"delivery_fee"`
	DeliveryMethod    *string     `json:"delivery_method" valid:"Required" schema:"delivery_method"`
	CreateMethod      string      `json:"create_method" valid:"Required"`
	OtherDiscount     float64     `json:"other_discount"`
	Email             string      `json:"email"`
	ListProductFast   []Product   `json:"list_product_fast"`
	Debit             *Debit      `json:"debit,omitempty"`
	BuyerReceived     bool        `json:"buyer_received"`
	//BuyerId           *uuid.UUID  `json:"buyer_id"`

}

type BuyerInfo struct {
	PhoneNumber string  `json:"phone_number" valid:"Required"`
	Name        string  `json:"name" valid:"Required"`
	Address     string  `json:"address" valid:"Required"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type Debit struct {
	BuyerPay *float64       `json:"buyer_pay"`
	Note     string         `json:"note"`
	Images   pq.StringArray `json:"images" type:"type:varchar(500)[]"`
}

type RevenueBusiness struct {
	SumGrandTotal float64 `json:"sum_grand_total"`
}

type RevenueBusinessParam struct {
	BusinessID  uuid.UUID  `json:"business_id" schema:"business_id"`
	ContactID   uuid.UUID  `json:"contact_id" schema:"contact_id"`
	DateFrom    *time.Time `json:"date_from" schema:"date_from"`
	DateTo      *time.Time `json:"date_to" schema:"date_to"`
	UserRole    string     `json:"user_role"`
	UserCallAPI uuid.UUID  `json:"user_call_api"`
}

// Define your request param here
// Remember to user scheme tag
type OrderParam struct {
	R              *http.Request
	BusinessId     uuid.UUID  `json:"business_id" schema:"business_id"`
	ContactId      uuid.UUID  `json:"contact_id" schema:"contact_id"`
	PromotionCode  string     `json:"promotion_code" schema:"promotion_code"`
	State          string     `json:"state"`
	OrderNumber    string     `json:"order_number" schema:"order_number"`
	PaymentMethod  string     `json:"payment_method" schema:"payment_method"`
	Note           string     `json:"note"`
	Size           int        `json:"size"`
	Page           int        `json:"page"`
	Sort           string     `json:"sort"`
	BuyerId        uuid.UUID  `json:"buyer_id" schema:"buyer_id"`
	DateFrom       *time.Time `json:"date_from" schema:"date_from"`
	DateTo         *time.Time `json:"date_to" schema:"date_to"`
	Search         string     `json:"search" schema:"search"`
	SellerID       uuid.UUID  `json:"seller_id" schema:"seller_id"`
	UserRole       string     `json:"user_role"`
	UserCallAPI    uuid.UUID  `json:"user_call_api"`
	DeliveryMethod *string    `json:"delivery_method" schema:"delivery_method"`
	IsPrinted      *bool      `json:"is_printed" schema:"is_printed"`
}
