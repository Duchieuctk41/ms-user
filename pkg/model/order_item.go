package model

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type OrderItem struct {
	BaseModel
	OrderID uuid.UUID `json:"order_id" sql:"index" gorm:"column:order_id;not null;"`
	//ProductID           uuid.UUID      `json:"product_id" sql:"index" gorm:"column:product_id;not null;"`
	ProductName         string         `json:"product_name" gorm:"column:product_name;"`
	ProductNormalPrice  float64        `json:"product_normal_price" gorm:"column:product_normal_price;"`
	ProductSellingPrice float64        `json:"product_selling_price" gorm:"column:product_selling_price;"`
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
	Price               float64        `json:"price" gorm:"column:price;"`
	HistoricalCost      float64        `json:"historical_cost" gorm:"column:historical_cost;"`
	WholesalePrice      *float64       `json:"wholesale_price"`
}

func (u *OrderItem) BeforeSave(tx *gorm.DB) (err error) {
	if u.Quantity != 0 {
		quantity, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", u.Quantity), 64)
		u.Quantity = quantity
	}
	return
}

func (u *OrderItem) AfterCreate(tx *gorm.DB) (err error) {
	// hieunm - 8/1/2022
	// Fix bug sort order items in order wrong bc duplicate created at value
	tx.Model(&OrderItem{}).Where("id = ?", u.ID).UpdateColumn("created_at", time.Now().UTC())
	return
}

func (OrderItem) TableName() string {
	return "order_item"
}

type OrderItemForSendEmail struct {
	ProductID           uuid.UUID `json:"product_id"`
	ProductName         string    `json:"product_name"`
	ProductImages       string    `json:"product_images"`
	Quantity            float64   `json:"quantity"`
	TotalAmount         float64   `json:"total_amount"`
	Note                string    `json:"note"`
	SkuID               uuid.UUID `json:"sku_id"`
	SkuCode             string    `json:"sku_code"`
	SkuName             string    `json:"sku_name"`
	UOM                 string    `json:"uom"`
	ProductNormalPrice  string    `json:"product_normal_price"`
	ProductSellingPrice string    `json:"product_selling_price"`
}

type OrderItemBuyerResponse struct {
	//BaseModel
	ID         uuid.UUID `json:"id"`
	BusinessID uuid.UUID `json:"business_id" sql:"index" gorm:"column:business_id;not null;" valid:"Required" scheme:"business_id"`
	OrderID    uuid.UUID `json:"order_id" sql:"index" gorm:"column:order_id;not null;"`
	//ProductID           uuid.UUID      `json:"product_id" sql:"index" gorm:"column:product_id;not null;"`
	ProductName         string         `json:"product_name" gorm:"column:product_name;"`
	ProductNormalPrice  float64        `json:"product_normal_price" gorm:"column:product_normal_price;"`
	ProductSellingPrice float64        `json:"product_selling_price" gorm:"column:product_selling_price;"`
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
	Price               float64        `json:"price" gorm:"column:price;"`
}
