package model

import 	"github.com/google/uuid"

type OrderTracking struct {
	BaseModel
	OrderId uuid.UUID `json:"order_id" sql:"index" gorm:"column:order_id;not null;" schema:"order_id"`
	State   string    `json:"state" sql:"index" gorm:"column:state;not null;"`
}

