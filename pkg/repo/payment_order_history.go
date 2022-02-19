package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/utils"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"gorm.io/gorm"
	"net/http"
)

func (r *RepoPG) CreatePaymentOrderHistory(ctx context.Context, payment *model.PaymentOrderHistory, tx *gorm.DB) (err error) {
	log := logger.WithCtx(ctx, "RepoPG.CreatePaymentOrderHistory")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Create(&payment).Error; err != nil {
		log.WithError(err).Error("error_500: create contact_transaction in CreatePaymentOrderHistory - RepoPG")
		return ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	return nil
}

func (r *RepoPG) GetAmountTotalPaymentOrderHistory(ctx context.Context, id string, tx *gorm.DB) (rs float64, err error) {
	log := logger.WithCtx(ctx, "RepoPG.GetAmountTotalPaymentOrderHistory")
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Model(&model.PaymentOrderHistory{}).Select("COALESCE(sum(amount),0)").Where("order_id = ?", id).Find(&rs).Error; err != nil {
		log.WithError(err).Error("error_500: create payment_order_history in GetAmountTotalPaymentOrderHistory - RepoPG")
		return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	return rs, nil
}

//
//func (r *RepoPG) GetListPaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryParam, tx *gorm.DB) (rs model.GetListPaymentOrderHistoryResponse, err error) {
//	log := logger.WithCtx(ctx, "RepoPG.GetListPaymentOrderHistory")
//
//	var cancel context.CancelFunc
//	if tx == nil {
//		tx, cancel = r.DBWithTimeout(ctx)
//		defer cancel()
//	}
//
//	page := r.GetPage(req.Page)
//	pageSize := r.GetPageSize(req.PageSize)
//
//	var total int64 = 0
//
//	if err = tx.Model(&model.PaymentOrderHistory{}).Where("order_id = ?", req.OrderID).Limit(pageSize).Offset(r.GetOffset(page, pageSize)).Count(&total).Find(&rs.Data).Error; err != nil {
//		if err == gorm.ErrRecordNotFound {
//			log.WithError(err).Error("error_404: record not found when call GetListPaymentOrderHistory - RepoPG")
//			return rs, ginext.NewError(http.StatusNotFound, "không tìm lịch sử thanh toán")
//		} else {
//			log.WithError(err).Error("error_500: create payment_order_history in GetListPaymentOrderHistory - RepoPG")
//			return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
//		}
//	}
//
//	if rs.Meta, err = r.GetPaginationInfo("", tx, int(total), page, pageSize); err != nil {
//		return rs, err
//	}
//
//	return rs, nil
//}

func (r *RepoPG) GetListPaymentOrderHistory(ctx context.Context, req model.PaymentOrderHistoryParam, tx *gorm.DB) (rs []*model.PaymentOrderHistoryResponse, err error) {
	log := logger.WithCtx(ctx, "RepoPG.GetListPaymentOrderHistory")

	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Model(&model.PaymentOrderHistory{}).Where("order_id = ?", req.OrderID).Find(&rs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.WithError(err).Error("error_404: record not found when call GetListPaymentOrderHistory - RepoPG")
			return rs, ginext.NewError(http.StatusNotFound, "không tìm lịch sử thanh toán")
		} else {
			log.WithError(err).Error("error_500: create payment_order_history in GetListPaymentOrderHistory - RepoPG")
			return rs, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
	}

	return rs, nil
}
