package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/lib/pq"
)

type GetContactRequest struct {
	BusinessId  uuid.UUID `json:"business_id"`
	PhoneNumber string    `json:"phone_number"`
	Name        string    `json:"name"`
	Address     string    `json:"address"`
}

type GetContactResponse struct {
	Data ContactInfo `json:"data"`
}

type ContactInfo struct {
	Business Business `json:"business"`
	Contact  Contact  `json:"contact"`
}

type Business struct {
	ID                 uuid.UUID       `json:"id"`
	CreatorID          uuid.UUID       `json:"creator_id"`
	Name               string          `json:"name"`
	Key                string          `json:"key"`
	Description        string          `json:"description"`
	Avatar             string          `json:"avatar"`
	PhoneNumber        string          `json:"phone_number"`
	Background         pq.StringArray  `json:"background"`
	Address            string          `json:"address"`
	OpenTime           time.Time       `json:"open_time"`
	CloseTime          time.Time       `json:"close_time"`
	CategoryBusinessID pq.StringArray  `json:"category_business_id"`
	CustomFields       postgres.Hstore `json:"custom_fields"`
	DeliveryFee        float64         `json:"delivery_fee"`
	MinPriceFreeShip   float64         `json:"min_price_free_ship"`
}

type Contact struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name" gorm:"not null"`
	Email                string     `json:"email" gorm:"null"`
	PhoneNumber          string     `json:"phone_number" gorm:"null"`
	Avatar               string     `json:"avatar"`
	BusinessID           uuid.UUID  `json:"business_id,omitempty"`
	SourceKey            string     `json:"source_key"`
	SellerID             string     `json:"seller_id"`
	Address              string     `json:"address"`
	IsExpired            bool       `json:"is_expired" gorm:"not null;default:false"`
	BusinessHasContactID uuid.UUID  `json:"business_has_contact_id,omitempty" gorm:"-"`
	LatestSyncTime       time.Time  `json:"latest_sync_time"`
	FavoriteTime         *time.Time `json:"favorite_time" gorm:"null"`
}

type CheckValidOrderItemResponse struct {
	Status    string                    `json:"status"`
	ItemsInfo []CheckValidStockResponse `json:"items_info"`
}

type CheckValidStockResponse struct {
	Sku
	Stock *StockForCheckValid `json:"stock,omitempty"`
}

type Sku struct {
	ID              uuid.UUID      `json:"id"`
	SkuName         string         `json:"sku_name"`
	ProductName     string         `json:"product_name"`
	Quantity        float64        `json:"quantity,omitempty"`
	Media           pq.StringArray `json:"media"`
	SellingPrice    float64        `json:"selling_price"`
	NormalPrice     float64        `json:"normal_price"`
	OldNormalPrice  *float64       `json:"old_normal_price,omitempty"`
	OldSellingPrice *float64       `json:"old_selling_price,omitempty"`
	Uom             string         `json:"uom"`
	SkuCode         string         `json:"sku_code"`
	Barcode         string         `json:"barcode"`
	CanPickQuantity float64        `json:"can_pick_quantity"`
	Type            string         `json:"type"`
}

type StockForCheckValid struct {
	TotalQuantity      float64 `json:"total_quantity"`
	DeliveringQuantity float64 `json:"delivering_quantity"`
	BlockedQuantity    float64 `json:"blocked_quantity"`
	HistoricalCost     float64 `json:"historical_cost"`
}

type User struct {
	ID               uuid.UUID `json:"id"`
	FullName         string    `json:"full_name"`
	IsActive         bool      `json:"is_active"`
	Email            string    `json:"email"`
	PhoneNumber      string    `json:"phone_number"`
	Birthday         time.Time `json:"birthday"`
	Avatar           string    `json:"avatar"`
	LastLogin        time.Time `json:"last_login"`
	CountSentFreeSMS int       `json:"count_sent_free_sms"`
	UIDFirebase      string    `json:"uid_firebase"`
}

type Promotion struct {
	BusinessId       uuid.UUID  `json:"business_id" valid:"Required" scheme:"business_id"`
	CurrentSize      int        `json:"current_size"`
	MaxSize          int        `json:"max_size"`
	IsActive         bool       `json:"is_active"`
	Name             string     `json:"name"`
	Description      string     `json:"description"`
	PromotionCode    string     `json:"promotion_code" `
	StartTime        *time.Time `json:"start_time"`
	EndTime          *time.Time `json:"end_time"`
	Value            float64    `json:"value"`
	ValueDiscount    float64    `json:"value_discount"`
	Type             string     `json:"type"`
	MinOrderPrice    float64    `json:"min_order_price"`
	MaxPriceDiscount float64    `json:"max_price_discount"`
}

type CustomFieldsRequest struct {
	BusinessID   uuid.UUID       `json:"business_id"`
	CustomFields postgres.Hstore `json:"custom_fields"`
}

type BusinessTransaction struct {
	ID                 uuid.UUID      `json:"id"`
	CreatorID          uuid.UUID      `json:"creator_id"`
	TransactionType    string         `json:"transaction_type" gorm:"not null"`
	Status             string         `json:"status" gorm:"null"`
	PaymentMethod      string         `json:"payment_method" gorm:"null"`
	PaymentInformation postgres.Jsonb `json:"payment_information"`
	Amount             float64        `json:"amount" gorm:"not null"`
	Currency           string         `json:"currency" gorm:"not null"`
	BusinessID         uuid.UUID      `json:"business_id" gorm:"not null"`
	Images             pq.StringArray `json:"images" gorm:"type:varchar(500)[]"`
	ContactID          uuid.UUID      `json:"contact_id"`
	Contact            *Contact       `json:"contact,omitempty" gorm:"-"`
	CategoryID         uuid.UUID      `json:"category_id"`
	CategoryName       string         `json:"category_name"`
	Day                time.Time      `json:"day" gorm:"not null"`
	Description        string         `json:"description"`
	PayoutID           uuid.UUID      `json:"payout_id,omitempty"`
	Action             string         `json:"action"`
	LatestSyncTime     string         `json:"latest_sync_time"`
	OrderNumber        string         `json:"order_number"`
	Table              string         `json:"table"`
}

type ContactTransaction struct {
	Status          string         `json:"status" `
	CreatorID       uuid.UUID      `json:"creator_id"`
	TransactionType string         `json:"transaction_type" valid:"Required"`
	Currency        string         `json:"currency" valid:"Required"`
	ContactID       uuid.UUID      `json:"contact_id" valid:"Required"`
	BusinessID      uuid.UUID      `json:"business_id"`
	Amount          float64        `json:"amount" valid:"Required"`
	Images          pq.StringArray `json:"images" gorm:"type:varchar(500)[]"`
	Description     string         `json:"description"`
	StartTime       time.Time      `json:"start_time" valid:"Required"`
	EndTime         time.Time      `json:"end_time"`
	Action          string         `json:"action"`
	ID              uuid.UUID      `json:"id"`
	LatestSyncTime  string         `json:"latest_sync_time"`
	OrderNumber     string         `json:"order_number"`
	Table           string         `json:"table"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type SendEmailRequest struct {
	ID       string `json:"id"`
	State    string `json:"state"`
	UserRole string `json:"user_role"`
}

type PurchaseOrderRequest struct {
	//ID            *uuid.UUID      `json:"id,omitempty"`
	//UpdaterID     *uuid.UUID      `json:"updater_id,omitempty"`
	//CreatorID     *uuid.UUID      `json:"creator_id,omitempty"`
	PoType string `json:"po_type,omitempty"`
	//Media         *pq.StringArray `json:"media,omitempty"`
	Note      string    `json:"note,omitempty"`
	ContactID uuid.UUID `json:"contact_id,omitempty"`
	//CategoryID    *uuid.UUID      `json:"category_id,omitempty"`
	TotalDiscount float64 `json:"total_discount,omitempty"`
	//SurCharge     *float64        `json:"sur_charge,omitempty"`
	TransactionID uuid.UUID  `json:"transaction_id,omitempty"`
	BusinessID    uuid.UUID  `json:"business_id,omitempty"`
	PoDetails     []PoDetail `json:"po_details,omitempty"`
	Option        string     `json:"option,omitempty"`
}

type PoDetail struct {
	BaseModel
	SkuID           uuid.UUID `json:"sku_id" gorm:"not null;index;type:uuid"`
	PoID            uuid.UUID `json:"po_id" gorm:"type:uuid;index;not null"`
	Pricing         float64   `json:"pricing" gorm:"not null"`
	Quantity        float64   `json:"quantity" gorm:"not null"`
	Note            string    `json:"note"`
	BlockedQuantity *float64  `json:"blocked_quantity,omitempty" gorm:"-"`
	WarningValue    *float64  `json:"warning_value,omitempty" gorm:"-"`
}

type SendNotificationRequest struct {
	UserId         uuid.UUID `json:"user_id" `
	EntityKey      string    `json:"entity_key"  `
	StateValue     string    `json:"state_value" `
	Language       string    `json:"language"    `
	ContentReplace string    `json:"content_replace"`
}

type CreateStockRequest struct {
	ListStock      []StockRequest `json:"list_stock" valid:"Required"`
	TrackingType   string         `json:"tracking_type" valid:"Required"`
	TrackingInfo   postgres.Jsonb `json:"tracking_info"`
	IDTrackingType string         `json:"id_tracking_type" valid:"Required"`
	BusinessID     uuid.UUID      `json:"business_id" valid:"Required"`
}

type StockRequest struct {
	ID                 uuid.UUID `json:"id,omitempty"`
	UpdaterID          uuid.UUID `json:"updater_id,omitempty"`
	CreatorID          uuid.UUID `json:"creator_id,omitempty"`
	SkuID              uuid.UUID `json:"sku_id,omitempty" valid:"Required"`
	BusinessID         uuid.UUID `json:"business_id,omitempty"`
	DeliveringQuantity float64   `json:"delivering_quantity,omitempty"`
	BlockedQuantity    float64   `json:"blocked_quantity,omitempty"`
	WarningValue       float64   `json:"warning_value,omitempty"`
	HistoricalCost     float64   `json:"historical_cost,omitempty"`
	QuantityChange     float64   `json:"quantity_change,omitempty"`
}

type Product struct {
	Name           string         `json:"name" valid:"Required"`
	Description    string         `json:"description"`
	IsActive       bool           `json:"is_active"`
	Images         pq.StringArray `json:"images"`
	Priority       int            `json:"priority"`
	SellingPrice   float64        `json:"selling_price"`
	HistoricalCost float64        `json:"historical_cost" valid:"Required"`
	NormalPrice    float64        `json:"normal_price" valid:"Required"`
	Uom            string         `json:"uom"`
	SkuCode        string         `json:"sku_code"`
	Type           string         `json:"type"`
	Barcode        string         `json:"barcode,omitempty"`
	CategoryID     uuid.UUID      `json:"category_id"`
	IsProductFast  bool           `json:"is_product_fast"`
	Quantity       float64        `json:"quantity" gorm:"column:quantity"`
	TotalAmount    float64        `json:"total_amount" gorm:"column:total_amount"`
}

type CreateProductFast struct {
	BusinessID      *uuid.UUID `json:"business_id" valid:"Required"`
	ListProductFast []Product  `json:"list_product_fast"`
}

type CheckDuplicateProductRequest struct {
	BusinessID *uuid.UUID `json:"business_id" valid:"Required"`
	Names      []string   `json:"names"`
}

type BusinessMainInfo struct {
	ID               uuid.UUID      `json:"id"`
	Domain           string         `json:"domain"`
	Name             string         `json:"name"`
	Description      string         `json:"description" `
	Avatar           string         `json:"avatar"`
	PhoneNumber      string         `json:"phone_number"`
	Background       pq.StringArray `json:"background"`
	Address          string         `json:"address"`
	OpenTime         time.Time      `json:"open_time"`
	CloseTime        time.Time      `json:"close_time"`
	DeliveryFee      float64        `json:"delivery_fee"`
	MinPriceFreeShip float64        `json:"min_price_free_ship"`
	BusinessType     interface{}    `json:"business_type,omitempty"`
}

type ProcessConsumerRequest struct {
	Topic   string `json:"topic"`
	Payload string `json:"payload"`
}
