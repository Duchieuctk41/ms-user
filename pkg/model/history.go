package model

import (
	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
)

type History struct {
	BaseModel
	ObjectID    uuid.UUID      `json:"object_id" sql:"index" gorm:"column:object_id;not null;" valid:"Required"`       // order_id | order_item_id | order_ecom_id
	ObjectTable string         `json:"object_table" sql:"index" gorm:"column:object_table;not null;" valid:"Required"` //object table used to define which table logged
	Action      string         `json:"action" sql:"index" gorm:"column:action;not null;" valid:"Required"`             // description action (create, update, delete, cancel
	Description string         `json:"description" gorm:"null"`
	Data        postgres.Jsonb `json:"data" gorm:"null type:jsonb"`
	Worker      string         `json:"worker" sql:"index" gorm:"column:worker;not null;" valid:"Required"`
}

func (History) TableName() string {
	return "history"
}
