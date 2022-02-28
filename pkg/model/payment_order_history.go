package model

import (
	"github.com/google/uuid"
	"time"
)

type PaymentOrderHistory struct {
	BaseModel
	Name            string    `json:"name" gorm:"column:name;null"`
	PaymentSourceID uuid.UUID `json:"payment_source_id" gorm:"column:payment_source_id; not null;" sql:"index"`
	OrderID         uuid.UUID `json:"order_id" gorm:"column:order_id; not null;" sql:"index"`
	Amount          float64   `json:"amount" gorm:"column:amount;not null;" sql:"index"`
	PaymentMethod   string    `json:"payment_method" gorm:"column:payment_method"`
}

func (PaymentOrderHistory) TableName() string {
	return "payment_order_history"
}

type PaymentOrderHistoryRequest struct {
	BusinessID      *uuid.UUID `json:"business_id" valid:"Required"`
	OrderID         *uuid.UUID `json:"order_id" valid:"Required"`
	Name            *string    `json:"name"`
	PaymentSourceID *uuid.UUID `json:"payment_source_id" valid:"Required"`
	Amount          *float64   `json:"amount,omitempty" valid:"Required"`
	PaymentMethod   *string    `json:"payment_method" valid:"Required"`
}

type PaymentOrderHistoryParam struct {
	OrderID    string `json:"order_id" form:"order_id"`
	BusinessID string `json:"business_id" form:"business_id"`
	Page       int    `json:"page" form:"page"`
	PageSize   int    `json:"page_size" form:"page_size"`
}

type PaymentOrderHistoryResponse struct {
	ID              uuid.UUID `json:"id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	Name            string    `json:"name"`
	PaymentSourceID uuid.UUID `json:"payment_source_id"`
	OrderID         uuid.UUID `json:"order_id"`
	Amount          float64   `json:"amount"`
	PaymentMethod   string    `json:"payment_method"`
}

type CreatePaymentOrderHistoryResponse struct {
	Data PaymentOrderHistory    `json:"data"`
	Meta map[string]interface{} `json:"meta"`
}
