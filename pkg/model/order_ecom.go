package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
)

type OrderEcom struct {
	ID                uuid.UUID      `gorm:"primary_key;type:uuid;default:uuid_generate_v4()" json:"id"`
	CreatedAt         time.Time      `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt         *time.Time     `json:"deleted_at,omitempty" sql:"index"`
	CreatedTime       *time.Time     `json:"created_time"`
	UpdatedTime       *time.Time     `json:"updated_time"`
	BusinessID        uuid.UUID      `json:"business_id" sql:"index" gorm:"column:business_id;not null;" valid:"Required"`
	ContactID         uuid.UUID      `json:"contact_id" sql:"index" gorm:"column:contact_id;not null;" valid:"Required"`
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
	DeliveryMethod    string         `json:"delivery_method" sql:"index" gorm:"column:delivery_method;"`
	OrderItem         postgres.Jsonb `json:"order_item" gorm:"type:jsonb" valid:"Required"`
	OtherDiscount     float64        `json:"other_discount" gorm:""`
	BusinessHasShopID uuid.UUID      `json:"business_has_shop_id" gorm:"type:uuid;not null" sql:"index" valid:"Required"`
	OrderIDEcom       string         `json:"order_id_ecom" sql:"index" gorm:"not null" valid:"Required"`
}

func (OrderEcom) TableName() string {
	return "order_ecom"
}

type OrderEcomRequest struct {
	BusinessID *string    `json:"business_id" form:"business_id" valid:"Required"`
	PageSize   int        `json:"page_size" form:"page_size"`
	Page       int        `json:"page" form:"page"`
	Sort       string     `json:"sort" form:"sort"`
	StartTime  *time.Time `json:"start_time" form:"start_time"`
	EndTime    *time.Time `json:"end_time" form:"end_time"`
	Search     string     `json:"search" form:"search"`
}

type ListOrderEcomResponse struct {
	Data []OrderEcom            `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}

type SkuHasSkuEcom struct {
	EcomSkuID string `json:"ecom_sku_id"`
	SkuID     string `json:"sku_id"`
}
