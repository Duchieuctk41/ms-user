package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"math"
	"time"

	"github.com/google/uuid"
	"gitlab.com/goxp/cloud0/ginext"
	"gorm.io/gorm"
)

const (
	StateNew byte = iota + 1 // starts from 1
	StateDoing
	StateDone

	generalQueryTimeout = 60 * time.Second
	defaultPageSize     = 30
	maxPageSize         = 1000
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
	OverviewSales(ctx context.Context, req model.OrverviewPandLRequest, tx *gorm.DB) (model.OverviewPandLResponse, error)
	OverviewCost(ctx context.Context, req model.OrverviewPandLRequest, overviewPandL model.OverviewPandLResponse, tx *gorm.DB) (model.OverviewPandLResponse, error)

	//
	GetListProfitAndLoss(ctx context.Context, req model.ProfitAndLossRequest, tx *gorm.DB) (model.GetListProfitAndLossResponse, error)
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

func (r *RepoPG) GetPage(page int) int {
	if page == 0 {
		return 1
	}
	return page
}

func (r *RepoPG) GetOffset(page int, pageSize int) int {
	return (page - 1) * pageSize
}

func (r *RepoPG) GetPageSize(pageSize int) int {
	if pageSize == 0 {
		return defaultPageSize
	}
	if pageSize > maxPageSize {
		return maxPageSize
	}
	return pageSize
}

func (r *RepoPG) GetTotalPages(totalRows, pageSize int) int {
	return int(math.Ceil(float64(totalRows) / float64(pageSize)))
}

func (r *RepoPG) GetOrder(sort string) string {
	if sort == "" {
		sort = "created_at desc"
	}
	return sort
}

func (r *RepoPG) GetPaginationInfo(query string, tx *gorm.DB, totalRow, page, pageSize int) (rs ginext.BodyMeta, err error) {
	tm := struct {
		Count int `json:"count"`
	}{}
	if query != "" {
		if err = tx.Raw(query).Scan(&tm).Error; err != nil {
			return nil, err
		}
		totalRow = tm.Count
	}

	return ginext.BodyMeta{
		"page":        page,
		"page_size":   pageSize,
		"total_pages": r.GetTotalPages(totalRow, pageSize),
		"total_rows":  totalRow,
	}, nil
}
