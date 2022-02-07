package service

import (
	"bytes"
	"context"
	"encoding/json"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/repo"
	"finan/ms-order-management/pkg/utils"
	"finan/ms-order-management/pkg/valid"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/sirupsen/logrus"
	"gitlab.com/goxp/cloud0/logger"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/praslar/lib/common"
	"github.com/sendgrid/rest"
	sendinblue "github.com/sendinblue/APIv3-go-library/lib"
	"gitlab.com/goxp/cloud0/ginext"
	"gorm.io/gorm"
)

type OrderService struct {
	repo           repo.PGInterface
	historyService HistoryServiceInterface
}

func NewOrderService(repo repo.PGInterface, historyService HistoryServiceInterface) OrderServiceInterface {
	return &OrderService{repo: repo, historyService: historyService}
}

type OrderServiceInterface interface {
	CreateOrder(ctx context.Context, req model.OrderBody) (res interface{}, err error)
	ProcessConsumer(ctx context.Context, req model.ProcessConsumerRequest) (res interface{}, err error)
	UpdateOrder(ctx context.Context, req model.OrderUpdateBody, userRole string) (res interface{}, err error)
	GetListOrderEcom(ctx context.Context, req model.OrderEcomRequest) (res model.ListOrderEcomResponse, err error)
	GetAllOrder(ctx context.Context, req model.OrderParam) (res model.ListOrderResponse, err error)
	UpdateDetailOrder(ctx context.Context, req model.UpdateDetailOrderRequest, userRole string) (res interface{}, err error)
	CountOrderState(ctx context.Context, req model.RevenueBusinessParam) (res interface{}, err error)
	GetOrderByContact(ctx context.Context, req model.OrderByContactParam) (res model.ListOrderResponse, err error)
	ExportOrderReport(ctx context.Context, req model.ExportOrderReportRequest) (res interface{}, err error)
	GetContactDelivering(ctx context.Context, req model.OrderParam) (res model.ContactDeliveringResponse, err error)
	GetOneOrder(ctx context.Context, req model.GetOneOrderRequest) (res interface{}, err error)

	// version 2
	CreateOrderV2(ctx context.Context, req model.OrderBody) (res interface{}, err error)

	CountDeliveringQuantity(ctx context.Context, req model.CountQuantityInOrderRequest) (rs interface{}, err error)
	GetTotalContactDelivery(ctx context.Context, req model.OrderParam) (rs model.TotalContactDelivery, err error)

	GetSumOrderCompleteContact(ctx context.Context, req model.GetTotalOrderByBusinessRequest) (rs interface{}, err error)

	//SendEmailOrder(ctx context.Context, req model.SendEmailRequest) (res interface{}, err error)
}

func (s *OrderService) GetOneOrder(ctx context.Context, req model.GetOneOrderRequest) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetOneOrder")

	order, err := s.repo.GetOneOrder(ctx, valid.String(req.ID), nil)
	if err != nil {
		// if err is not found return 404
		if err == gorm.ErrRecordNotFound {
			log.WithError(err).Error("GetOneOrder not found")
			return res, nil
		} else {
			log.WithError(err).Error("Record not found")
			return res, ginext.NewError(http.StatusBadRequest, err.Error())
		}
	}
	// check permission
	if err := utils.CheckPermissionV2(ctx, req.UserRole, req.UserID, order.BusinessID.String(), order.BuyerId.String()); err != nil {
		return nil, ginext.NewError(http.StatusUnauthorized, err.Error())
	}

	rs := struct {
		model.Order
		BusinessInfo model.BusinessMainInfo `json:"business_info"`
	}{Order: order}

	// get shop info
	if rs.BusinessInfo, err = s.GetDetailBusiness(ctx, rs.BusinessID.String()); err != nil {
		logrus.Errorf("Fail to get business detail due to %v", err)
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	return rs, nil
}

func (s *OrderService) CreateOrder(ctx context.Context, req model.OrderBody) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.CreateOrder")

	// Check format phone
	if !utils.ValidPhoneFormat(req.BuyerInfo.PhoneNumber) {
		log.WithError(err).Error("Error when check format phone")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	//
	orderGrandTotal := 0.0
	promotionDiscount := 0.0
	deliveryFee := 0.0
	grandTotal := 0.0

	getContactRequest := model.GetContactRequest{
		BusinessID:  *req.BusinessID,
		Name:        req.BuyerInfo.Name,
		PhoneNumber: req.BuyerInfo.PhoneNumber,
		Address:     req.BuyerInfo.Address,
	}

	// Get Contact Info
	info, err := s.GetContactInfo(ctx, getContactRequest)
	if err != nil {
		return nil, err
	}

	var lstOrderItem []model.OrderItem

	if len(req.ListProductFast) > 0 {

		// Check duplicate name
		productFast := make(map[string]string)
		productNormal := make(map[string]string)
		var lstProduct []string
		for _, v := range req.ListProductFast {
			if v.IsProductFast { // san pham nhanh
				if productFast[v.Name] == v.Name {
					log.WithError(err).Errorf("Error when create duplicated product name")
					return nil, ginext.NewError(http.StatusBadRequest, "Tạo sản phẩm không được trùng tên trong cùng một đơn hàng")
				}
				productFast[v.Name] = v.Name
			} else { // san pham thuong
				if productNormal[v.Name] == v.Name {
					log.WithError(err).Errorf("Error when create duplicated product name")
					return nil, ginext.NewError(http.StatusBadRequest, "Tạo sản phẩm không được trùng tên trong cùng một đơn hàng")
				}
				productNormal[v.Name] = v.Name
				lstProduct = append(lstProduct, v.Name)
			}
		}
		checkDuplicateProduct := model.CheckDuplicateProductRequest{
			BusinessID: req.BusinessID,
			Names:      lstProduct,
		}

		// call ms-product-management to check duplicate product name of product normal
		header := make(map[string]string)
		header["x-user-roles"] = strconv.Itoa(utils.ADMIN_ROLE)
		header["x-user-id"] = req.UserID.String()
		_, _, err = common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v1/product/check-duplicate-name", rest.Post, header, nil, checkDuplicateProduct)
		if err != nil {
			log.WithError(err).Errorf("Error when create duplicated product name")
			return nil, ginext.NewError(http.StatusBadRequest, "Tạo sản phẩm không được trùng tên")
		}

		// call create multi product
		listProductFast := model.CreateProductFast{
			BusinessID:      req.BusinessID,
			ListProductFast: req.ListProductFast,
		}

		productFastResponse, err := s.CreateMultiProduct(ctx, header, listProductFast)
		if err == nil {
			lstOrderItem = productFastResponse.Data
		}
	}

	// append ListOrderItem from request to listOrderItem received from createMultiProduct
	for _, v := range lstOrderItem {
		if v.SkuID == uuid.Nil {
			log.WithError(err).Error("Error when received from createMultiProduct")
			return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
		req.ListOrderItem = append(req.ListOrderItem, v)
	}

	// check listOrderItem empty
	if len(req.ListOrderItem) == 0 {
		log.Error("ListOrderItem mustn't empty")
		return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Đơn hàng phải có ít nhất 1 sản phẩm")
	}

	// Check valid order item
	log.WithField("list order item", req.ListOrderItem).Info("Request Order Item")

	// check can pick quantity
	rCheck, err := utils.CheckCanPickQuantity(req.UserID.String(), req.ListOrderItem, nil)
	if err != nil {
		log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	} else {
		if rCheck.Status != utils.STATUS_SUCCESS {
			return rCheck, nil
		}
	}

	mapSku := make(map[string]model.CheckValidStockResponse)
	for _, v := range rCheck.ItemsInfo {
		mapSku[v.ID.String()] = v
	}
	// Tính tổng tiền
	for i, v := range req.ListOrderItem {
		itemTotalAmount := 0.0
		if v.ProductSellingPrice > 0 {
			itemTotalAmount = v.ProductSellingPrice * v.Quantity
		} else {
			itemTotalAmount = v.ProductNormalPrice * v.Quantity
		}
		req.ListOrderItem[i].TotalAmount = math.Round(itemTotalAmount)
		orderGrandTotal += req.ListOrderItem[i].TotalAmount
	}

	// Set buyer_id from Create Method request
	buyerID := uuid.UUID{}
	switch req.CreateMethod {
	case utils.BUYER_CREATE_METHOD:
		buyerID = req.UserID
		if info.Data.Business.DeliveryFee == 0 || (info.Data.Business.DeliveryFee > 0 && orderGrandTotal >= info.Data.Business.MinPriceFreeShip && info.Data.Business.MinPriceFreeShip > 0) {
			deliveryFee = 0
		} else {
			deliveryFee = info.Data.Business.DeliveryFee
		}
		break
	case utils.SELLER_CREATE_METHOD:
		tUser, err := s.GetUserList(ctx, req.BuyerInfo.PhoneNumber, "")
		if err != nil {
			log.WithError(err).Error("Error when get user info from phone number of buyer info")
			return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
		}
		if len(tUser) > 0 {
			buyerID = tUser[0].ID
		}
		deliveryFee = req.DeliveryFee
		break
	default:
		log.WithError(err).Error("Error when Create method, expected: [buyer, seller]")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	if req.DeliveryMethod != nil && *req.DeliveryMethod == utils.DELIVERY_METHOD_BUYER_PICK_UP {
		deliveryFee = 0
	} else {
		if deliveryFee != req.DeliveryFee {
			log.WithError(err).Error("Error when get check valid delivery fee")
			return nil, ginext.NewError(http.StatusBadRequest, "Cửa hàng đã cập nhật phí vận chuyển mới, vui lòng kiểm tra lại")
		}
	}

	// Check valid grand total
	if req.OtherDiscount > (req.OrderedGrandTotal + req.DeliveryFee - req.PromotionDiscount) {
		log.WithError(err).Error("Error when get check valid delivery fee")
		return nil, ginext.NewError(http.StatusBadRequest, "Số tiền chiết khấu không được lớn hơn số tiền phải trả")
	}

	// Check Promotion Code
	if req.PromotionCode != "" {
		promotion, err := s.ProcessPromotion(ctx, *req.BusinessID, req.PromotionCode, orderGrandTotal, info.Data.Contact.ID, req.UserID, true)
		if err != nil {
			log.WithField("req process promotion", req).Errorf("Get promotion error: %v", err.Error())
			return nil, ginext.NewError(http.StatusBadRequest, "Không đủ điều kiện để sử dụng mã khuyến mãi")
		}
		promotionDiscount = promotion.ValueDiscount
	}

	grandTotal = orderGrandTotal + deliveryFee - promotionDiscount - req.OtherDiscount
	if grandTotal < 0 {
		grandTotal = 0
	}

	// Check số tiền request lên và số tiền trong db có khớp
	if math.Round(req.OrderedGrandTotal) != math.Round(orderGrandTotal) ||
		math.Round(req.PromotionDiscount) != math.Round(promotionDiscount) ||
		math.Round(req.DeliveryFee) != deliveryFee ||
		math.Round(req.GrandTotal) != math.Round(grandTotal) {
		return nil, ginext.NewError(http.StatusBadRequest, "Số tiền không hợp lệ")
	}

	// check buyer received or not
	checkCompleted := utils.ORDER_COMPLETED
	if req.BuyerReceived {
		req.State = utils.ORDER_STATE_COMPLETE
	}

	// if req.State == utils.ORDER_STATE_COMPLETE {
	// 	checkCompleted = utils.FAST_ORDER_COMPLETED
	// }

	order := model.Order{
		BusinessID:        *req.BusinessID,
		ContactID:         info.Data.Contact.ID,
		PromotionCode:     req.PromotionCode,
		PromotionDiscount: promotionDiscount,
		DeliveryFee:       deliveryFee,
		OrderedGrandTotal: orderGrandTotal,
		GrandTotal:        grandTotal,
		State:             req.State,
		PaymentMethod:     strings.ToLower(req.PaymentMethod),
		DeliveryMethod:    *req.DeliveryMethod,
		Note:              req.Note,
		CreateMethod:      req.CreateMethod,
		BuyerId:           &buyerID,
		OtherDiscount:     req.OtherDiscount,
		Email:             req.Email,
	}

	req.BuyerInfo.PhoneNumber = utils.ConvertVNPhoneFormat(req.BuyerInfo.PhoneNumber)

	order.CreatorID = req.UserID

	buyerInfo, err := json.Marshal(req.BuyerInfo)
	if err != nil {
		log.WithError(err).Error("Error when parse buyerInfo")
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	order.BuyerInfo.RawMessage = buyerInfo

	log.Info("Begin work with DB")
	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	log.Info("Start DB transaction")

	// create order
	order, err = s.repo.CreateOrder(ctx, order, tx)
	if err != nil {
		log.WithError(err).Error("Error when CreateOrder")
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}
	log.WithField("order created", order).Info("Finish createOrder")

	if err = s.CreateOrderTracking(ctx, order, tx); err != nil {
		log.WithError(err).Error("Create order tracking error")
		return res, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	for _, orderItem := range req.ListOrderItem {
		orderItem.OrderID = order.ID
		orderItem.CreatorID = order.CreatorID
		if _, ok := mapSku[orderItem.SkuID.String()]; ok {
			orderItem.UOM = mapSku[orderItem.SkuID.String()].Uom
			orderItem.HistoricalCost = mapSku[orderItem.SkuID.String()].HistoricalCost
		}
		if orderItem.ProductSellingPrice != 0 {
			orderItem.Price = orderItem.ProductSellingPrice
		} else {
			orderItem.Price = orderItem.ProductNormalPrice
		}
		tm, err := s.repo.CreateOrderItem(ctx, orderItem, tx)
		if err != nil {
			log.WithError(err).Errorf("Error when CreateOrderItem: %v", err.Error())
			return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
		order.OrderItem = append(order.OrderItem, tm)
	}

	debit := model.Debit{}
	if req.Debit != nil {
		debit = *req.Debit
	}

	tx.Commit()
	go s.CountCustomer(ctx, order)
	go s.OrderProcessing(ctx, order, debit, checkCompleted, *req.BuyerInfo)
	go s.UpdateContactUser(ctx, order, order.CreatorID)
	go s.CheckCompletedTutorialCreate(context.Background(), order.CreatorID) // tutorial flow

	// push consumer to complete order mission
	go CompletedOrderMission(ctx, order)

	return order, nil
}

func (s *OrderService) ProcessPromotion(ctx context.Context, businessId uuid.UUID, promotionCode string, orderGrandTotal float64, contactID uuid.UUID, currentUser uuid.UUID, isUse bool) (model.Promotion, error) {
	log := logger.WithCtx(ctx, "OrderService.ProcessPromotion")

	type ProcessPromotionRequest struct {
		BusinessId    uuid.UUID `json:"business_id" valid:"Required"`
		PromotionCode string    `json:"promotion_code" valid:"Required"`
		GrandTotal    float64   `json:"grand_total" valid:"Required"`
		IsUse         bool      `json:"is_use" valid:"Required"`
		ContactID     uuid.UUID `json:"contact_id,omitempty"`
	}

	req := ProcessPromotionRequest{
		BusinessId:    businessId,
		PromotionCode: promotionCode,
		GrandTotal:    orderGrandTotal,
		IsUse:         isUse,
		ContactID:     contactID,
	}

	header := map[string]string{}
	header["x-user-id"] = currentUser.String()

	type PromotionResponse struct {
		Data model.Promotion `json:"data"`
	}
	promotion := PromotionResponse{}
	bodyResponse, _, err := common.SendRestAPI(conf.LoadEnv().MSPromotionManagement+"/api/v2/promotion/process", rest.Post, header, nil, req)
	if err != nil {
		log.WithError(err).Errorf("Error when Get promotion info error: %v", err.Error())
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	if err = json.Unmarshal([]byte(bodyResponse), &promotion); err != nil {
		log.WithError(err).Errorf("Error when unmarshal promotion: %v", err.Error())
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	return promotion.Data, nil
}

func (s *OrderService) OrderProcessing(ctx context.Context, order model.Order, debit model.Debit, checkCompleted string, buyerInfo model.BuyerInfo) (err error) {
	log := logger.WithCtx(ctx, "OrderService.OrderProcessing")
	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()
	//TODO--------Update Business custom_field--------------------------------------------------------------START
	allState := []string{utils.ORDER_STATE_WAITING_CONFIRM, utils.ORDER_STATE_DELIVERING, utils.ORDER_STATE_COMPLETE, utils.ORDER_STATE_CANCEL}

	// get seller_id from business_id
	uhb, err := utils.GetUserHasBusiness("", order.BusinessID.String())
	if err != nil {
		log.WithError(err).Errorf("Error when get user has busines: %v", err.Error())
		return
	}
	if len(uhb) == 0 {
		log.Error("Error: Empty user has business info")
		return
	}

	for _, state := range allState {
		countState := s.repo.CountOneStateOrder(context.Background(), order.BusinessID, state, tx)
		customFieldName := ""
		switch state {
		case utils.ORDER_STATE_WAITING_CONFIRM:
			customFieldName = "order_waiting_confirm_count"
		case utils.ORDER_STATE_DELIVERING:
			customFieldName = "order_delivering_count"
		case utils.ORDER_STATE_COMPLETE:
			customFieldName = "order_complete_count"
		case utils.ORDER_STATE_CANCEL:
			customFieldName = "order_cancel_count"
		}
		s.UpdateBusinessCustomField(ctx, order.BusinessID, customFieldName, strconv.Itoa(countState))
	}

	//TODO--------Update Business custom_field--------------------------------------------------------------END

	// send email
	go s.PushConsumerSendEmail(context.Background(), order.ID.String(), order.State)

	switch order.State {

	case utils.ORDER_STATE_WAITING_CONFIRM:
		go s.SendNotificationV2(context.Background(), uhb[0].UserID, utils.NOTIFICATION_ENTITY_KEY_ORDER, order.State+"_v2", fmt.Sprintf(utils.NOTI_CONTENT_WAITING_CONFIRM, utils.StrDelimitForSum(order.OrderedGrandTotal, "đ")))
		go s.ReminderProcessOrderV2(context.Background(), order.ID, uhb[0].UserID, utils.ORDER_STATE_WAITING_CONFIRM, fmt.Sprintf(utils.NOTI_CONTENT_REMINDER_WAITING_CONFIRM, order.OrderNumber))
		go utils.SendAutoChatWhenUpdateOrder(utils.UUID(order.BuyerId).String(), utils.MESS_TYPE_UPDATE_ORDER, order.OrderNumber, fmt.Sprintf(utils.MESS_ORDER_WAITING_CONFIRM, order.OrderNumber))
		break
	case utils.ORDER_STATE_DELIVERING:
		go s.ReminderProcessOrderV2(context.Background(), order.ID, uhb[0].UserID, utils.ORDER_STATE_DELIVERING, fmt.Sprintf(utils.NOTI_CONTENT_REMINDER_DELIVERING, order.OrderNumber, utils.StrDelimitForSum(order.OrderedGrandTotal, "đ"), buyerInfo.Name))
		go utils.SendAutoChatWhenUpdateOrder(utils.UUID(order.BuyerId).String(), utils.MESS_TYPE_UPDATE_ORDER, order.OrderNumber, fmt.Sprintf(utils.MESS_ORDER_DELIVERING, order.OrderNumber))
		go s.UpdateStock(context.Background(), order, "order_delivering")
		break
	case utils.ORDER_STATE_COMPLETE:
		//TODO--------Update Business custom_field Revenue -------------------------------------------------------------START
		revenue, err := s.repo.RevenueBusiness(ctx, model.RevenueBusinessParam{
			BusinessID: order.BusinessID.String(),
		}, tx)
		if err == nil {
			strSumGrandTotal := fmt.Sprintf("%.0f", revenue.SumGrandTotal)
			s.UpdateBusinessCustomField(ctx, order.BusinessID, "business_revenue", strSumGrandTotal)
		}

		//----------------------------------------------------------------------------------------------------

		// Create Business transaction
		cateIDSell, _ := uuid.Parse(utils.CATEGORY_SELL)
		businessTransaction := model.BusinessTransaction{
			ID:              uuid.New(),
			CreatorID:       uhb[0].UserID,
			BusinessID:      order.BusinessID,
			Day:             time.Now().UTC(),
			Amount:          order.GrandTotal,
			Currency:        "VND",
			TransactionType: "in",
			Status:          "paid",
			Action:          "create",
			Description:     "Đơn hàng " + order.OrderNumber,
			CategoryID:      cateIDSell,
			CategoryName:    "Bán hàng",
			LatestSyncTime:  time.Now().UTC().Format("2006-01-02T15:04:05Z"),
			OrderNumber:     order.OrderNumber,
			Table:           "income",
		}

		err = s.CreateBusinessTransaction(ctx, businessTransaction)
		if err != nil {
			log.WithError(err).Errorf("Error when create business transaction: " + err.Error())
			return err
		}

		if debit.BuyerPay != nil && *debit.BuyerPay < order.GrandTotal {
			contactTransaction := model.ContactTransaction{
				ID:              uuid.New(),
				CreatorID:       uhb[0].UserID,
				BusinessID:      order.BusinessID,
				Amount:          order.GrandTotal - *debit.BuyerPay,
				ContactID:       order.ContactID,
				Currency:        "VND",
				TransactionType: "in",
				Status:          "create",
				Action:          "create",
				Description:     debit.Note,
				StartTime:       time.Now().UTC(),
				Images:          debit.Images,
				LatestSyncTime:  time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				OrderNumber:     order.OrderNumber,
				Table:           "lent",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}
			err = s.CreateContactTransaction(ctx, contactTransaction)
			if err != nil {
				log.WithError(err).Errorf("Error when contact transaction: " + err.Error())
				return err
			}
		}
		go PushConsumer(context.Background(), order.OrderItem, utils.TOPIC_UPDATE_SOLD_QUANTITY)
		go s.CreatePo(context.Background(), order, checkCompleted, utils.PO_OUT)
		//if err = s.CreatePo(ctx, order, checkCompleted); err != nil {
		//	log.WithError(err).Errorf("Error when call func CreatePo: " + err.Error())
		//}
		break
	case utils.ORDER_STATE_CANCEL:
		go utils.SendAutoChatWhenUpdateOrder(utils.UUID(order.BuyerId).String(), utils.MESS_TYPE_UPDATE_ORDER, order.OrderNumber, fmt.Sprintf(utils.MESS_ORDER_CANCELED, order.OrderNumber))
		break
	default:
		break
	}

	return nil
}

func (s *OrderService) CountCustomer(ctx context.Context, order model.Order) {
	log := logger.WithCtx(ctx, "OrderService.CountCustomer")

	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	_, countCustomer, err := s.repo.GetContactHaveOrder(ctx, order.BusinessID, tx)
	if err != nil {
		log.WithError(err).Error("Fail to get contact have order")
		return
	}

	s.UpdateBusinessCustomField(ctx, order.BusinessID, "customer_count", strconv.Itoa(countCustomer))
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	tx.Commit()
}

func (s *OrderService) UpdateBusinessCustomField(ctx context.Context, businessId uuid.UUID, customField string, customValue string) {

	request := model.CustomFieldsRequest{
		BusinessID:   businessId,
		CustomFields: postgres.Hstore{customField: utils.String(customValue)},
	}
	PushConsumer(ctx, request, utils.TOPIC_UPDATE_CUSTOM_FIELDS)
}

func PushConsumer(ctx context.Context, value interface{}, topic string) {
	log := logger.WithCtx(ctx, "PushConsumer")

	s, _ := json.Marshal(value)
	_, err := utils.PushConsumer(utils.ConsumerRequest{
		Topic: topic,
		Body:  string(s),
	})
	log.WithError(err).Error("PushConsumer topic: " + topic + " body: " + string(s))
	if err != nil {
		log.WithError(err).Error("Fail to push consumer " + topic + ": %")
	}
}

func CompletedOrderMission(ctx context.Context, order model.Order) {
	log := logger.WithCtx(ctx, "CompletedOrderMission")
	var userID uuid.UUID
	if order.CreateMethod == utils.SELLER_CREATE_METHOD {
		userID = order.CreatorID
	} else {
		userHasBusiness, err := utils.GetUserHasBusiness("", order.BusinessID.String())
		if err != nil {
			log.WithError(err).Error("Fail to GetUserHasBusiness")
			return
		}
		userID = userHasBusiness[0].UserID
	}

	req := map[string]string{
		"mission_type": "completed_order",
		"user_id":      userID.String(),
	}

	PushConsumer(ctx, req, utils.TOPIC_PROCESS_MISSION)
}

func (s *OrderService) UpdateContactUser(ctx context.Context, order model.Order, user_id uuid.UUID) (err error) {
	log := logger.WithCtx(ctx, "OrderService.UpdateContactUser")

	var buyerInfo *model.BuyerInfo
	values, _ := order.BuyerInfo.MarshalJSON()
	err = json.Unmarshal(values, &buyerInfo)
	if err != nil {
		log.WithError(err).Error("Error when unmarshal update user contact")
	}
	type UserContact struct {
		UserID      uuid.UUID `json:"user_id"`
		PhoneNumber string    `json:"phone_number"`
		Address     string    `json:"address"`
	}

	header := make(map[string]string)
	header["x-user-id"] = user_id.String()
	_, _, err = common.SendRestAPI(conf.LoadEnv().MSUserManagement+"/api/user-contact", rest.Post, header, nil, &UserContact{
		UserID:      user_id,
		PhoneNumber: buyerInfo.PhoneNumber,
		Address:     buyerInfo.Address,
	})
	if err != nil {
		log.WithError(err).Error("Error when update user contact")
		return err
	} else {
		log.WithError(err).Error("Update profile user contact")
	}

	return nil
}

func (s *OrderService) CreateOrderTracking(ctx context.Context, req model.Order, tx *gorm.DB) error {
	logger.WithCtx(ctx, "OrderService.CreateOrderTracking").Info()

	orderTracking := model.OrderTracking{
		OrderID: req.ID,
		State:   req.State,
	}

	return s.repo.CreateOrderTracking(ctx, orderTracking, tx)
}

func (s *OrderService) PushConsumerSendEmail(ctx context.Context, id string, state string) {
	logger.WithCtx(ctx, "OrderService.PushConsumerSendEmail").Info()

	request := model.SendEmailRequest{
		ID:       id,
		State:    state,
		UserRole: strconv.Itoa(utils.ADMIN_ROLE),
	}
	PushConsumer(ctx, request, utils.TOPIC_SEND_EMAIL_ORDER)
}

func (s *OrderService) CreateBusinessTransaction(ctx context.Context, req model.BusinessTransaction) error {
	log := logger.WithCtx(ctx, "OrderService.CreateBusinessTransaction")

	header := make(map[string]string)
	header["x-user-id"] = req.CreatorID.String()

	// 22-01-2022 - thanhvc - skip process complete mission case_book
	// add more header skip-complete-mission = true, when call api to ms-transaction
	// it will skip processing complete mission cash_book
	header["skip-complete-mission"] = "true"

	_, _, err := common.SendRestAPI(conf.LoadEnv().MSTransactionManagement+"/api/business-transaction/v2/create", rest.Post, header, nil, req)
	if err != nil {
		log.WithError(err).Error("Error when create business transaction")
		return err
	}
	return nil
}

func (s *OrderService) CreateContactTransaction(ctx context.Context, req model.ContactTransaction) error {
	log := logger.WithCtx(ctx, "OrderService.CreateContactTransaction")

	header := make(map[string]string)
	header["x-user-id"] = req.CreatorID.String()
	_, _, err := common.SendRestAPI(conf.LoadEnv().MSTransactionManagement+"/api/v2/contact-transaction/create", rest.Post, header, nil, req)
	if err != nil {
		log.WithError(err).Error("Error when create contact transaction")
		return err
	}
	return nil
}

func (s *OrderService) CreatePo(ctx context.Context, order model.Order, checkCompleted string, poType string) (err error) {
	log := logger.WithCtx(ctx, "OrderService.CreatePo")
	// Make data for push consumer
	reqCreatePo := model.PurchaseOrderRequest{
		PoType:        poType,
		Note:          "Đơn hàng " + order.OrderNumber,
		ContactID:     order.ContactID,
		TotalDiscount: order.OtherDiscount,
		BusinessID:    order.BusinessID,
		PoDetails:     nil,
		Option:        checkCompleted,
	}
	skuIDs, err := utils.CheckSkuHasStock(order.CreatorID.String(), order.OrderItem)
	if err != nil {
		log.WithError(err).Error("error when CheckSkuHasStock")
		return err
	}
	if len(skuIDs) > 0 {
		tmp := strings.Join(skuIDs, ",")
		for _, v := range order.OrderItem {
			req := model.CountQuantityInOrderRequest{
				BusinessID: order.BusinessID,
				SkuID:      v.SkuID,
				States:     []string{utils.ORDER_STATE_DELIVERING},
			}
			countQuantityInOrder, _ := s.repo.GetCountQuantityInOrder(ctx, req, nil)
			if strings.Contains(tmp, v.SkuID.String()) {
				reqCreatePo.PoDetails = append(reqCreatePo.PoDetails, model.PoDetail{
					SkuID:              v.SkuID,
					Pricing:            v.TotalAmount / v.Quantity,
					Quantity:           v.Quantity,
					DeliveringQuantity: &countQuantityInOrder.Sum,
				})
			}
		}
		go PushConsumer(ctx, reqCreatePo, utils.TOPIC_CREATE_PO_V2)
	}
	return nil
}

func (s *OrderService) GetContactHaveOrder(ctx context.Context, req model.OrderParam) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetContactHaveOrder")

	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	contactIds, _, err := s.repo.GetContactHaveOrder(ctx, uuid.MustParse(req.BusinessID), tx)
	if err != nil {
		log.WithError(err).Error("Fail to get contact have order")
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, "Fail to get contact have order: "+err.Error())
	}

	lstContact, err := s.GetContactList(ctx, contactIds)
	if err != nil {
		log.WithError(err).Error("Fail to get contact list")
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, "Fail to get contact list: "+err.Error())
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	tx.Commit()
	return lstContact, nil
}

func (s *OrderService) GetContactList(ctx context.Context, contactIDs string) (res []model.Contact, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetContactList")

	queryParam := make(map[string]string)
	queryParam["ids"] = contactIDs

	bodyBusiness, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/contacts", rest.Get, nil, queryParam, nil)
	if err != nil {
		log.WithError(err).Error("Fail to get contact list")
		return res, err
	}
	tmpResContact := new(struct {
		Data []model.Contact `json:"data"`
	})
	if err = json.Unmarshal([]byte(bodyBusiness), &tmpResContact); err != nil {
		log.WithError(err).Error("Fail to get contact list")
		return res, err
	}
	return tmpResContact.Data, nil
}

func (s *OrderService) SendNotification(ctx context.Context, userId uuid.UUID, entityKey string, state string, content string) {
	log := logger.WithCtx(ctx, "OrderService.SendNotification")

	notiRequest := model.SendNotificationRequest{
		UserID:         userId,
		EntityKey:      entityKey,
		StateValue:     state,
		Language:       "vi",
		ContentReplace: content,
	}

	_, _, err := common.SendRestAPI(conf.LoadEnv().MSNotificationManagement+"/api/notification/send-notification", rest.Post, nil, nil, notiRequest)
	if err != nil {
		log.WithError(err).Error("Send noti " + entityKey + "_" + state + " error")
	} else {
		log.WithError(err).Error("Send noti " + entityKey + "_" + state + " successfully")
	}
}

func (s *OrderService) UpdateStock(ctx context.Context, order model.Order, trackingType string) (err error) {
	log := logger.WithCtx(ctx, "OrderService.UpdateStock").WithField("OrderService.UpdateStock", order.OrderItem)

	// Make data for push consumer
	reqUpdateStock := model.CreateStockRequest{
		TrackingType:   trackingType,
		IDTrackingType: order.OrderNumber,
		BusinessID:     order.BusinessID,
	}
	tResToJson, _ := json.Marshal(order)
	if err = json.Unmarshal(tResToJson, &reqUpdateStock.TrackingInfo); err != nil {
		log.WithError(err).Error("Error when marshal parse response to json when create stock")
	} else {
		for _, v := range order.OrderItem {
			req := model.CountQuantityInOrderRequest{
				BusinessID: order.BusinessID,
				SkuID:      v.SkuID,
				States:     []string{utils.ORDER_STATE_DELIVERING},
			}
			countQuantityInOrder, _ := s.repo.GetCountQuantityInOrder(ctx, req, nil)
			reqUpdateStock.ListStock = append(reqUpdateStock.ListStock, model.StockRequest{
				SkuID:              v.SkuID,
				QuantityChange:     v.Quantity,
				DeliveringQuantity: countQuantityInOrder.Sum,
			})
		}
		go PushConsumer(ctx, reqUpdateStock, utils.TOPIC_UPDATE_STOCK_V2)
	}
	return nil
}

func (s *OrderService) ReminderProcessOrder(ctx context.Context, orderId uuid.UUID, sellerID uuid.UUID, stateCheck string) {
	log := logger.WithCtx(ctx, "OrderService.ReminderProcessOrder")

	time.AfterFunc(60*time.Minute, func() {
		// Create transaction
		var cancel context.CancelFunc
		tx, cancel := s.repo.DBWithTimeout(ctx)
		tx = tx.Begin()
		defer func() {
			tx.Rollback()
			cancel()
		}()

		order, err := s.repo.GetOneOrder(ctx, orderId.String(), tx)
		if err != nil {
			log.WithError(err).Error("ReminderProcessOrder get order " + orderId.String() + " error")
		}

		if order.State == stateCheck {
			s.SendNotification(ctx, sellerID, utils.NOTIFICATION_ENTITY_KEY_ORDER, "reminder_"+order.State, order.OrderNumber)
		}

		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()

		tx.Commit()
	})
}

func (s *OrderService) SendEmailOrder(ctx context.Context, req model.SendEmailRequest) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.SendEmailOrder")

	userRoles, _ := strconv.Atoi(req.UserRole)
	if !((userRoles&utils.ADMIN_ROLE > 0) || (userRoles&utils.ADMIN_ROLE == utils.ADMIN_ROLE)) {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, err.Error())
	}

	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	order, err := s.repo.GetOneOrder(ctx, req.ID, tx)
	if err != nil || order.Email == "" {
		return
	}

	cfg := sendinblue.NewConfiguration()
	cfg.AddDefaultHeader("api-key", conf.LoadEnv().ApiKeySendinblue)
	cfg.AddDefaultHeader("partner-key", conf.LoadEnv().ApiKeySendinblue)
	sib := sendinblue.NewAPIClient(cfg)

	var orderItems []model.OrderItemForSendEmail
	for _, item := range order.OrderItem {
		var orderItem = model.OrderItemForSendEmail{
			//ProductID:           item.ProductID,
			ProductName:         item.ProductName,
			Quantity:            item.Quantity,
			TotalAmount:         item.TotalAmount,
			SkuID:               item.SkuID,
			SkuName:             item.SkuName,
			SkuCode:             item.SkuCode,
			Note:                item.Note,
			UOM:                 item.UOM,
			ProductNormalPrice:  utils.StrDelimitForSum(item.ProductNormalPrice, ""),
			ProductSellingPrice: utils.StrDelimitForSum(item.ProductSellingPrice, ""),
		}
		if len(item.ProductImages) > 0 {
			orderItem.ProductImages = utils.ResizeImage(item.ProductImages[0], 80, 80)
		}
		orderItems = append(orderItems, orderItem)
	}

	tmpBuyerInfo := order.BuyerInfo.RawMessage
	buyerInfo := model.BuyerInfo{}
	if err = json.Unmarshal(tmpBuyerInfo, &buyerInfo); err != nil {
		log.WithError(err).Error("Fail to Unmarshal contact")
		return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	businessInfo, err := s.GetDetailBusiness(ctx, order.BusinessID.String())
	if err != nil {
		log.WithError(err).Error("Fail to get business detail")
		return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	to := sendinblue.SendSmtpEmailTo{
		Email: order.Email,
		Name:  buyerInfo.Name,
	}

	tParams := map[string]interface{}{
		// customer
		"NAME_CUSTOMER":    buyerInfo.Name,
		"ADDRESS_CUSTOMER": buyerInfo.Address,
		"PHONE_CUSTOMER":   utils.RevertBeginPhone(buyerInfo.PhoneNumber),
		"EMAIL_CUSTOMER":   order.Email,
		// order
		"ORDER_NUMBER":        order.OrderNumber,
		"ORDERED_GRAND_TOTAL": utils.StrDelimitForSum(order.OrderedGrandTotal, ""),
		"PROMOTION_DISCOUNT":  utils.StrDelimitForSum(order.PromotionDiscount, ""),
		"OTHER_DISCOUNT":      utils.StrDelimitForSum(order.OtherDiscount, ""),
		"DELIVERY_FEE":        utils.StrDelimitForSum(order.DeliveryFee, ""),
		"GRAND_TOTAL":         utils.StrDelimitForSum(order.GrandTotal, ""),
		"PAYMENT_METHOD":      order.PaymentMethod,
		"DELIVERY_METHOD":     order.DeliveryMethod,
		"ORDER_ITEMS":         orderItems,
		"TOTAL_ITEMS":         len(orderItems),
		// seller
		"NAME_BUSINESS":    businessInfo.Name,
		"ADDRESS_BUSINESS": businessInfo.Address,
		"PHONE_BUSINESS":   utils.RevertBeginPhone(businessInfo.PhoneNumber),
		"DOMAIN_BUSINESS":  businessInfo.Domain,
	}
	if order.CreatorID != uuid.Nil {
		tParams["QRCODE"] = "https://" + businessInfo.Domain + "/order/" + order.OrderNumber + "?required-login=true"
	} else {
		tParams["QRCODE"] = "https://" + businessInfo.Domain + "/order/" + order.OrderNumber
	}
	avatarBusiness := businessInfo.Avatar
	if businessInfo.Avatar == "" {
		avatarBusiness = utils.AVATAR_BUSINESS_DEFAULT
	}
	tParams["AVATAR_BUSINESS"] = utils.ResizeImage(avatarBusiness, 128, 128)

	if len(businessInfo.Background) > 0 && businessInfo.Background[0] != "" {
		tParams["BACKGROUND"] = businessInfo.Background[0]
	} else {
		tParams["BACKGROUND"] = "https://" + businessInfo.Domain + "/_next/static/image/assets/default-cover.9b114bb9b20bbfc62de02a837e18e07a.webp"
	}

	switch req.State {
	case utils.ORDER_STATE_WAITING_CONFIRM:
		tParams["STATE"] = "đã đặt hàng thành công"
		break
	case utils.ORDER_STATE_DELIVERING:
		tParams["STATE"] = "đã được xác nhận"
		break
	case utils.ORDER_STATE_CANCEL:
		tParams["STATE"] = "đã hủy"
		break
	case utils.ORDER_STATE_COMPLETE:
		tParams["STATE"] = "đã hoàn thành"
		break
	case utils.ORDER_STATE_UPDATE:
		tParams["STATE"] = "đã được cập nhật"
		break
	default:
		return nil, nil
	}

	var params interface{} = tParams
	body := sendinblue.SendSmtpEmail{
		Sender: &sendinblue.SendSmtpEmailSender{
			Name:  businessInfo.Name, //
			Email: utils.DefaultFromEmail,
		},
		To:     []sendinblue.SendSmtpEmailTo{to},
		Params: &params,
	}

	switch req.State {
	case utils.ORDER_STATE_WAITING_CONFIRM:
		body.TemplateId = int64(utils.SEND_EMAIL_WAITING_CONFIRM)
		break
	case utils.ORDER_STATE_DELIVERING:
		body.TemplateId = int64(utils.SEND_EMAIL_DELIVERING)
		break
	case utils.ORDER_STATE_CANCEL:
		body.TemplateId = int64(utils.SEND_EMAIL_COMPLETE)
		break
	case utils.ORDER_STATE_COMPLETE:
		body.TemplateId = int64(utils.SEND_EMAIL_CANCEL)
		break
	case utils.ORDER_STATE_UPDATE:
		body.TemplateId = int64(utils.ORDER_EMAIL_UPDATE)
		break
	default:
		return nil, nil
	}

	obj, resp, err := sib.TransactionalEmailsApi.SendTransacEmail(ctx, body)
	if err != nil {
		fmt.Println("Error in TransactionalEmailsApi->SendTransacEmail ", err.Error())
		return nil, err
	}
	fmt.Println("SendTransacEmail, response:", resp, "SendTransacEmail object", obj)

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	tx.Commit()

	return req.State, err
}

func (s *OrderService) GetDetailBusiness(ctx context.Context, businessID string) (res model.BusinessMainInfo, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetDetailBusiness")

	bodyBusiness, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/business/"+businessID, rest.Get, nil, nil, nil)
	if err != nil {
		log.WithError(err).Error("Error when GetDetailBusiness")
		return res, err
	}
	tmpResBusiness := new(struct {
		Data model.BusinessMainInfo `json:"data"`
	})
	if err = json.Unmarshal([]byte(bodyBusiness), &tmpResBusiness); err != nil {
		log.WithError(err).Error("Cannot unmarshal BusinessMainInfo")
		return res, ginext.NewError(http.StatusBadRequest, "Cannot unmarshal BusinessMainInfo")
	}
	return tmpResBusiness.Data, nil
}

func (s *OrderService) UpdateEmailForOrderRecent(ctx context.Context, req model.UpdateEmailOrderRecentRequest) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.UpdateEmailForOrderRecent")

	// get order recent
	order, err := s.repo.GetOneOrderRecent(ctx, req.UserID.String(), nil)
	if err != nil {
		log.WithError(err).Error("Error when call func GetOneOrderRecent")
		return nil, err
	}

	// update order
	order.Email = req.Email
	order.UpdaterID = req.UserID

	res, err = s.repo.UpdateOrder(ctx, order, nil)
	if err != nil {
		log.WithError(err).Error("Error when call func UpdateOrder")
		return nil, err
	}
	return res, nil
}

func (s *OrderService) UpdateOrder(ctx context.Context, req model.OrderUpdateBody, userRole string) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.UpdateOrder")

	order, err := s.repo.GetOneOrder(ctx, req.ID.String(), nil)
	if err != nil {
		log.WithError(err).Errorf("Error when GetOneOrder")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	// Check permission
	if err = utils.CheckPermission(ctx, req.UpdaterID.String(), order.BusinessID.String(), userRole); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	if req.State != nil && order.State == *req.State {
		log.WithError(err).Errorf("Error when State not change")
		return nil, ginext.NewError(http.StatusBadRequest, "Error when State not change")
	}

	preOrderState := order.State

	if req.State != nil && *req.State == utils.ORDER_STATE_DELIVERING && preOrderState == utils.ORDER_STATE_WAITING_CONFIRM {
		if rCheck, err := utils.CheckCanPickQuantityV4(order.CreatorID.String(), order.OrderItem, order.BusinessID.String(), nil, order.CreateMethod); err != nil {
			log.WithError(err).Errorf("Error when CheckValidOrderItems from MS Product")
			return nil, ginext.NewError(http.StatusBadRequest, err.Error())
		} else {
			if rCheck.Status == utils.STATUS_SKU_NOT_FOUND {
				log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
				return nil, ginext.NewError(http.StatusBadRequest, "Không tìm thấy sản phẩm trong cửa hàng")
			}
			if rCheck.Status != utils.STATUS_SUCCESS {
				log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
				return rCheck, nil
			}
		}
	}

	common.Sync(req, &order)

	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	res, err = s.repo.UpdateOrder(ctx, order, tx)
	if err != nil {
		log.WithError(err).Errorf("Cannot update order")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	if err = s.CreateOrderTracking(ctx, order, tx); err != nil {
		log.WithError(err).Errorf("Create order tracking error")
	}

	tx.Commit()

	if req.State != nil && *req.State == utils.ORDER_STATE_CANCEL && preOrderState == utils.ORDER_STATE_COMPLETE {
		go s.OrderCancelProcessing(context.Background(), order, tx)
	} else {
		debit := model.Debit{}
		if req.Debit != nil {
			debit = *req.Debit
		}

		buyerInfo := model.BuyerInfo{}
		if err := json.Unmarshal(order.BuyerInfo.RawMessage, &buyerInfo); err != nil {
			log.WithError(err).Errorf("Cannot unmarshal buyerInfo")
			return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
		}
		if err := s.OrderProcessing(ctx, order, debit, utils.ORDER_COMPLETED, buyerInfo); err != nil {
			log.WithError(err).Error("Fail to create transaction")
			return nil, err
		}
	}

	if preOrderState == utils.ORDER_STATE_DELIVERING && req.State != nil && *req.State == utils.ORDER_STATE_CANCEL {
		go s.UpdateStock(context.Background(), order, "order_cancelled_when_delivering")
	}

	// log history UpdateOrder ver1
	go func() {
		history := model.History{
			BaseModel: model.BaseModel{
				CreatorID: order.UpdaterID,
			},
			ObjectID:    order.ID,
			ObjectTable: utils.TABLE_ORDER,
			Action:      utils.ACTION_UPDATE_ORDER,
			Description: utils.ACTION_UPDATE_ORDER + " in UpdateOrder func - OrderService",
			Worker:      order.UpdaterID.String(),
		}

		dataOrder, err := json.Marshal(order)
		if err != nil {
			log.WithError(err).Error("Error when parse order in  in UpdateOrder func - OrderService")
			return
		}
		history.Data.RawMessage = dataOrder

		requestData, err := json.Marshal(req)
		if err != nil {
			log.WithError(err).Error("Error when parse order request in UpdateOrder - OrderService")
			return
		}
		history.DataRequest.RawMessage = requestData

		s.historyService.LogHistory(context.Background(), history, nil)
	}()

	return res, err
}

func (s *OrderService) OrderCancelProcessing(ctx context.Context, order model.Order, tx *gorm.DB) {
	log := logger.WithCtx(ctx, "OrderService.OrderCancelProcessing")

	//TODO--------Update Business custom_field--------------------------------------------------------------START
	allState := []string{utils.ORDER_STATE_WAITING_CONFIRM, utils.ORDER_STATE_DELIVERING, utils.ORDER_STATE_COMPLETE, utils.ORDER_STATE_CANCEL}

	// get seller_id from business_id
	uhb, err := utils.GetUserHasBusiness("", order.BusinessID.String())
	if err != nil {
		log.WithError(err).Errorf("Error when get user has busines: %v", err.Error())
		return
	}
	if len(uhb) == 0 {
		log.Error("Error: Empty user has business info")
		return
	}

	for _, state := range allState {
		countState := s.repo.CountOneStateOrder(ctx, order.BusinessID, state, tx)
		customFieldName := ""
		switch state {
		case utils.ORDER_STATE_WAITING_CONFIRM:
			customFieldName = "order_waiting_confirm_count"
		case utils.ORDER_STATE_DELIVERING:
			customFieldName = "order_delivering_count"
		case utils.ORDER_STATE_COMPLETE:
			customFieldName = "order_complete_count"
		case utils.ORDER_STATE_CANCEL:
			customFieldName = "order_cancel_count"
		}
		s.UpdateBusinessCustomField(ctx, order.BusinessID, customFieldName, strconv.Itoa(countState))
	}
	//TODO--------Update Business custom_field--------------------------------------------------------------END

	switch order.State {
	case utils.ORDER_STATE_CANCEL:
		//TODO--------Update Business custom_field Revenue -------------------------------------------------------------START
		revenue, err := s.repo.RevenueBusiness(ctx, model.RevenueBusinessParam{
			BusinessID: order.BusinessID.String(),
		}, tx)
		if err == nil {
			strSumGrandTotal := fmt.Sprintf("%.0f", revenue.SumGrandTotal)
			s.UpdateBusinessCustomField(ctx, order.BusinessID, "business_revenue", strSumGrandTotal)
		}
		//TODO--------Update Business custom_field Revenue --------------------------------------------------------------END

		//TODO--------Update Product sold_quantity -------------------------------------------------------------START
		PushConsumer(ctx, order.OrderItem, utils.TOPIC_UPDATE_SOLD_QUANTITY_CANCEL)
		//TODO--------Update Product sold_quantity -------------------------------------------------------------END
		go s.CreatePo(context.Background(), order, utils.ORDER_CANCELLED, utils.PO_IN)
		break
	default:
		break
	}
	return
}

func (s *OrderService) GetListOrderEcom(ctx context.Context, req model.OrderEcomRequest) (res model.ListOrderEcomResponse, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetListOrderEcom")

	// get list order ecom
	res, err = s.repo.GetListOrderEcom(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("Error when call func GetListOrderEcom")
		return res, err
	}
	return res, nil
}

func (s *OrderService) GetAllOrder(ctx context.Context, req model.OrderParam) (res model.ListOrderResponse, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetAllOrder")

	rs, err := s.repo.GetAllOrder(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("Error when call func GetListOrderEcom")
		return res, err
	}

	if req.ContactID != "" {
		// Get số đơn giao thành công và tính tổng tiền
		resCompleteOrders, err := s.repo.GetCompleteOrders(ctx, uuid.MustParse(req.ContactID), nil)
		if err != nil {
			return res, err
		}
		rs.Meta["count_complete"] = resCompleteOrders.Count
		val := strconv.FormatFloat(resCompleteOrders.SumAmount, 'f', 0, 64)
		rs.Meta["sum_grand_total_complete"] = val
	} else {
		rs.Meta["count_complete"] = 0
		rs.Meta["sum_grand_total_complete"] = 0
	}
	return rs, nil
}

func (s *OrderService) UpdateDetailOrder(ctx context.Context, req model.UpdateDetailOrderRequest, userRole string) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.UpdateDetailOrder")

	if len(req.ListOrderItem) == 0 {
		return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Đơn hàng phải có ít nhất 1 sản phẩm")
	}

	order, err := s.repo.GetOneOrder(ctx, req.ID.String(), nil)
	if err != nil {
		log.WithError(err).Error("Error when call func GetOneOrder")
		return nil, err
	}

	// check permission
	if err = utils.CheckPermission(ctx, req.UpdaterID.String(), order.BusinessID.String(), userRole); err != nil {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, utils.MessageError()[http.StatusUnauthorized])
	}

	if req.DeliveryFee != nil && req.DeliveryMethod != nil {
		if *req.OtherDiscount > (*req.OrderedGrandTotal + *req.DeliveryFee - *req.PromotionDiscount) {
			log.WithError(err).Error("Lỗi: Số tiền chiết khấu không thể lớn hơn số tiền phải trả")
			return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Số tiền chiết khấu không thể lớn hơn số tiền phải trả")
		}
	}

	if order.State != utils.ORDER_STATE_WAITING_CONFIRM && order.State != utils.ORDER_STATE_DELIVERING {
		log.WithError(err).Error("Trạng thái đơn hàng hiện tại không cho phép chỉnh sửa")
		return nil, ginext.NewError(http.StatusBadRequest, "Trạng thái đơn hàng hiện tại không cho phép chỉnh sửa")
	}

	// Check valid order
	mapItem := make(map[string]model.OrderItem)
	if len(req.ListOrderItem) > 0 && req.OrderedGrandTotal != nil && req.GrandTotal != nil && req.PromotionDiscount != nil && req.OtherDiscount != nil {
		orderGrandTotal := 0.0
		grandTotal := 0.0
		deliveryFee := 0.0

		//  Check valid order item
		mapItemOld := make(map[string]model.OrderItem)
		for _, v := range order.OrderItem {
			mapItemOld[v.SkuID.String()] = v
		}

		rCheck, err := utils.CheckCanPickQuantityV4(req.UpdaterID.String(), req.ListOrderItem, order.BusinessID.String(), mapItemOld, order.CreateMethod)
		if err != nil {
			log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
			return nil, ginext.NewError(http.StatusBadRequest, err.Error())
		} else {
			if rCheck.Status == utils.STATUS_SKU_NOT_FOUND {
				log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
				return nil, ginext.NewError(http.StatusBadRequest, "Không tìm thấy sản phẩm trong cửa hàng")
			}
			if rCheck.Status != utils.STATUS_SUCCESS {
				log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
				return rCheck, nil
			}
		}

		mapSku := make(map[string]model.CheckValidStockResponse)
		for _, v := range rCheck.ItemsInfo {
			mapSku[v.ID.String()] = v
		}

		for i, v := range req.ListOrderItem {
			itemTotalAmount := 0.0
			if v.ProductSellingPrice > 0 {
				itemTotalAmount = v.ProductSellingPrice * v.Quantity
			} else {
				itemTotalAmount = v.ProductNormalPrice * v.Quantity
			}
			req.ListOrderItem[i].TotalAmount = math.Round(itemTotalAmount)
			orderGrandTotal += req.ListOrderItem[i].TotalAmount
			if _, ok := mapSku[v.SkuID.String()]; ok {
				req.ListOrderItem[i].UOM = mapSku[v.SkuID.String()].Uom
				req.ListOrderItem[i].HistoricalCost = mapSku[v.SkuID.String()].HistoricalCost
			}
			if req.ListOrderItem[i].ProductSellingPrice != 0 {
				req.ListOrderItem[i].Price = v.ProductSellingPrice
			} else {
				req.ListOrderItem[i].Price = v.ProductNormalPrice
			}
			mapItem[v.SkuID.String()] = req.ListOrderItem[i]
		}

		// Check promotion discount
		if req.PromotionDiscount != nil && math.Round(*req.PromotionDiscount) != math.Round(order.PromotionDiscount) {
			return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Số tiền khuyến mãi không được thay đổi khi cập nhật đơn")
		}

		// Check order grand total
		if math.Round(orderGrandTotal) != math.Round(*req.OrderedGrandTotal) {
			return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Số tiền tổng sản phẩm không hợp lệ")
		}

		// Check valid delivery fee
		if req.DeliveryMethod != nil {
			switch *req.DeliveryMethod {
			case utils.DELIVERY_METHOD_BUYER_PICK_UP:
				if req.DeliveryFee != nil && *req.DeliveryFee > 0 {
					return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Phí giao hàng phải là 0đ cho trường hợp khách tự tới lấy")
				}
				deliveryFee = 0
				break
			case utils.DELIVERY_METHOD_SELLER_DELIVERY:
				if req.DeliveryFee != nil && *req.DeliveryFee >= 0 {
					deliveryFee = *req.DeliveryFee
				}
				break
			}
		}

		// Check other discount
		if *req.OtherDiscount < 0 || orderGrandTotal-order.PromotionDiscount-*req.OtherDiscount < 0 {
			return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Số tiền chiết khấu không hợp lệ")
		}

		// Check grand total
		grandTotal = orderGrandTotal + deliveryFee - order.PromotionDiscount - *req.OtherDiscount
		if grandTotal < 0 {
			grandTotal = 0
		}
		if math.Round(grandTotal) != math.Round(*req.GrandTotal) {
			return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Số tiền phải trả không hợp lệ")
		}
	}

	// Check and update buyer info
	if req.BuyerInfo != nil && req.DeliveryMethod != nil && *req.DeliveryMethod == utils.DELIVERY_METHOD_SELLER_DELIVERY {
		// Update to order record
		req.BuyerInfo.PhoneNumber = utils.ConvertVNPhoneFormat(req.BuyerInfo.PhoneNumber)
		buyerInfo, err := json.Marshal(req.BuyerInfo)
		if err != nil {
			log.WithError(err).Errorf("Error when parse buyerInfo: %v", err.Error())
		}
		order.BuyerInfo.RawMessage = buyerInfo

		// Update address of contact
		getContactRequest := model.GetContactRequest{
			BusinessID:  order.BusinessID,
			Name:        req.BuyerInfo.Name,
			PhoneNumber: req.BuyerInfo.PhoneNumber,
			Address:     req.BuyerInfo.Address,
		}
		_, _, err = common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/contact/get-contact-by-phone-number", rest.Post, nil, nil, getContactRequest)
		if err != nil {
			log.WithError(err).Errorf("Get contact error: %v", err.Error())
		}
	}

	req.BuyerInfo = nil
	common.Sync(req, &order)

	res, stocks, err := s.repo.UpdateDetailOrder(ctx, order, mapItem, nil)
	if err != nil {
		log.WithError(err).Error("Error when UpdateDetailOrder")
		return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	if order.State == utils.ORDER_STATE_DELIVERING {
		go s.UpdateStockWhenUpdateDetailOrder(context.Background(), order, stocks, "order_delivering")
	}

	go utils.SendAutoChatWhenUpdateOrder(utils.UUID(order.BuyerId).String(), utils.MESS_TYPE_UPDATE_ORDER, order.OrderNumber, fmt.Sprintf(utils.MESS_ORDER_UPDATE_DETAIL, order.OrderNumber))
	go s.PushConsumerSendEmail(context.Background(), order.ID.String(), utils.ORDER_STATE_UPDATE)

	// log history order detail
	go func() {
		history := model.History{
			BaseModel: model.BaseModel{
				CreatorID: order.UpdaterID,
			},
			ObjectID:    order.ID,
			ObjectTable: utils.TABLE_ORDER,
			Action:      utils.ACTION_UPDATE_ORDER,
			Description: utils.ACTION_UPDATE_ORDER + " in UpdateDetailOrder func - OrderService",
			Worker:      order.UpdaterID.String(),
		}

		dataOrder, err := json.Marshal(order)
		if err != nil {
			log.WithError(err).Error("Error when parse order in UpdateDetailOrder func - OrderService")
			return
		}
		history.Data.RawMessage = dataOrder

		requestData, err := json.Marshal(req)
		if err != nil {
			log.WithError(err).Error("Error when parse order request in UpdateDetailOrder - OrderService")
			return
		}
		history.DataRequest.RawMessage = requestData

		s.historyService.LogHistory(context.Background(), history, nil)
	}()

	return res, nil
}

func (s *OrderService) UpdateStockWhenUpdateDetailOrder(ctx context.Context, order model.Order, listStock []model.StockRequest, trackingType string) (err error) {
	log := logger.WithCtx(ctx, "OrderService.UpdateStockWhenUpdateDetailOrder")

	// Make data for push consumer
	reqUpdateStock := model.CreateStockRequest{
		TrackingType:   trackingType,
		IDTrackingType: order.OrderNumber,
		BusinessID:     order.BusinessID,
		ListStock:      listStock,
	}
	tResToJson, _ := json.Marshal(order)
	if err = json.Unmarshal(tResToJson, &reqUpdateStock.TrackingInfo); err != nil {
		log.WithError(err).Error("Error when marshal parse response to json when create stock")
	} else {
		go PushConsumer(context.Background(), reqUpdateStock, utils.TOPIC_UPDATE_STOCK)
	}
	return nil
}

func (s *OrderService) CountOrderState(ctx context.Context, req model.RevenueBusinessParam) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.CountOrderState")

	res, err = s.repo.CountOrderState(ctx, req, nil)
	if err != nil {
		// if err is not found return 404
		if err == gorm.ErrRecordNotFound {
			log.WithError(err).Error("CountOrder not found")
			return res, nil
		} else {
			log.WithError(err).Error("Error CountOrderState")
			return res, err
		}
	}
	return res, nil
}

func (s *OrderService) GetOrderByContact(ctx context.Context, req model.OrderByContactParam) (res model.ListOrderResponse, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetOrderByContact")

	res, err = s.repo.GetOrderByContact(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("Error GetOrderByContact")
		return res, err
	}
	return res, nil
}

func (s *OrderService) ExportOrderReport(ctx context.Context, req model.ExportOrderReportRequest) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.ExportOrderReport")

	//  Get data order list
	orders, err := s.repo.GetAllOrderForExport(ctx, req, nil)
	if err != nil {
		log.WithError(err).Error("Error GetAllOrderForExport")
		return nil, ginext.NewError(http.StatusBadRequest, "Error GetAllOrderForExport")

	}

	if len(orders) == 0 {
		log.WithError(err).Error("Record not found while GetAllOrderForExport")
		return nil, ginext.NewError(http.StatusBadRequest, "Record not found while GetAllOrderForExport")
	}

	// get business info
	businessInfo := model.BusinessMainInfo{}
	if businessInfo, err = s.GetDetailBusiness(ctx, req.BusinessID.String()); err != nil {
		log.WithError(err).Errorf("Fail to get business detail due to %v", err.Error())
		return nil, err
	}

	// Make excel file
	f := excelize.NewFile()

	// Config some new style
	styleHeader, _ := f.NewStyle(`{"font":{"bold":true, "size":14},"alignment":{"horizontal":"left","indent":0,"vertical":"center"},"border":[{"type":"left","color":"000000","style":1},{"type":"top","color":"000000","style":1},{"type":"bottom","color":"000000","style":1},{"type":"right","color":"000000","style":1}]}`)
	styleTitle, _ := f.NewStyle(`{"fill":{"type":"pattern","color":["#aad08e"],"pattern":1},"font":{"bold":true},"alignment":{"horizontal":"center","indent":0,"vertical":"center"},"border":[{"type":"left","color":"000000","style":1},{"type":"top","color":"000000","style":1},{"type":"bottom","color":"000000","style":1},{"type":"right","color":"000000","style":1}]}`)
	styleBorder, _ := f.NewStyle(`{"alignment":{"indent":0,"vertical":"center"},"border":[{"type":"left","color":"000000","style":1},{"type":"top","color":"000000","style":1},{"type":"bottom","color":"000000","style":1},{"type":"right","color":"000000","style":1}]}`)
	styleCenterHorizontal, _ := f.NewStyle(`{"alignment":{"horizontal":"center", "indent":0,"vertical":"center"},"border":[{"type":"left","color":"000000","style":1},{"type":"top","color":"000000","style":1},{"type":"bottom","color":"000000","style":1},{"type":"right","color":"000000","style":1}]}`)
	styleCurrency, _ := f.NewStyle(`{"number_format": 41, "alignment":{"indent":0,"vertical":"center"},"border":[{"type":"left","color":"000000","style":1},{"type":"top","color":"000000","style":1},{"type":"bottom","color":"000000","style":1},{"type":"right","color":"000000","style":1}]}`)
	styleDatetime, _ := f.NewStyle(`{"number_format": 22, "alignment":{"indent":0,"vertical":"center"},"border":[{"type":"left","color":"000000","style":1},{"type":"top","color":"000000","style":1},{"type":"bottom","color":"000000","style":1},{"type":"right","color":"000000","style":1}]}`)

	//======================================================================================
	//        					Sheet TỔNG QUAN ĐƠN HÀNG
	//======================================================================================
	orderSheetName := "Tổng quan đơn hàng"
	f.SetDefaultFont("Arial")
	f.SetSheetName("Sheet1", orderSheetName)
	for i := 3; i < 500; i++ {
		_ = f.SetRowHeight(orderSheetName, i, 17)
	}
	_ = f.SetRowHeight(orderSheetName, 1, 25)
	_ = f.SetRowHeight(orderSheetName, 2, 25)

	_ = f.MergeCell(orderSheetName, "A1", "R1")
	_ = f.MergeCell(orderSheetName, "A2", "R2")
	_ = f.SetCellValue(orderSheetName, "A1", fmt.Sprintf("%s - %s", businessInfo.Name, businessInfo.Domain))
	_ = f.SetCellValue(orderSheetName, "A2", "Ngày xuất báo cáo: "+utils.ConvertTimeFormatForReport(time.Now()))

	headers := map[string]string{
		"A3": "STT", "B3": "Đơn hàng", "C3": "Ngày giờ đặt",
		"D3": "Số sản phẩm", "E3": "Tổng số món", "F3": "Tổng tiền",
		"G3": "Khuyến mãi", "H3": "Phí giao hàng", "I3": "Chiết khấu", "J3": "Tổng cộng", "K3": "Trạng thái",
		"L3": "Hình thức giao hàng", "M3": "Mã khuyến mãi", "N3": "Hình thức thanh toán",
		"O3": "Tên khách hàng", "P3": "SĐT nhận", "Q3": "Địa chỉ", "R3": "Ghi chú",
	}
	for k, v := range headers {
		_ = f.SetCellValue(orderSheetName, k, v)
	}

	// Set style
	_ = f.SetCellStyle(orderSheetName, "A2", "R"+strconv.Itoa(3+len(orders)), styleBorder)
	_ = f.SetCellStyle(orderSheetName, "A3", "R3", styleTitle)
	_ = f.SetCellStyle(orderSheetName, "A1", "R1", styleHeader)
	_ = f.SetCellStyle(orderSheetName, "A4", "A"+strconv.Itoa(3+len(orders)), styleCenterHorizontal) // STT
	_ = f.SetCellStyle(orderSheetName, "D4", "D"+strconv.Itoa(3+len(orders)), styleCenterHorizontal) // Số sản phẩm
	_ = f.SetCellStyle(orderSheetName, "E4", "E"+strconv.Itoa(3+len(orders)), styleCenterHorizontal) // Tổng số món
	_ = f.SetCellStyle(orderSheetName, "O4", "O"+strconv.Itoa(3+len(orders)), styleCenterHorizontal) // Tên khách hàng
	_ = f.SetCellStyle(orderSheetName, "F4", "F"+strconv.Itoa(3+len(orders)), styleCurrency)         // Tổng tiền
	_ = f.SetCellStyle(orderSheetName, "G4", "G"+strconv.Itoa(3+len(orders)), styleCurrency)         // Khuyến mãi
	_ = f.SetCellStyle(orderSheetName, "I4", "I"+strconv.Itoa(3+len(orders)), styleCurrency)         // Chiết khấu
	_ = f.SetCellStyle(orderSheetName, "H4", "H"+strconv.Itoa(3+len(orders)), styleCurrency)         // Phí giao hàng
	_ = f.SetCellStyle(orderSheetName, "J4", "J"+strconv.Itoa(3+len(orders)), styleCurrency)         // Tổng cộng
	_ = f.SetCellStyle(orderSheetName, "C4", "C"+strconv.Itoa(3+len(orders)), styleDatetime)         // Ngày giờ đặt

	// Set col width
	_ = f.SetColWidth(orderSheetName, "A", "A", 8)
	_ = f.SetColWidth(orderSheetName, "B", "B", 13)
	_ = f.SetColWidth(orderSheetName, "C", "C", 18)
	_ = f.SetColWidth(orderSheetName, "D", "D", 14)
	_ = f.SetColWidth(orderSheetName, "E", "E", 14)
	_ = f.SetColWidth(orderSheetName, "F", "F", 14)
	_ = f.SetColWidth(orderSheetName, "G", "G", 16)
	_ = f.SetColWidth(orderSheetName, "H", "H", 16)
	_ = f.SetColWidth(orderSheetName, "I", "I", 16)
	_ = f.SetColWidth(orderSheetName, "J", "J", 16)
	_ = f.SetColWidth(orderSheetName, "K", "K", 14)
	_ = f.SetColWidth(orderSheetName, "L", "L", 19)
	_ = f.SetColWidth(orderSheetName, "M", "M", 15)
	_ = f.SetColWidth(orderSheetName, "N", "N", 20)
	_ = f.SetColWidth(orderSheetName, "O", "O", 19)
	_ = f.SetColWidth(orderSheetName, "P", "P", 15)
	_ = f.SetColWidth(orderSheetName, "Q", "Q", 30)
	_ = f.SetColWidth(orderSheetName, "R", "R", 25)

	// Set data order for sheet
	rowNumber := 4
	var rowItemsValues [][]interface{}
	for index, order := range orders {
		// Count Sum Item and Sum Quantity
		sumQuantity := 0.0
		for _, item := range order.OrderItem {
			sumQuantity += item.Quantity
		}
		rowValues := []interface{}{
			index + 1,
			order.OrderNumber,
			order.CreatedAt.Add(7 * time.Hour),
			len(order.OrderItem),
			sumQuantity,
			order.OrderedGrandTotal,
			order.PromotionDiscount,
			order.DeliveryFee,
			order.OtherDiscount,
			order.GrandTotal,
		}
		switch order.State {
		case utils.ORDER_STATE_WAITING_CONFIRM:
			rowValues = append(rowValues, "Chờ xác nhận")
			break
		case utils.ORDER_STATE_DELIVERING:
			rowValues = append(rowValues, "Đang giao")
			break
		case utils.ORDER_STATE_CANCEL:
			rowValues = append(rowValues, "Đã hủy")
			break
		case utils.ORDER_STATE_COMPLETE:
			rowValues = append(rowValues, "Hoàn thành")
			break
		}
		switch order.DeliveryMethod {
		case utils.DELIVERY_METHOD_SELLER_DELIVERY:
			rowValues = append(rowValues, "Tự đi giao")
			break
		case utils.DELIVERY_METHOD_BUYER_PICK_UP:
			rowValues = append(rowValues, "Khách đến lấy")
			break
		}
		rowValues = append(rowValues, order.PromotionCode)
		rowValues = append(rowValues, order.PaymentMethod)

		buyerInfo := model.BuyerInfo{}
		tBuyer, _ := json.Marshal(order.BuyerInfo)
		if err = json.Unmarshal(tBuyer, &buyerInfo); err != nil {
			log.WithError(err).Errorf("Cannot unmarshal buyerInfo %v", err.Error())
			rowValues = append(rowValues, "")
			rowValues = append(rowValues, "")
			rowValues = append(rowValues, "")
		} else {
			rowValues = append(rowValues, buyerInfo.Name)
			if len(buyerInfo.PhoneNumber) > 3 {
				rowValues = append(rowValues, "0"+buyerInfo.PhoneNumber[3:])
			} else {
				rowValues = append(rowValues, "")
			}
			rowValues = append(rowValues, buyerInfo.Address)
		}
		rowValues = append(rowValues, order.Note)

		for _, item := range order.OrderItem {
			itemData := []interface{}{
				len(rowItemsValues) + 1,
				rowValues[1],
				rowValues[2],
			}
			name := item.ProductName
			if item.SkuName != "" {
				name += " - " + item.SkuName
			}
			itemData = append(itemData, name)
			if item.ProductSellingPrice > 0 && item.ProductSellingPrice < item.ProductNormalPrice {
				itemData = append(itemData, item.ProductSellingPrice)
			} else {
				itemData = append(itemData, item.ProductNormalPrice)
			}
			itemData = append(itemData, item.Quantity)
			itemData = append(itemData, item.TotalAmount)
			//itemData = append(itemData, rowValues[8])
			itemData = append(itemData, rowValues[10]) // Trang thai
			itemData = append(itemData, rowValues[13]) // hinh thuc thanh toan
			itemData = append(itemData, rowValues[14]) // ten khach hang
			itemData = append(itemData, rowValues[15]) // Phone
			itemData = append(itemData, rowValues[16]) // Dia chi
			itemData = append(itemData, item.Note)     // Note

			rowItemsValues = append(rowItemsValues, itemData)
		}
		_ = f.SetSheetRow(orderSheetName, "A"+strconv.Itoa(rowNumber), &rowValues)
		rowNumber++
	}

	//======================================================================================
	//        					Sheet Chi tiết đơn hàng
	//======================================================================================
	orderDetailSheetName := "Chi tiết đơn hàng"
	f.NewSheet(orderDetailSheetName)
	for i := 3; i < 500; i++ {
		_ = f.SetRowHeight(orderDetailSheetName, i, 17)
	}

	// Header for order item
	headersItem := map[string]string{
		"A1": "STT", "B1": "Đơn hàng", "C1": "Ngày giờ đặt",
		"D1": "Tên sản phẩm", "E1": "Giá bán", "F1": "Số lượng sản phẩm",
		"G1": "Số tiền", "H1": "Trạng thái", "I1": "Hình thức giao hàng",
		"J1": "Tên khách hàng", "K1": "SĐT nhận", "L1": "Địa chỉ",
		"M1": "Ghi chú",
	}
	for k, v := range headersItem {
		_ = f.SetCellValue(orderDetailSheetName, k, v)
	}

	// Set style
	_ = f.SetCellStyle(orderDetailSheetName, "A1", "M"+strconv.Itoa(1+len(rowItemsValues)), styleBorder)
	_ = f.SetCellStyle(orderDetailSheetName, "A1", "M1", styleTitle)
	_ = f.SetCellStyle(orderDetailSheetName, "A2", "A"+strconv.Itoa(1+len(rowItemsValues)), styleCenterHorizontal)
	_ = f.SetCellStyle(orderDetailSheetName, "F2", "F"+strconv.Itoa(1+len(rowItemsValues)), styleCenterHorizontal)
	_ = f.SetCellStyle(orderDetailSheetName, "J2", "J"+strconv.Itoa(1+len(rowItemsValues)), styleCenterHorizontal)
	_ = f.SetCellStyle(orderDetailSheetName, "E2", "E"+strconv.Itoa(1+len(rowItemsValues)), styleCurrency)
	_ = f.SetCellStyle(orderDetailSheetName, "G2", "G"+strconv.Itoa(1+len(rowItemsValues)), styleCurrency)
	_ = f.SetCellStyle(orderDetailSheetName, "C2", "C"+strconv.Itoa(1+len(rowItemsValues)), styleDatetime)

	// Set col width
	_ = f.SetColWidth(orderDetailSheetName, "A", "A", 8)
	_ = f.SetColWidth(orderDetailSheetName, "B", "B", 13)
	_ = f.SetColWidth(orderDetailSheetName, "C", "C", 18)
	_ = f.SetColWidth(orderDetailSheetName, "D", "D", 22)
	_ = f.SetColWidth(orderDetailSheetName, "E", "E", 14)
	_ = f.SetColWidth(orderDetailSheetName, "F", "F", 16)
	_ = f.SetColWidth(orderDetailSheetName, "G", "G", 14)
	_ = f.SetColWidth(orderDetailSheetName, "H", "H", 16)
	_ = f.SetColWidth(orderDetailSheetName, "I", "I", 16)
	_ = f.SetColWidth(orderDetailSheetName, "J", "J", 19)
	_ = f.SetColWidth(orderDetailSheetName, "K", "K", 14)
	_ = f.SetColWidth(orderDetailSheetName, "L", "L", 30)
	_ = f.SetColWidth(orderDetailSheetName, "M", "M", 25)

	// Set data order item
	for i, item := range rowItemsValues {
		_ = f.SetSheetRow(orderDetailSheetName, "A"+strconv.Itoa(i+2), &item)
	}

	f.SetActiveSheet(0)

	// Save spreadsheet by the given path.
	fileName := "DonHang_" +
		strings.ReplaceAll(businessInfo.Domain, ".", "") +
		"_" +
		strconv.Itoa(time.Now().Year()) +
		utils.ConvertTimeIntToString(int(time.Now().Month())) +
		utils.ConvertTimeIntToString(time.Now().Day()) +
		utils.ConvertTimeIntToString(time.Now().Hour()) +
		utils.ConvertTimeIntToString(time.Now().Minute()) +
		utils.ConvertTimeIntToString(time.Now().Second()) +
		".xlsx"
	if err := f.SaveAs(fileName); err != nil {
		log.WithError(err).Error("Error when SaveAs")
		return nil, ginext.NewError(http.StatusInternalServerError, "Error when SaveAs"+err.Error())
	}

	// Save to S3 and get URL link to response
	linkReportOrders, err := s.ExcelUpFileToS3(ctx, model.UpFileToS3Request{
		File:      "./" + fileName,
		Name:      fileName,
		MediaType: "EXCEL",
		UserID:    req.UserID,
	})

	_ = os.Remove("./" + fileName)
	if err != nil {
		log.WithError(err).Error("Error when Remove")
		return nil, ginext.NewError(http.StatusBadRequest, "Error when Remove")
	}

	return linkReportOrders, nil
}

func (s *OrderService) ExcelUpFileToS3(ctx context.Context, data model.UpFileToS3Request) (string, error) {
	log := logger.WithCtx(ctx, "OrderService.ExcelUpFileToS3")

	url, err := s.S3PreUpload(ctx, data)
	if err != nil {
		log.WithError(err).Error("Error when S3PreUpload")
		return "", err
	}

	urlUpload, err := s.S3Upload(ctx, data, url)
	if err != nil {
		log.WithError(err).Error("Error when S3Upload")
		return "", err
	}

	_, err = s.S3PosUpload(ctx, data)
	if err != nil {
		log.WithError(err).Error("Error when S3PosUpload")
		return "", err
	}

	return urlUpload, nil
}

func (s *OrderService) S3PreUpload(ctx context.Context, data model.UpFileToS3Request) (string, error) {
	log := logger.WithCtx(ctx, "OrderService.S3PreUpload")

	header := map[string]string{
		"x-user-id": data.UserID.String(),
	}
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSMediaManagement+"/api/media/pre_up", http.MethodPost, header, nil, data)
	if err != nil {
		log.WithError(err).Error("Error when preUpload")
		return "", err
	}
	preUpResponse := struct {
		Data string `json:"data"`
	}{}
	err = json.Unmarshal([]byte(body), &preUpResponse)
	if err != nil {
		log.WithError(err).Error("Error unmarshal preUpResponse")
		return "", err
	}
	return preUpResponse.Data, nil
}

func (s *OrderService) S3Upload(ctx context.Context, data model.UpFileToS3Request, urlUpload string) (string, error) {
	log := logger.WithCtx(ctx, "OrderService.S3Upload")

	method := "POST"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	_ = writer.WriteField("upload_url", urlUpload)

	file, errFile7 := os.Open(data.File)
	defer file.Close()
	part7, errFile7 := writer.CreateFormFile("file", filepath.Base(data.File))
	_, errFile7 = io.Copy(part7, file)
	if errFile7 != nil {
		return "", errFile7
	}
	err := writer.Close()
	if err != nil {
		return "", err
	}

	clientUpload := &http.Client{}
	reqUpload, err := http.NewRequest(method, conf.LoadEnv().MSMediaManagement+"/api/media/upload", payload)

	if err != nil {
		log.Error("Error when upload media")
		return "", err
	}
	reqUpload.Header.Set("Content-Type", writer.FormDataContentType())
	reqUpload.Header.Set("x-user-id", data.UserID.String())

	resUpload, err := clientUpload.Do(reqUpload)
	if err != nil {
		return "", err
	}
	defer resUpload.Body.Close()

	body, err := ioutil.ReadAll(resUpload.Body)
	if err != nil {
		return "", err
	}
	fmt.Println(string(body))

	tmpResUpload := model.S3ResponseUpload{}

	if err = json.Unmarshal(body, &tmpResUpload); err != nil {
		log.Error("Error when unmarshal body")
		return "", err
	}

	if tmpResUpload.Status != http.StatusOK {
		return "", fmt.Errorf("Upload error :" + tmpResUpload.Message)
	}

	return tmpResUpload.Data.UploadUrl, nil
}

func (s *OrderService) S3PosUpload(ctx context.Context, data model.UpFileToS3Request) (string, error) {
	log := logger.WithCtx(ctx, "OrderService.S3PosUpload")

	header := map[string]string{
		"x-user-id": data.UserID.String(),
	}
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSMediaManagement+"/api/media/pos_up", http.MethodPost, header, nil, data)
	if err != nil {
		log.WithError(err).Error("Error when Pos upload")
		return "", err
	}
	return body, nil
}

func (s *OrderService) GetContactDelivering(ctx context.Context, req model.OrderParam) (res model.ContactDeliveringResponse, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetContactDelivering")

	contact, err := s.repo.GetContactDelivering(ctx, req, nil)
	if err != nil {
		log.WithError(err).Errorf("Error when get contact have order due to %v", err.Error())
		return res, ginext.NewError(http.StatusBadRequest, "Fail to get contact have order: "+err.Error())
	}

	for i, _ := range contact.Data {
		lstContact, err := s.GetContactList(ctx, contact.Data[i].ContactID.String())
		if err != nil {
			log.WithError(err).Errorf("Error when get contact list due to %v", err.Error())
			return res, ginext.NewError(http.StatusBadRequest, "Fail to get contact list: "+err.Error())
		}
		if len(lstContact) > 0 {
			contact.Data[i].ContactInfo = lstContact[0]
		}
	}
	return contact, nil
}

func (s *OrderService) GetTotalContactDelivery(ctx context.Context, req model.OrderParam) (res model.TotalContactDelivery, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetContactDelivering")

	contact, err := s.repo.GetTotalContactDelivery(ctx, req, nil)
	if err != nil {
		log.WithError(err).Errorf("Error when get contact have order due to %v", err.Error())
		return res, err
	}

	return contact, nil
}

//============================== version 2 ===========================================//
func (s *OrderService) CreateOrderV2(ctx context.Context, req model.OrderBody) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "OrderService.CreateOrder")

	// Check format phone
	if !utils.ValidPhoneFormat(req.BuyerInfo.PhoneNumber) {
		log.WithError(err).Error("Error when check format phone")
		return nil, ginext.NewError(http.StatusBadRequest, "Error when check format phone")
	}

	orderGrandTotal := 0.0
	promotionDiscount := 0.0
	deliveryFee := 0.0
	grandTotal := 0.0

	getContactRequest := model.GetContactRequest{
		BusinessID:  *req.BusinessID,
		Name:        req.BuyerInfo.Name,
		PhoneNumber: req.BuyerInfo.PhoneNumber,
		Address:     req.BuyerInfo.Address,
	}

	// Get Contact Info
	info, err := s.GetContactInfo(ctx, getContactRequest)
	if err != nil {
		return nil, err
	}

	// check warehouse
	checkCompleted := utils.ORDER_COMPLETED

	// Set buyer_id from Create Method request
	buyerID := uuid.UUID{}
	switch req.CreateMethod {
	case utils.BUYER_CREATE_METHOD:
		// buyer mustn't create product fast
		if len(req.ListProductFast) > 0 {
			log.Error("Buyer cannot create product fast")
			return nil, ginext.NewError(http.StatusUnauthorized, "Bạn không có quyền tạo sản phẩm nhanh")
		}

		// with buyer state always waiting confirm
		req.State = utils.ORDER_STATE_WAITING_CONFIRM
		buyerID = req.UserID

		break
	case utils.SELLER_CREATE_METHOD:
		// check buyer received or not
		if req.BuyerReceived {
			req.State = utils.ORDER_STATE_COMPLETE
		}

		// if req.State == utils.ORDER_STATE_COMPLETE {
		// 	checkCompleted = utils.FAST_ORDER_COMPLETED
		// }

		tUser, err := s.GetUserList(ctx, req.BuyerInfo.PhoneNumber, "")
		if err != nil {
			log.WithError(err).Error("Error when get user info from phone number of buyer info")
			return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
		}
		if len(tUser) > 0 {
			buyerID = tUser[0].ID
		}
		deliveryFee = req.DeliveryFee
		break
	default:
		log.WithError(err).Error("Error when Create method, expected: [buyer, seller]")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	//
	var lstOrderItem []model.OrderItem
	if len(req.ListProductFast) > 0 {

		// Check duplicate name
		productFast := make(map[string]string)
		productNormal := make(map[string]string)
		var lstProduct []string
		for _, v := range req.ListProductFast {
			if v.IsProductFast { // san pham nhanh
				if productFast[v.Name] == v.Name {
					log.WithError(err).Errorf("Error when create duplicated product name")
					return nil, ginext.NewError(http.StatusBadRequest, "Tạo sản phẩm không được trùng tên trong cùng một đơn hàng")
				}
				productFast[v.Name] = v.Name
			} else { // san pham thuong
				if productNormal[v.Name] == v.Name {
					log.WithError(err).Errorf("Error when create duplicated product name")
					return nil, ginext.NewError(http.StatusBadRequest, "Tạo sản phẩm không được trùng tên trong cùng một đơn hàng")
				}
				productNormal[v.Name] = v.Name
				lstProduct = append(lstProduct, v.Name)
			}
		}
		checkDuplicateProduct := model.CheckDuplicateProductRequest{
			BusinessID: req.BusinessID,
			Names:      lstProduct,
		}

		// call ms-product-management to check duplicate product name of product normal
		header := make(map[string]string)
		header["x-user-roles"] = strconv.Itoa(utils.ADMIN_ROLE)
		header["x-user-id"] = req.UserID.String()
		_, _, err = common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v1/product/check-duplicate-name", rest.Post, header, nil, checkDuplicateProduct)
		if err != nil {
			log.WithError(err).Errorf("Error when create duplicated product name")
			return nil, ginext.NewError(http.StatusBadRequest, "Tạo sản phẩm không được trùng tên")
		}

		// call create multi product
		listProductFast := model.CreateProductFast{
			BusinessID:      req.BusinessID,
			ListProductFast: req.ListProductFast,
		}

		productFastResponse, err := s.CreateMultiProduct(ctx, header, listProductFast)
		if err == nil {
			lstOrderItem = productFastResponse.Data
		}
	}

	// append ListOrderItem from request to listOrderItem received from createMultiProduct
	for _, v := range lstOrderItem {
		if v.SkuID == uuid.Nil {
			log.WithError(err).Error("Error when received from createMultiProduct")
			return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
		req.ListOrderItem = append(req.ListOrderItem, v)
	}

	// check listOrderItem empty
	if len(req.ListOrderItem) == 0 {
		log.Error("ListOrderItem mustn't empty")
		return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Đơn hàng phải có ít nhất 1 sản phẩm")
	}

	// Check valid order item
	log.WithField("list order item", req.ListOrderItem).Info("Request Order Item")

	// check can pick quantity
	rCheck, err := utils.CheckCanPickQuantityV4(req.UserID.String(), req.ListOrderItem, req.BusinessID.String(), nil, req.CreateMethod)
	if err != nil {
		log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
		return nil, ginext.NewError(http.StatusBadRequest, err.Error())
	} else {
		if rCheck.Status == utils.STATUS_SKU_NOT_FOUND {
			log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
			return nil, ginext.NewError(http.StatusBadRequest, "Không tìm thấy sản phẩm trong cửa hàng")
		}
		if rCheck.Status != utils.STATUS_SUCCESS {
			log.WithError(err).Error("Error when CheckValidOrderItems from MS Product")
			return rCheck, nil
		}
	}
	mapSku := make(map[string]model.CheckValidStockResponse)
	for _, v := range rCheck.ItemsInfo {
		mapSku[v.ID.String()] = v
	}

	// Tính tổng tiền
	for i, v := range req.ListOrderItem {
		itemTotalAmount := 0.0
		if v.ProductSellingPrice > 0 {
			itemTotalAmount = v.ProductSellingPrice * v.Quantity
		} else {
			itemTotalAmount = v.ProductNormalPrice * v.Quantity
		}
		req.ListOrderItem[i].TotalAmount = math.Round(itemTotalAmount)
		orderGrandTotal += req.ListOrderItem[i].TotalAmount
	}

	// check if order is match condition free ship
	if req.CreateMethod == utils.BUYER_CREATE_METHOD {
		if info.Data.Business.DeliveryFee == 0 || (info.Data.Business.DeliveryFee > 0 && orderGrandTotal >= info.Data.Business.MinPriceFreeShip && info.Data.Business.MinPriceFreeShip > 0) {
			deliveryFee = 0
		} else {
			deliveryFee = info.Data.Business.DeliveryFee
		}
	}

	if req.DeliveryMethod != nil && *req.DeliveryMethod == utils.DELIVERY_METHOD_BUYER_PICK_UP {
		deliveryFee = 0
	} else {
		if deliveryFee != req.DeliveryFee {
			log.WithError(err).Error("Error when get check valid delivery fee")
			return nil, ginext.NewError(http.StatusBadRequest, "Cửa hàng đã cập nhật phí vận chuyển mới, vui lòng kiểm tra lại")
		}
	}

	// Check valid Other discount
	if req.OtherDiscount < 0 || orderGrandTotal < req.OtherDiscount {
		log.WithField("other discount", req.OtherDiscount).Error("Error when get check valid delivery fee")
		return nil, ginext.NewError(http.StatusBadRequest, "Số tiền chiết khấu không hợp lệ")
	}

	// Check Promotion Code
	if req.PromotionCode != "" {
		promotion, err := s.ProcessPromotion(ctx, *req.BusinessID, req.PromotionCode, orderGrandTotal, info.Data.Contact.ID, req.UserID, true)
		if err != nil {
			log.WithField("req process promotion", req).Errorf("Get promotion error: %v", err.Error())
			return nil, ginext.NewError(http.StatusBadRequest, "Không đủ điều kiện để sử dụng mã khuyến mãi")
		}
		if promotion.ValueDiscount+req.OtherDiscount > orderGrandTotal {
			promotionDiscount = orderGrandTotal - req.OtherDiscount
		} else {
			promotionDiscount = promotion.ValueDiscount
		}
	}

	grandTotal = orderGrandTotal + deliveryFee - promotionDiscount - req.OtherDiscount
	if grandTotal < 0 {
		grandTotal = 0
	}

	// Check số tiền request lên và số tiền trong db có khớp
	if math.Round(req.OrderedGrandTotal) != math.Round(orderGrandTotal) ||
		math.Round(req.PromotionDiscount) != math.Round(promotionDiscount) ||
		math.Round(req.DeliveryFee) != math.Round(deliveryFee) ||
		math.Round(req.GrandTotal) != math.Round(grandTotal) {
		return nil, ginext.NewError(http.StatusBadRequest, "Lỗi: Số tiền không hợp lệ")
	}

	order := model.Order{
		BusinessID:        *req.BusinessID,
		ContactID:         info.Data.Contact.ID,
		PromotionCode:     req.PromotionCode,
		PromotionDiscount: promotionDiscount,
		DeliveryFee:       deliveryFee,
		OrderedGrandTotal: orderGrandTotal,
		GrandTotal:        grandTotal,
		State:             req.State,
		PaymentMethod:     strings.ToLower(req.PaymentMethod),
		DeliveryMethod:    *req.DeliveryMethod,
		Note:              req.Note,
		CreateMethod:      req.CreateMethod,
		BuyerId:           &buyerID,
		OtherDiscount:     req.OtherDiscount,
		Email:             req.Email,
	}

	req.BuyerInfo.PhoneNumber = utils.ConvertVNPhoneFormat(req.BuyerInfo.PhoneNumber)

	order.CreatorID = req.UserID

	buyerInfo, err := json.Marshal(req.BuyerInfo)
	if err != nil {
		log.WithError(err).Error("Error when parse buyerInfo")
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	order.BuyerInfo.RawMessage = buyerInfo

	log.Info("Begin work with DB")
	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	log.Info("Start DB transaction")

	// create order
	order, err = s.repo.CreateOrder(ctx, order, tx)
	if err != nil {
		log.WithError(err).Error("Error when CreateOrder")
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}
	log.WithField("order created", order).Info("Finish createOrder")

	// log history create order
	go func() {
		// order
		history := model.History{
			BaseModel: model.BaseModel{
				CreatorID: order.CreatorID,
			},
			ObjectID:    order.ID,
			ObjectTable: utils.TABLE_ORDER,
			Action:      utils.ACTION_CREATE_ORDER,
			Description: order.CreateMethod + " " + utils.ACTION_CREATE_ORDER + " in CreateOrderV2 func - OrderService",
			Worker:      order.CreatorID.String(),
		}

		dataOrder, err := json.Marshal(order)
		if err != nil {
			log.WithError(err).Error("Error when parse order in CreateOrderV2 func - OrderService")
			return
		}
		history.Data.RawMessage = dataOrder

		requestData, err := json.Marshal(req)
		if err != nil {
			log.WithError(err).Error("Error when parse order request in CreateOrderV2 - OrderService")
			return
		}
		history.DataRequest.RawMessage = requestData

		s.historyService.LogHistory(ctx, history, tx)
	}()

	if err = s.CreateOrderTracking(ctx, order, tx); err != nil {
		log.WithError(err).Error("Create order tracking error")
		return res, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	for _, orderItem := range req.ListOrderItem {
		orderItem.OrderID = order.ID
		orderItem.CreatorID = order.CreatorID
		if _, ok := mapSku[orderItem.SkuID.String()]; ok {
			orderItem.UOM = mapSku[orderItem.SkuID.String()].Uom
			orderItem.HistoricalCost = mapSku[orderItem.SkuID.String()].HistoricalCost
		}
		if orderItem.ProductSellingPrice != 0 {
			orderItem.Price = orderItem.ProductSellingPrice
		} else {
			orderItem.Price = orderItem.ProductNormalPrice
		}
		tm, err := s.repo.CreateOrderItem(ctx, orderItem, tx)
		if err != nil {
			log.WithError(err).Errorf("Error when CreateOrderItem: %v", err.Error())
			return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
		order.OrderItem = append(order.OrderItem, tm)

		// log history create order_item
		func() {
			history := model.History{
				BaseModel: model.BaseModel{
					CreatorID: orderItem.CreatorID,
				},
				ObjectID:    tm.ID,
				ObjectTable: utils.TABLE_ORDER_ITEM,
				Action:      utils.ACTION_CREATE_ORDER_ITEM,
				Description: order.CreateMethod + " " + utils.ACTION_CREATE_ORDER_ITEM + " in CreateOrderV2 func - OrderService",
				Worker:      orderItem.CreatorID.String(),
			}

			tmpData, err := json.Marshal(tm)
			if err != nil {
				log.WithError(err).Error("Error when parse order_item in CreateOrderV2 func - OrderService")
				return
			}
			history.Data.RawMessage = tmpData

			requestData, err := json.Marshal(req)
			if err != nil {
				log.WithError(err).Error("Error when parse order_item request in CreateOrderV2 - OrderService")
				return
			}
			history.DataRequest.RawMessage = requestData

			s.historyService.LogHistory(ctx, history, nil)
		}()
	}

	debit := model.Debit{}
	if req.Debit != nil {
		debit = *req.Debit
	}

	tx.Commit()

	go s.CountCustomer(context.Background(), order)
	go s.OrderProcessing(context.Background(), order, debit, checkCompleted, *req.BuyerInfo)
	go s.UpdateContactUser(context.Background(), order, order.CreatorID)
	go s.CheckCompletedTutorialCreate(context.Background(), order.CreatorID) // tutorial flow

	// push consumer to complete order mission
	go CompletedOrderMission(context.Background(), order)

	return order, nil
}

func (s *OrderService) ReminderProcessOrderV2(ctx context.Context, orderId uuid.UUID, sellerID uuid.UUID, stateCheck string, content string) {
	log := logger.WithCtx(ctx, "OrderService.ReminderProcessOrderV2")

	//time.AfterFunc(60*time.Minute, func() {
	//	// Create transaction
	var cancel context.CancelFunc
	tx, cancel := s.repo.DBWithTimeout(ctx)
	tx = tx.Begin()
	defer func() {
		tx.Rollback()
		cancel()
	}()

	order, err := s.repo.GetOneOrder(ctx, orderId.String(), tx)
	if err != nil {
		log.WithError(err).Error("ReminderProcessOrder get order " + orderId.String() + " error")
	}

	if order.State == stateCheck {
		s.SendNotificationV2(ctx, sellerID, utils.NOTIFICATION_ENTITY_KEY_ORDER, "reminder_"+order.State+"_v2", content)
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	tx.Commit()
	//})
}

func (s *OrderService) SendNotificationV2(ctx context.Context, userId uuid.UUID, entityKey string, state string, content string) {
	log := logger.WithCtx(ctx, "OrderService.SendNotificationV2")
	log.Info("begin SendNotificationV2")

	notiRequest := model.SendNotificationRequest{
		UserID:         userId,
		EntityKey:      entityKey,
		StateValue:     state,
		Language:       "vi",
		ContentReplace: content,
	}

	PushConsumer(ctx, notiRequest, utils.TOPIC_SEND_NOTIFICATION)
}

//============================== call another service ===================================//

func (s *OrderService) GetProduct(ctx context.Context, productID string) (model.Product, error) {
	log := logger.WithCtx(ctx, "OrderService.GetProduct")

	bodyData, _, err := common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/product/"+productID, rest.Get, nil, nil, nil)
	if err != nil {
		log.WithError(err).Errorf("Fail to get product due to %v", err.Error())
		return model.Product{}, fmt.Errorf("fail to get product due to %v", err.Error())
	}
	tmpResProduct := new(struct {
		Data model.Product `json:"data"`
	})
	if err = json.Unmarshal([]byte(bodyData), &tmpResProduct); err != nil {
		log.WithError(err).Errorf("GetProduct Unmarshal error %v", err.Error())
		return model.Product{}, err
	}

	return tmpResProduct.Data, nil
}

func (s *OrderService) ProcessConsumer(ctx context.Context, req model.ProcessConsumerRequest) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "ProcessConsumer").WithFields(logrus.Fields{
		"body":  req.Payload,
		"topic": req.Topic,
	})
	switch req.Topic {
	case utils.TOPIC_SEND_EMAIL_ORDER:
		var sendEmailReq model.SendEmailRequest
		if err := json.Unmarshal([]byte(req.Payload), &sendEmailReq); err != nil {
			log.Errorf("Error send email: %v", err.Error())
			return nil, err
		}
		var sendEmailOrderReq model.SendEmailRequest
		sendEmailOrderReq = sendEmailReq
		if _, err = s.SendEmailOrder(ctx, sendEmailOrderReq); err != nil {
			return nil, err
		}
		break
	case utils.TOPIC_UPDATE_EMAIL_ORDER_RECENT:
		var updateEmailOrderRecentRequest model.UpdateEmailOrderRecentRequest
		if err := json.Unmarshal([]byte(req.Payload), &updateEmailOrderRecentRequest); err != nil {
			log.Errorf("Error parse updateEmailOrderForResentRequest: %v", err.Error())
			return nil, err
		}
		if _, err = s.UpdateEmailForOrderRecent(ctx, updateEmailOrderRecentRequest); err != nil {
			return nil, err
		}
		break
	default:
		log.Errorf("Topic not found in this service!")
		return nil, fmt.Errorf("Topic not found in this service!")
	}
	return "Process consumer successfully", nil
}

func (s *OrderService) GetUserList(ctx context.Context, phoneNumber string, userIDs string) (res []model.User, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetUserList")

	param := map[string]string{}
	if phoneNumber != "" {
		param["phone_number"] = phoneNumber
	}
	if userIDs != "" {
		param["id"] = userIDs
	}
	bodyUser, _, err := common.SendRestAPI(conf.LoadEnv().MSUserManagement+"/api/user", rest.Get, nil, param, nil)
	if err != nil {
		log.WithError(err).Error("Fail to get user info")
		return res, err
	}
	tmpResUser := new(struct {
		Data []model.User `json:"data"`
	})
	if err = json.Unmarshal([]byte(bodyUser), &tmpResUser); err != nil {
		log.WithError(err).Error("Fail to unmarshal user info")
		return res, err
	}
	return tmpResUser.Data, nil
}

func (s *OrderService) GetContactInfo(ctx context.Context, req model.GetContactRequest) (res model.GetContactResponse, err error) {
	log := logger.WithCtx(ctx, "OrderService.GetContactInfo")

	bodyResponse, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/v2/contact/get-contact-by-phone-number", rest.Post, nil, nil, req)
	if err != nil {
		log.WithError(err).Error("Get contact error")
		return res, ginext.NewError(http.StatusBadRequest, "Cannot get contact error")
	}

	if err = json.Unmarshal([]byte(bodyResponse), &res); err != nil {
		log.WithError(err).Error("Fail to Unmarshal contact")
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	return res, nil
}

func (s *OrderService) CreateMultiProduct(ctx context.Context, header map[string]string, req model.CreateProductFast) (res model.ProductFastResponse, err error) {
	log := logger.WithCtx(ctx, "OrderService.CreateMultiProduct")

	bodyResponse, _, err := common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v1/create-multi-product", rest.Post, header, nil, req)
	if err != nil {
		log.WithError(err).Error("Error when create multi product")
		return res, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}
	if err = json.Unmarshal([]byte(bodyResponse), &res); err != nil {
		log.WithError(err).Error("Error when Unmarshal contact")
		return res, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}
	return res, nil
}

func (s *OrderService) CountDeliveringQuantity(ctx context.Context, req model.CountQuantityInOrderRequest) (rs interface{}, err error) {
	if err := common.CheckRequireValid(req); err != nil {
		return nil, err
	}
	return s.repo.GetCountQuantityInOrder(ctx, req, nil)
}

// check first create then push consumer update completed tutorial
func (s *OrderService) CheckCompletedTutorialCreate(ctx context.Context, creatorID uuid.UUID) {
	log := logger.WithCtx(ctx, "OrderService.CheckCompletedTutorialCreate")
	log.Info("CheckCompletedTutorialCreate")

	userGuideRequest := model.UserGuideRequest{
		GuideKey: utils.TUTORIAL_CREATE_ORDER,
		State:    utils.COMPLETED_TUTORIAL,
		UserID:   creatorID.String(),
	}

	PushConsumer(context.Background(), userGuideRequest, utils.TOPIC_SET_USER_GUIDE)
	return
}

func (s *OrderService) GetSumOrderCompleteContact(ctx context.Context, req model.GetTotalOrderByBusinessRequest) (rs interface{}, err error) {
	return s.repo.GetSumOrderCompleteContact(ctx, req, nil)
}
