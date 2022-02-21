package service

import (
	"context"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/repo"
	"finan/ms-order-management/pkg/utils"
	"finan/ms-order-management/pkg/valid"
	"github.com/google/uuid"
	"github.com/praslar/lib/common"
	"github.com/sendgrid/rest"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type PaymentOrderHistoryService struct {
	repo           repo.PGInterface
	historyService HistoryServiceInterface
}

func NewPaymentOrderHistoryService(repo repo.PGInterface, historyService HistoryServiceInterface) PaymentOrderHistoryInterface {
	return &PaymentOrderHistoryService{repo: repo, historyService: historyService}
}

type PaymentOrderHistoryInterface interface {
	CreatePaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryRequest, userID uuid.UUID) (res interface{}, err error)
	GetListPaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryParam) (res interface{}, err error)
}

func (s *PaymentOrderHistoryService) CreatePaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryRequest, userID uuid.UUID) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "PaymentOrderHistoryService.CreatePaymentOrderHistory")

	// get grand-total-order
	order, err := s.repo.GetOneOrder(ctx, req.OrderID.String(), nil)
	if err != nil {
		// if err is not found return 404
		if err == gorm.ErrRecordNotFound {
			log.WithError(err).Error("error_404: GetOneOrder not found")
			return res, nil
		} else {
			log.WithError(err).Error("error_500: Error when call GetOneOrder in CreatePaymentOrderHistory")
			return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
	}

	// get amount-total of payment_order_history
	totalPayment, err := s.repo.GetAmountTotalPaymentOrderHistory(ctx, req.OrderID.String(), nil)
	if err != nil {
		return res, err
	}

	// check totalPayment vs order_grand_total
	if totalPayment >= order.GrandTotal {
		log.WithError(err).Error("error_400: Khách đã thanh toán đủ tiền")
		return res, ginext.NewError(http.StatusInternalServerError, "Khách đã thanh toán đủ tiền")
	}

	// common sync to payment_order_history
	payment := model.PaymentOrderHistory{
		BaseModel: model.BaseModel{
			CreatorID: userID,
		},
		OrderID:         valid.UUID(req.OrderID),
		Name:            valid.String(req.Name),
		PaymentMethod:   valid.String(req.PaymentMethod),
		PaymentSourceID: valid.UUID(req.PaymentSourceID),
		Day:             time.Now().UTC(),
	}

	// check debt_amount vs request amount
	debtAmount := order.GrandTotal - totalPayment
	if debtAmount <= valid.Float64(req.Amount) {
		payment.Amount = debtAmount
	} else {
		payment.Amount = valid.Float64(req.Amount)
	}

	// begin transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	// get shop info
	if err = s.repo.CreatePaymentOrderHistory(ctx, &payment, tx); err != nil {
		log.Errorf("Fail to CreatePaymentOrderHistory due to %v", err)
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	// log history payment_order_history
	go func() {
		desc := utils.ACTION_CREATE_PAYMENT_ORDER_HISTORY + " in CreatePaymentOrderHistory func - PaymentOrderHistoryService"
		history, _ := utils.PackHistoryModel(context.Background(), userID, userID.String(), payment.ID, utils.TABLE_PAYMENT_ORDER_HISTORY, utils.ACTION_CREATE_PAYMENT_ORDER_HISTORY, desc, payment, req)
		s.historyService.LogHistory(context.Background(), history, tx)
	}()

	// set description
	desc := "Thanh toán trước một phần cho đơn" + order.OrderNumber
	if order.GrandTotal <= payment.Amount {
		desc = "Thanh toán trước cho đơn" + order.OrderNumber
	}
	// Create Business transaction
	if err = s.CreateBusinessTransactionV2(ctx, order, payment, desc, userID); err != nil {
		return res, err
	}

	// count
	order.AmountPaid = totalPayment + payment.Amount
	order.UpdaterID = userID
	if _, err = s.repo.UpdateOrderV2(ctx, order, tx); err != nil {
		return res, err
	}

	// log history payment_order_history
	go func() {
		desc := utils.ACTION_UPDATE_ORDER + " amount_paid in CreatePaymentOrderHistory func - PaymentOrderHistoryService"
		history, _ := utils.PackHistoryModel(context.Background(), userID, userID.String(), payment.ID, utils.TABLE_ORDER, utils.ACTION_UPDATE_ORDER, desc, payment, req)
		s.historyService.LogHistory(context.Background(), history, tx)
	}()

	tx.Commit()

	return payment, nil
}

func (s *PaymentOrderHistoryService) GetListPaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryParam) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "PaymentOrderHistoryService.GetListPaymentOrderHistory")

	if res, err = s.repo.GetListPaymentOrderHistory(ctx, req, nil); err != nil {
		log.Errorf("Fail to GetListPaymentOrderHistory due to %v", err)
		return res, err
	}

	return res, nil
}

func (s *PaymentOrderHistoryService) CreateBusinessTransactionV2(ctx context.Context, order model.Order, payment model.PaymentOrderHistory, desc string, userID uuid.UUID) error {
	log := logger.WithCtx(ctx, "OrderService.CreateBusinessTransactionV2 - PaymentOrderHistoryService")
	// Create Business transaction
	cateIDSell, _ := uuid.Parse(utils.CATEGORY_SELL)
	businessTransaction := model.BusinessTransaction{
		ID:                uuid.New(),
		CreatorID:         userID,
		BusinessID:        order.BusinessID,
		Day:               time.Now().UTC(),
		Amount:            payment.Amount,
		Currency:          "VND",
		TransactionType:   "in",
		Status:            "paid",
		Action:            "create",
		Description:       desc,
		CategoryID:        cateIDSell,
		CategoryName:      "Bán hàng",
		LatestSyncTime:    time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		ObjectKey:         order.OrderNumber,
		ObjectType:        "order",
		Table:             "income",
		PaymentMethod:     order.PaymentMethod,
		PaymentSourceID:   payment.PaymentSourceID,
		PaymentSourceName: payment.Name,
	}

	header := make(map[string]string)
	header["x-user-id"] = userID.String()

	// 22-01-2022 - thanhvc - skip process complete mission case_book
	// add more header skip-complete-mission = true, when call api to ms-transaction
	// it will skip processing complete mission cash_book
	header["skip-complete-mission"] = "true"

	_, _, err := common.SendRestAPI(conf.LoadEnv().FinanTransaction+"/api/v1/business-transaction/create", rest.Post, header, nil, businessTransaction)
	if err != nil {
		log.WithError(err).Error("Error when create business transaction in CreateBusinessTransactionV2 - PaymentOrderHistoryService")
		return err
	}
	return nil
}
