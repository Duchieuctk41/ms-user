package model

import "github.com/google/uuid"

type OrderTracking struct {
	BaseModel
	OrderID uuid.UUID `json:"order_id" sql:"index" gorm:"column:order_id;not null;" schema:"order_id"`
	State   string    `json:"state" sql:"index" gorm:"column:state;not null;"`
}

func (OrderTracking) TableName() string {
	return "order_tracking"
}

type OrderTrackingRequest struct {
	UserID   uuid.UUID `json:"user_id"`
	UserRole string    `json:"user_role"`
	OrderID  uuid.UUID `json:"order_id" form:"order_id"`
	Page     int       `json:"page" form:"page"`
	PageSize int       `json:"page_size" form:"page_size"`
	Sort     string    `json:"sort" form:"sort"`
}

type OrderTrackingResponse struct {
	Data []OrderTracking        `json:"order_tracking"`
	Meta map[string]interface{} `json:"meta"`
}
