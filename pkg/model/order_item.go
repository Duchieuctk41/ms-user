package model

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type OrderItem struct {
	BaseModel
	OrderId             uuid.UUID      `json:"order_id" sql:"index" gorm:"column:order_id;not null;"`
	ProductId           uuid.UUID      `json:"product_id" sql:"index" gorm:"column:product_id;not null;"`
	ProductName         string         `json:"product_name" gorm:"column:product_name;"`
	ProductNormalPrice  float64        `json:"product_normal_price" gorm:"column:product_normal_price;default:0"`
	ProductSellingPrice float64        `json:"product_selling_price" gorm:"column:product_selling_price;default:0"`
	ProductImages       pq.StringArray `json:"product_images" gorm:"type:varchar(500)[]"`
	Quantity            float64        `json:"quantity" gorm:"column:quantity"`
	TotalAmount         float64        `json:"total_amount" gorm:"column:total_amount"`
	Note                string         `json:"note" gorm:"column:note"`
	SkuID               uuid.UUID      `json:"sku_id" gorm:"type:uuid"`
	SkuCode             string         `json:"sku_code" gorm:"type:varchar(500)"`
	SkuName             string         `json:"sku_name" gorm:"type:varchar(1000)"`
	UOM                 string         `json:"uom" gorm:"type:varchar(1000)"`
	ProductType         *string        `json:"product_type,omitempty" gorm:"-"`
	CanPickQuantity     *float64       `json:"can_pick_quantity,omitempty" gorm:"-"`
	SkuActive           *bool          `json:"sku_active,omitempty" gorm:"-"`
}

func (u *OrderItem) BeforeSave(tx *gorm.DB) (err error) {
	quantity, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", u.Quantity), 64)
	u.Quantity = quantity
	return
}

type OrderItemForSendEmail struct {
	ProductId           uuid.UUID `json:"product_id"`
	ProductName         string    `json:"product_name"`
	ProductImages       string    `json:"product_images"`
	Quantity            float64   `json:"quantity"`
	TotalAmount         float64   `json:"total_amount"`
	Note                string    `json:"note"`
	SkuID               uuid.UUID `json:"sku_id"`
	SkuCode             string    `json:"sku_code"`
	SkuName             string    `json:"sku_name"`
	UOM                 string    `json:"uom"`
	ProductNormalPrice  float64   `json:"product_normal_price"`
	ProductSellingPrice float64   `json:"product_selling_price"`
}
