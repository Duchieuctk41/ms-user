package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	generalQueryTimeout = 60 * time.Second
)

func NewPGRepo(db *gorm.DB) PGInterface {
	return &RepoPG{DB: db}
}

type PGInterface interface {
	// DB
	DBWithTimeout(ctx context.Context) (*gorm.DB, context.CancelFunc)

	CreateOrder(ctx context.Context, order model.Order, tx *gorm.DB) (rs model.Order, err error)
	CreateOrderItem(ctx context.Context, orderItem model.OrderItem, tx *gorm.DB) (rs model.OrderItem, err error)
	CountOneStateOrder(ctx context.Context, businessId uuid.UUID, state string, tx *gorm.DB) int
	CreateOrderTracking(ctx context.Context, orderTracking model.OrderTracking, tx *gorm.DB) (err error)
	RevenueBusiness(ctx context.Context, req model.RevenueBusinessParam, tx *gorm.DB) (rs model.RevenueBusiness, err error)
	GetContactHaveOrder(ctx context.Context, businessId uuid.UUID, tx *gorm.DB) (string, int, error)
	GetOneOrder(ctx context.Context, id string, tx *gorm.DB) (rs model.Order, err error)
	GetOneOrderRecent(ctx context.Context, buyerID string, tx *gorm.DB) (rs model.Order, err error)
	UpdateOrder(ctx context.Context, order model.Order, tx *gorm.DB) (rs model.Order, err error)
}

type BaseModel struct {
	ID        uuid.UUID  `json:"id" gorm:"primary_key;type:uuid;default:uuid_generate_v4()"`
	CreatorID uuid.UUID  `json:"creator_id"`
	UpdaterID uuid.UUID  `json:"updater_id"`
	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time  `json:"updated_at" gorm:"column:updated_at;default:CURRENT_TIMESTAMP"`
	DeletedAt *time.Time `json:"deleted_at" sql:"index"`
}

type RepoPG struct {
	DB    *gorm.DB
	debug bool
}

func (r *RepoPG) GetRepo() *gorm.DB {
	return r.DB
}

func (r *RepoPG) DBWithTimeout(ctx context.Context) (*gorm.DB, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, generalQueryTimeout)
	return r.DB.WithContext(ctx), cancel
}
