package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
)

type EcomOrder struct {
	ID                uuid.UUID       `gorm:"primary_key;type:uuid;default:uuid_generate_v4()" json:"id"`
	CreatedAt         time.Time       `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt         time.Time       `gorm:"column:updated_at;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt         *time.Time      `json:"deleted_at,omitempty" sql:"index"`
	CreatedTime       *time.Time      `json:"created_time"`
	UpdatedTime       *time.Time      `json:"updated_time"`
	BusinessID        uuid.UUID       `json:"business_id" sql:"index" gorm:"column:business_id;not null;" valid:"Required"`
	ContactID         uuid.UUID       `json:"contact_id" sql:"index" gorm:"column:contact_id;not null;" valid:"Required"`
	OrderNumber       string          `json:"order_number" sql:"index" gorm:"column:order_number;not null;"`
	PromotionCode     string          `json:"promotion_code" gorm:"column:promotion_code;null;"`
	OrderedGrandTotal float64         `json:"ordered_grand_total" gorm:"column:ordered_grand_total"`
	PromotionDiscount float64         `json:"promotion_discount" gorm:"promotion_discount"`
	DeliveryFee       float64         `json:"delivery_fee" gorm:"column:delivery_fee"`
	GrandTotal        float64         `json:"grand_total" gorm:"grand_total"`
	State             string          `json:"state" sql:"index" gorm:"column:state;not null;"`
	PaymentMethod     string          `json:"payment_method" sql:"index" gorm:"column:payment_method;"`
	Note              string          `json:"note" gorm:"column:note;null;"`
	BuyerInfo         postgres.Jsonb  `json:"buyer_info" gorm:"null"`
	DeliveryMethod    string          `json:"delivery_method" sql:"index" gorm:"column:delivery_method;"`
	EcomOrderItem     []EcomOrderItem `json:"ecom_order_item"  gorm:"foreignkey:order_id;association_foreignkey:id;"`
	OtherDiscount     float64         `json:"other_discount" gorm:""`
	BusinessHasShopID uuid.UUID       `json:"business_has_shop_id" gorm:"type:uuid;not null" sql:"index" valid:"Required"`
	OrderIDEcom       string          `json:"order_id_ecom" sql:"index" gorm:"not null" valid:"Required"`
}

func (EcomOrder) TableName() string {
	return "ecom_order"
}

type EcomOrderItem struct {
	BaseModel
	OrderID uuid.UUID `json:"order_id" sql:"index" gorm:"column:order_id;not null;"`
	//ProductID           uuid.UUID      `json:"product_id" sql:"index" gorm:"column:product_id;not null;"`
	ProductName     string         `json:"product_name" gorm:"column:product_name;"`
	NormalPrice     float64        `json:"normal_price" gorm:"column:normal_price;"`
	SellingPrice    float64        `json:"selling_price" gorm:"column:selling_price;"`
	ProductImages   pq.StringArray `json:"product_images" gorm:"type:varchar(500)[]"`
	Quantity        float64        `json:"quantity" gorm:"column:quantity"`
	TotalAmount     float64        `json:"total_amount" gorm:"column:total_amount"`
	Note            string         `json:"note" gorm:"column:note"`
	SkuID           uuid.UUID      `json:"sku_id" gorm:"type:uuid"`
	SkuCode         string         `json:"sku_code" gorm:"type:varchar(500)"`
	SkuName         string         `json:"sku_name" gorm:"type:varchar(1000)"`
	UOM             string         `json:"uom" gorm:"type:varchar(1000)"`
	ProductType     *string        `json:"product_type,omitempty" gorm:"-"`
	CanPickQuantity *float64       `json:"can_pick_quantity,omitempty" gorm:"-"`
	SkuActive       *bool          `json:"sku_active,omitempty" gorm:"-"`
	Price           float64        `json:"price" gorm:"column:price;"`
	HistoricalCost  float64        `json:"historical_cost" gorm:"column:historical_cost;"`
	WholesalePrice  *float64       `json:"wholesale_price"`
}

func (OrderItem) EcomOrderItem() string {
	return "ecom_order_item"
}
