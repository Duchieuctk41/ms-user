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
	GetOneOrderBuyer(ctx context.Context, id string, tx *gorm.DB) (rs model.OrderBuyerResponse, err error)
	GetOneOrderRecent(ctx context.Context, buyerID string, tx *gorm.DB) (rs model.Order, err error)
	UpdateOrder(ctx context.Context, order model.Order, tx *gorm.DB) (rs model.Order, err error)
	UpdateOrderV2(ctx context.Context, order model.Order, tx *gorm.DB) (rs model.Order, err error)
	GetListOrderEcom(ctx context.Context, req model.OrderEcomRequest, tx *gorm.DB) (rs model.ListOrderEcomResponse, err error)
	GetAllOrder(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ListOrderResponse, err error)
	GetCompleteOrders(ctx context.Context, contactID uuid.UUID, tx *gorm.DB) (res model.GetCompleteOrdersResponse, err error)
	UpdateDetailOrder(ctx context.Context, order model.Order, mapItem map[string]model.OrderItem, tx *gorm.DB) (rs model.Order, stocks []model.StockRequest, err error)
	UpdateDetailOrderSellerV2(ctx context.Context, order model.Order, req []model.OrderItem, tx *gorm.DB) (rs model.Order, stocks []model.StockRequest, err error)
	GetOrderTracking(ctx context.Context, req model.OrderTrackingRequest, tx *gorm.DB) (rs model.OrderTrackingResponse, err error)
	CountOrderState(ctx context.Context, req model.RevenueBusinessParam, tx *gorm.DB) (res model.CountOrderState, err error)
	GetOrderByContact(ctx context.Context, req model.OrderByContactParam, tx *gorm.DB) (rs model.ListOrderResponse, err error)
	GetAllOrderForExport(ctx context.Context, req model.ExportOrderReportRequest, tx *gorm.DB) (orders []model.Order, err error)
	GetContactDelivering(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ContactDeliveringResponse, err error)
	GetTotalContactDelivery(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.TotalContactDelivery, err error)

	//analytics
	CountOrderAnalytics(ctx context.Context, req model.GetOrderAnalyticsRequest) (model.CountOrderAnalytics, error)

	//
	CreateOrderV2(ctx context.Context, order *model.Order, tx *gorm.DB) error
	GetlistOrderV2(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ListOrderResponse, err error)

	OverviewSales(ctx context.Context, req model.OrverviewPandLRequest, tx *gorm.DB) (model.OverviewPandLResponse, error)
	OverviewCost(ctx context.Context, req model.OrverviewPandLRequest, overviewPandL model.OverviewPandLResponse, tx *gorm.DB) (model.OverviewPandLResponse, error)

	GetListProfitAndLoss(ctx context.Context, req model.ProfitAndLossRequest, tx *gorm.DB) (model.GetListProfitAndLossResponse, error)
	GetCountQuantityInOrder(ctx context.Context, req model.CountQuantityInOrderRequest, tx *gorm.DB) (rs model.CountQuantityInOrderResponse, err error)
	GetCountQuantityInOrderEcom(ctx context.Context, req model.CountQuantityInOrderRequest, tx *gorm.DB) (rs model.CountQuantityInOrderResponse, err error)
	GetSumOrderCompleteContact(ctx context.Context, req model.GetTotalOrderByBusinessRequest, tx *gorm.DB) (rs []model.GetTotalOrderByBusinessResponse, err error)

	// tutorial flow
	CountOrderForTutorial(ctx context.Context, creatorID uuid.UUID, tx *gorm.DB) (count int, err error)

	// log history
	LogHistory(ctx context.Context, history model.History, tx *gorm.DB) (rs model.History, err error)
	DeleteLogHistory(ctx context.Context, tx *gorm.DB) error

	// ecom
	UpdateMultiOrderEcom(ctx context.Context, rs []model.OrderEcom, tx *gorm.DB)
	UpdateMultiEcomOrder(ctx context.Context, rs []model.EcomOrder, tx *gorm.DB)
	GetStateOrderEcom(ctx context.Context, id string, tx *gorm.DB) (rs model.EcomOrderState, err error)

	// payment_order_history
	CreatePaymentOrderHistory(ctx context.Context, payment *model.PaymentOrderHistory, tx *gorm.DB) (err error)
	GetAmountTotalPaymentOrderHistory(ctx context.Context, id string, tx *gorm.DB) (rs float64, err error)
	GetListPaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryParam, tx *gorm.DB) (rs []*model.PaymentOrderHistoryResponse, err error)
	//GetListPaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryParam, tx *gorm.DB) (rs model.GetListPaymentOrderHistoryResponse, err error)

	//web pro
	CountOrder(ctx context.Context, req model.OrverviewRequest, tx *gorm.DB) (model.OrderTotal, error)
	OverviewCostPandL(ctx context.Context, req model.OrverviewRequest, tx *gorm.DB) (model.CostTotal, error)
	GetOrderItemRevenueAnalytics(ctx context.Context, input model.GetOrderRevenueAnalyticsParam, tx *gorm.DB) (rs model.ListOrderRevenueAnalyticsResponse, err error)
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
