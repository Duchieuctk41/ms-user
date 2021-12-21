package service

import (
	"context"
	"encoding/json"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/repo"
	"finan/ms-order-management/pkg/utils"
	"fmt"
	"gitlab.com/goxp/cloud0/logger"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jinzhu/gorm/dialects/postgres"
	"github.com/praslar/lib/common"
	"github.com/sendgrid/rest"
	sendinblue "github.com/sendinblue/APIv3-go-library/lib"
	"github.com/sirupsen/logrus"
	"gitlab.com/goxp/cloud0/ginext"
	"gorm.io/gorm"
)

type OrderService struct {
	repo repo.PGInterface
}

func NewOrderService(repo repo.PGInterface) OrderServiceInterface {
	return &OrderService{repo: repo}
}

type OrderServiceInterface interface {
	CreateOrder(ctx context.Context, req model.OrderBody) (res interface{}, err error)
	ProcessConsumer(ctx context.Context, req model.ProcessConsumerRequest) (res interface{}, err error)
}

func (s *OrderService) CreateOrder(ctx context.Context, req model.OrderBody) (res interface{}, err error) {
	log := logger.WithCtx(ctx, "Service.CreateOrder")
	// Check format phone
	if !s.ValidPhoneFormat(req.BuyerInfo.PhoneNumber) {
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	//
	orderGrandTotal := 0.0
	promotionDiscount := 0.0
	deliveryFee := 0.0
	grandTotal := 0.0

	getContactRequest := model.GetContactRequest{
		BusinessId:  *req.BusinessId,
		Name:        req.BuyerInfo.Name,
		PhoneNumber: req.BuyerInfo.PhoneNumber,
		Address:     req.BuyerInfo.Address,
	}

	// Get Contact Info
	bodyResponse, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/v2/contact/get-contact-by-phone-number", rest.Post, nil, nil, getContactRequest)
	if err != nil {
		logrus.Errorf("Get contact error: %v", err.Error())
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}
	info := model.GetContactResponse{}

	if err = json.Unmarshal([]byte(bodyResponse), &info); err != nil {
		logrus.Errorf("Fail to Unmarshal contact : %v", err.Error())
		return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}
	fmt.Println(info)

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
					return nil, ginext.NewError(http.StatusBadRequest, "Error when create duplicated product name")
				}
				productFast[v.Name] = v.Name
			} else { // san pham thuong
				if productNormal[v.Name] == v.Name {
					log.WithError(err).Errorf("Error when create duplicated product name")
					return nil, ginext.NewError(http.StatusBadRequest, "Error when create duplicated product name")
				}
				productNormal[v.Name] = v.Name
				lstProduct = append(lstProduct, v.Name)
			}
		}
		checkDuplicateProduct := model.CheckDuplicateProductRequest{
			BusinessID: req.BusinessId,
			Names:      lstProduct,
		}

		// call ms-product-management to check duplicate product name of product normal
		header := make(map[string]string)
		header["x-user-roles"] = strconv.Itoa(utils.ADMIN_ROLE)
		header["x-user-id"] = req.UserId.String()
		_, _, err = common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v1/product/check-duplicate-name", rest.Post, header, nil, checkDuplicateProduct)
		if err != nil {
			log.WithError(err).Errorf("Error when create duplicated product name")
			return nil, ginext.NewError(http.StatusBadRequest, "Tạo sản phẩm không được trùng tên")
		}

		// call create product
		listProductFast := model.CreateProductFast{
			BusinessID:      req.BusinessId,
			ListProductFast: req.ListProductFast,
		}
		var productFastResponse model.ProductFastResponse

		bodyResponse, _, err = common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v1/create-multi-product", rest.Post, header, nil, listProductFast)
		if err != nil {
			logrus.Errorf("Get contact error: %v", err.Error())
			return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
		}
		if err = json.Unmarshal([]byte(bodyResponse), &productFastResponse); err != nil {
			logrus.Errorf("Fail to Unmarshal contact : %v", err.Error())
			return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
		lstOrderItem = productFastResponse.Data
	}

	// append ListOrderItem from request to listOrderItem received from createMultiProduct
	for _, v := range lstOrderItem {
		if v.SkuID == uuid.Nil {
			return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
		req.ListOrderItem = append(req.ListOrderItem, v)
	}

	// Check valid order item
	logrus.WithField("list order item", req.ListOrderItem).Info("Request Order Item")

	// check can pick quantity, Bỏ qua với trường hợp sku_id == nil (sản phẩm )
	if rCheck, err := utils.CheckCanPickQuantity(req.UserId.String(), req.ListOrderItem, nil); err != nil {
		logrus.Errorf("Error when CheckValidOrderItems from MS Product")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	} else {
		if rCheck.Status != utils.STATUS_SUCCESS {
			return rCheck, nil
		}
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
		buyerID = req.UserId
		if info.Data.Business.DeliveryFee == 0 || (info.Data.Business.DeliveryFee > 0 && orderGrandTotal >= info.Data.Business.MinPriceFreeShip && info.Data.Business.MinPriceFreeShip > 0) {
			deliveryFee = 0
		} else {
			deliveryFee = info.Data.Business.DeliveryFee
		}
		break
	case utils.SELLER_CREATE_METHOD:
		tUser, err := s.GetUserList(req.BuyerInfo.PhoneNumber, "")
		if err != nil {
			logrus.Errorf("Error when get user info from phone number of buyer info")
			return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
		}
		if len(tUser) > 0 {
			buyerID = tUser[0].ID
		}
		deliveryFee = req.DeliveryFee
		break
	default:
		logrus.Errorf("Error when Create method, expected: [buyer, seller]")
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	if req.DeliveryMethod != nil && *req.DeliveryMethod == utils.DELIVERY_METHOD_BUYER_PICK_UP {
		deliveryFee = 0
	} else {
		if deliveryFee != req.DeliveryFee {
			logrus.Errorf("Error when get check valid delivery fee")
			return nil, ginext.NewError(http.StatusBadRequest, "Cửa hàng đã cập nhật phí vận chuyển mới, vui lòng kiểm tra lại")
		}
	}

	// Check valid grand total
	if req.OtherDiscount > (req.OrderedGrandTotal + req.DeliveryFee - req.PromotionDiscount) {
		logrus.Errorf("Error when get check valid delivery fee")
		return nil, ginext.NewError(http.StatusBadRequest, "Số tiền chiết khấu không được lớn hơn số tiền phải trả")
	}

	// Check Promotion Code
	if req.PromotionCode != "" {
		promotion, err := s.ProcessPromotion(*req.BusinessId, req.PromotionCode, orderGrandTotal, info.Data.Contact.ID, req.UserId, true)
		if err != nil {
			logrus.WithField("req process promotion", req).Errorf("Get promotion error: %v", err.Error())
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
	if req.BuyerReceived {
		req.State = utils.ORDER_STATE_COMPLETE
	}

	order := model.Order{
		BusinessId:        *req.BusinessId,
		ContactId:         info.Data.Contact.ID,
		PromotionCode:     req.PromotionCode,
		PromotionDiscount: promotionDiscount,
		DeliveryFee:       deliveryFee,
		OrderedGrandTotal: orderGrandTotal,
		GrandTotal:        grandTotal,
		State:             req.State,
		PaymentMethod:     req.PaymentMethod,
		DeliveryMethod:    *req.DeliveryMethod,
		Note:              req.Note,
		CreateMethod:      req.CreateMethod,
		BuyerId:           &buyerID,
		OtherDiscount:     req.OtherDiscount,
		Email:             req.Email,
	}

	req.BuyerInfo.PhoneNumber = s.ConvertVNPhoneFormat(req.BuyerInfo.PhoneNumber)

	order.CreatorID = req.UserId

	buyerInfo, err := json.Marshal(req.BuyerInfo)
	if err != nil {
		logrus.Errorf("Error when parse buyerInfo: %v", err.Error())
		return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	order.BuyerInfo.RawMessage = buyerInfo

	// Create transaction
	tx := s.repo.GetRepo().Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// create order
	order, err = s.repo.CreateOrder(ctx, order, tx)
	if err != nil {
		logrus.Errorf("Error when CreateOrder: %v", err.Error())
		return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	if err = s.CreateOrderTracking(ctx, order, tx); err != nil {
		logrus.Errorf("Create order tracking error", err.Error())
		return nil, ginext.NewError(http.StatusBadRequest, utils.MessageError()[http.StatusBadRequest])
	}

	for _, orderItem := range req.ListOrderItem {
		orderItem.OrderId = order.ID
		tm, err := s.repo.CreateOrderItem(ctx, orderItem, tx)
		if err != nil {
			logrus.Errorf("Error when CreateOrderItem: %v", err.Error())
			return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
		}
		order.OrderItem = append(order.OrderItem, tm)
	}

	debit := model.Debit{}
	if req.Debit != nil {
		debit = *req.Debit
	}

	tx.Commit()
	go s.CountCustomer(ctx, order)
	go s.OrderProcessing(ctx, order, debit)
	go s.UpdateContactUser(order, order.CreatorID)

	// push consumer to complete order mission
	go CompletedOrderMission(order)

	return order, nil
}

func (s *OrderService) ValidPhoneFormat(phone string) bool {
	if phone == "" {
		return false
	}
	if len(phone) == 13 {
		return true
	}
	internationalPhone := regexp.MustCompile("^\\+[1-9]\\d{1,14}$")
	vietnamPhone := regexp.MustCompile(`((09|03|07|08|05)+([0-9]{8})\b)`)
	if !vietnamPhone.MatchString(phone) {
		if !internationalPhone.MatchString(phone) {
			return false
		}
	}
	return true
}

func (s *OrderService) GetUserList(phoneNumber string, userIDs string) (res []model.User, err error) {
	param := map[string]string{}
	if phoneNumber != "" {
		param["phone_number"] = phoneNumber
	}
	if userIDs != "" {
		param["id"] = userIDs
	}
	bodyUser, _, err := common.SendRestAPI(conf.LoadEnv().MSUserManagement+"/api/user", rest.Get, nil, param, nil)
	if err != nil {
		logrus.Errorf("Fail to get user info due to %v", err)
		return res, err
	}
	tmpResUser := new(struct {
		Data []model.User `json:"data"`
	})
	if err = json.Unmarshal([]byte(bodyUser), &tmpResUser); err != nil {
		logrus.Errorf("Fail to get user info due to %v", err)
		return res, err
	}
	return tmpResUser.Data, nil
}

func (s *OrderService) ProcessPromotion(businessId uuid.UUID, promotionCode string, orderGrandTotal float64, contactID uuid.UUID, currentUser uuid.UUID, isUse bool) (model.Promotion, error) {
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
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, "Get promotion info error: "+err.Error())
	}

	if err = json.Unmarshal([]byte(bodyResponse), &promotion); err != nil {
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, "Get promotion info error: "+err.Error())
	}

	return promotion.Data, nil

}

func (s *OrderService) ConvertVNPhoneFormat(phone string) string {
	if phone != "" {
		if strings.HasPrefix(phone, "84") {
			phone = "+" + phone
		}
		if strings.HasPrefix(phone, "0") {
			phone = "+84" + phone[1:]
		}
	}
	return phone
}

func (s *OrderService) OrderProcessing(ctx context.Context, order model.Order, debit model.Debit) (err error) {
	log := logrus.WithContext(ctx).WithField("Order", order)
	tx := s.repo.GetRepo().Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
	//TODO--------Update Business custom_field--------------------------------------------------------------START
	allState := []string{utils.ORDER_STATE_WAITING_CONFIRM, utils.ORDER_STATE_DELIVERING, utils.ORDER_STATE_COMPLETE, utils.ORDER_STATE_CANCEL}

	// get seller_id from business_id
	uhb, err := utils.GetUserHasBusiness("", order.BusinessId.String())
	if err != nil {
		log.Error("Error when get user has busines: " + err.Error())
		return
	}
	if len(uhb) == 0 {
		log.Error("Error: Empty user has business info")
		return
	}

	for _, state := range allState {
		countState := s.repo.CountOneStateOrder(ctx, order.BusinessId, state, tx)
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
		s.UpdateBusinessCustomField(order.BusinessId, customFieldName, strconv.Itoa(countState))
	}

	//TODO--------Update Business custom_field--------------------------------------------------------------END

	// send email
	go s.PushConsumerSendEmail(order.ID.String(), order.State)

	switch order.State {

	case utils.ORDER_STATE_WAITING_CONFIRM:
		go s.SendNotification(uhb[0].UserID, utils.NOTIFICATION_ENTITY_KEY_ORDER, order.State, order.OrderNumber)
		go s.ReminderProcessOrder(ctx, order.ID, uhb[0].UserID, utils.ORDER_STATE_WAITING_CONFIRM)
		go utils.SendAutoChatWhenUpdateOrder(utils.UUID(order.BuyerId).String(), utils.MESS_TYPE_UPDATE_ORDER, order.OrderNumber, fmt.Sprintf(utils.MESS_ORDER_WAITING_CONFIRM, order.OrderNumber))
		break
	case utils.ORDER_STATE_DELIVERING:
		go s.ReminderProcessOrder(ctx, order.ID, uhb[0].UserID, utils.ORDER_STATE_DELIVERING)
		go utils.SendAutoChatWhenUpdateOrder(utils.UUID(order.BuyerId).String(), utils.MESS_TYPE_UPDATE_ORDER, order.OrderNumber, fmt.Sprintf(utils.MESS_ORDER_DELIVERING, order.OrderNumber))
		go s.UpdateStock(ctx, order, "order_delivering")
		break
	case utils.ORDER_STATE_COMPLETE:
		//TODO--------Update Business custom_field Revenue -------------------------------------------------------------START
		revenue, err := s.repo.RevenueBusiness(ctx, model.RevenueBusinessParam{
			BusinessID: order.BusinessId,
		}, tx)
		if err == nil {
			strSumGrandTotal := fmt.Sprintf("%.0f", revenue.SumGrandTotal)
			s.UpdateBusinessCustomField(order.BusinessId, "business_revenue", strSumGrandTotal)
		}

		//--------------------------------------------------------------------------------------------------------------------

		// Create Business transaction
		cateIDSell, _ := uuid.Parse(utils.CATEGORY_SELL)
		businessTransaction := model.BusinessTransaction{
			ID:              uuid.New(),
			CreatorID:       uhb[0].UserID,
			BusinessID:      order.BusinessId,
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

		err = s.CreateBusinessTransaction(businessTransaction)
		if err != nil {
			logrus.Error("Error when create business transaction: " + err.Error())
			return err
		}

		//go utils.SendAutoChatWhenUpdateOrder(utils.UUID(order.BuyerId).String(), utils.MESS_TYPE_SHOW_INVOICE, order.OrderNumber, fmt.Sprintf(utils.MESS_ORDER_COMPLETED, order.OrderNumber))

		if debit.BuyerPay != nil && *debit.BuyerPay < order.GrandTotal {
			contactTransaction := model.ContactTransaction{
				ID:              uuid.New(),
				CreatorID:       uhb[0].UserID,
				BusinessID:      order.BusinessId,
				Amount:          order.GrandTotal - *debit.BuyerPay,
				ContactID:       order.ContactId,
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
			err = s.CreateContactTransaction(contactTransaction)
			if err != nil {
				logrus.Error("Error when contact transaction: " + err.Error())
				return err
			}
		}
		go PushConsumer(order.OrderItem, utils.TOPIC_UPDATE_SOLD_QUANTITY)
		go s.CreatePo(ctx, order)
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
	tx := s.repo.GetRepo().Begin()

	_, countCustomer, err := s.repo.GetContactHaveOrder(ctx, order.BusinessId, tx)
	if err != nil {
		logrus.Errorf("Fail to get contact have order due to %v", err)
		return
	}

	s.UpdateBusinessCustomField(order.BusinessId, "customer_count", strconv.Itoa(countCustomer))
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()
}

func (s *OrderService) UpdateBusinessCustomField(businessId uuid.UUID, customField string, customValue string) {
	request := model.CustomFieldsRequest{
		BusinessID:   businessId,
		CustomFields: postgres.Hstore{customField: utils.String(customValue)},
	}
	PushConsumer(request, utils.TOPIC_UPDATE_CUSTOM_FIELDS)
}

func PushConsumer(value interface{}, topic string) {
	s, _ := json.Marshal(value)
	_, err := utils.PushConsumer(utils.ConsumerRequest{
		Topic: topic,
		Body:  string(s),
	})
	logrus.Errorf("PushConsumer topic: " + topic + " body: " + string(s))
	if err != nil {
		logrus.Errorf("Fail to push consumer "+topic+": %", err)
	}
}

func CompletedOrderMission(order model.Order) {
	var userID uuid.UUID
	if order.CreateMethod == utils.SELLER_CREATE_METHOD {
		userID = order.CreatorID
	} else {
		userHasBusiness, err := utils.GetUserHasBusiness("", order.BusinessId.String())
		if err != nil {
			logrus.Errorf("Fail to GetUserHasBusiness : %", err)
			return
		}
		userID = userHasBusiness[0].UserID
	}

	PushConsumer(map[string]string{
		"mission_type": "completed_order",
		"user_id":      userID.String(),
	}, utils.TOPIC_PROCESS_MISSION)
}

func (s *OrderService) UpdateContactUser(order model.Order, user_id uuid.UUID) (err error) {
	var buyerInfo *model.BuyerInfo
	values, _ := order.BuyerInfo.MarshalJSON()
	err = json.Unmarshal(values, &buyerInfo)
	if err != nil {
		logrus.Errorf("Fail to update user contact due to %v", err)
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
		logrus.Errorf("Fail to update user contact due to %v", err)
		return err
	} else {
		logrus.Errorf("Update profile user contact to %v", "successfully")
	}

	return nil
}

func (s *OrderService) CreateOrderTracking(ctx context.Context, req model.Order, tx *gorm.DB) error {
	orderTracking := model.OrderTracking{
		OrderId: req.ID,
		State:   req.State,
	}

	return s.repo.CreateOrderTracking(ctx, orderTracking, tx)
}

func (s *OrderService) PushConsumerSendEmail(id string, state string) {
	request := model.SendEmailRequest{
		ID:       id,
		State:    state,
		UserRole: strconv.Itoa(utils.ADMIN_ROLE),
	}
	PushConsumer(request, utils.TOPIC_SEND_EMAIL_ORDER)
}

func (s *OrderService) CreateBusinessTransaction(req model.BusinessTransaction) error {
	header := make(map[string]string)
	header["x-user-id"] = req.CreatorID.String()
	_, _, err := common.SendRestAPI(conf.LoadEnv().MSTransactionManagement+"/api/business-transaction/v2/create", rest.Post, header, nil, req)
	if err != nil {
		logrus.Errorf("Fail to create business transaction to %v", err)
		return err
	}
	return nil
}

func (s *OrderService) CreateContactTransaction(req model.ContactTransaction) error {
	header := make(map[string]string)
	header["x-user-id"] = req.CreatorID.String()
	_, _, err := common.SendRestAPI(conf.LoadEnv().MSTransactionManagement+"/api/v2/contact-transaction/create", rest.Post, header, nil, req)
	if err != nil {
		logrus.Errorf("Fail to create contact transaction to %v", err)
		return err
	}
	return nil
}

func (s *OrderService) CreatePo(ctx context.Context, order model.Order) (err error) {
	log := logrus.WithContext(ctx)
	// Make data for push consumer
	reqCreatePo := model.PurchaseOrderRequest{
		PoType:        "out",
		Note:          "Đơn hàng " + order.OrderNumber,
		ContactID:     order.ContactId,
		TotalDiscount: order.OtherDiscount,
		BusinessID:    order.BusinessId,
		PoDetails:     nil,
		Option:        "order_completed",
	}
	skuIDs, err := utils.CheckSkuHasStock(order.CreatorID.String(), order.OrderItem)
	if err != nil {
		log.WithError(err).Error("error when CheckSkuHasStock")
		return err
	}
	if len(skuIDs) > 0 {
		tmp := strings.Join(skuIDs, ",")
		for _, v := range order.OrderItem {
			if strings.Contains(tmp, v.SkuID.String()) {
				reqCreatePo.PoDetails = append(reqCreatePo.PoDetails, model.PoDetail{
					SkuID:    v.SkuID,
					Pricing:  v.TotalAmount / v.Quantity,
					Quantity: v.Quantity,
				})
			}
		}
		go PushConsumer(reqCreatePo, utils.TOPIC_CREATE_PO)
	}
	return nil
}

func (s *OrderService) GetContactHaveOrder(ctx context.Context, req model.OrderParam) (rs interface{}, err error) {
	tx := s.repo.GetRepo().Begin()

	contactIds, _, err := s.repo.GetContactHaveOrder(ctx, req.BusinessId, tx)
	if err != nil {
		logrus.Errorf("Fail to get contact have order due to %v", err)
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, "Fail to get contact have order: "+err.Error())
	}

	lstContact, err := s.GetContactList(contactIds)
	if err != nil {
		logrus.Errorf("Fail to get contact list due to %v", err)
		return model.Promotion{}, ginext.NewError(http.StatusBadRequest, "Fail to get contact list: "+err.Error())
	}

	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	return lstContact, nil
}

func (s *OrderService) GetContactList(contactIDs string) (res []model.Contact, err error) {

	queryParam := make(map[string]string)
	queryParam["ids"] = contactIDs

	bodyBusiness, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/contacts", rest.Get, nil, queryParam, nil)
	if err != nil {
		logrus.Errorf("Fail to get contact list due to %v", err)
		return res, err
	}
	tmpResContact := new(struct {
		Data []model.Contact `json:"data"`
	})
	if err = json.Unmarshal([]byte(bodyBusiness), &tmpResContact); err != nil {
		logrus.Errorf("Fail to get contact list due to %v", err)
		return res, err
	}
	return tmpResContact.Data, nil
}

func (s *OrderService) SendNotification(userId uuid.UUID, entityKey string, state string, content string) {
	notiRequest := model.SendNotificationRequest{
		UserId:         userId,
		EntityKey:      entityKey,
		StateValue:     state,
		Language:       "vi",
		ContentReplace: content,
	}

	_, _, err := common.SendRestAPI(conf.LoadEnv().MSNotificationManagement+"/api/notification/send-notification", rest.Post, nil, nil, notiRequest)
	if err != nil {
		logrus.Errorf("Send noti "+entityKey+"_"+state+" error %v", err.Error())
	} else {
		logrus.Errorf("Send noti " + entityKey + "_" + state + " successfully")
	}
}

func (s *OrderService) UpdateStock(ctx context.Context, order model.Order, trackingType string) (err error) {
	log := logrus.WithContext(ctx).WithField("order Items", order.OrderItem)

	// Make data for push consumer
	reqUpdateStock := model.CreateStockRequest{
		TrackingType:   trackingType,
		IDTrackingType: order.OrderNumber,
		BusinessID:     order.BusinessId,
	}
	tResToJson, _ := json.Marshal(order)
	if err = json.Unmarshal(tResToJson, &reqUpdateStock.TrackingInfo); err != nil {
		log.WithError(err).Error("Error when marshal parse response to json when create stock")
	} else {
		for _, v := range order.OrderItem {
			reqUpdateStock.ListStock = append(reqUpdateStock.ListStock, model.StockRequest{
				SkuID:          v.SkuID,
				QuantityChange: v.Quantity,
			})
		}
		go PushConsumer(reqUpdateStock, utils.TOPIC_UPDATE_STOCK)
	}
	return nil
}

func (s *OrderService) ReminderProcessOrder(ctx context.Context, orderId uuid.UUID, sellerID uuid.UUID, stateCheck string) {

	time.AfterFunc(60*time.Minute, func() {
		tx := s.repo.GetRepo().Begin()

		order, err := s.repo.GetOneOrder(ctx, orderId.String(), tx)
		if err != nil {
			logrus.Errorf("ReminderProcessOrder get order "+orderId.String()+" error %v", err.Error())
		}

		if order.State == stateCheck {
			s.SendNotification(sellerID, utils.NOTIFICATION_ENTITY_KEY_ORDER, "reminder_"+order.State, order.OrderNumber)
		}

		defer func() {
			if err != nil {
				tx.Rollback()
			}
		}()
	})
}

func (s *OrderService) RevertBeginPhone(phone string) string {
	if phone != "" {
		if strings.HasPrefix(phone, "+84") {
			phone = "0" + phone[3:]
		}
	}
	return phone
}

func (s *OrderService) SendEmailOrder(ctx context.Context, req model.SendEmailRequest) (rs interface{}, err error) {
	log := logrus.WithContext(ctx)

	userRoles, _ := strconv.Atoi(req.UserRole)
	if !((userRoles&utils.ADMIN_ROLE > 0) || (userRoles&utils.ADMIN_ROLE == utils.ADMIN_ROLE)) {
		log.WithError(err).Error("Unauthorized")
		return nil, ginext.NewError(http.StatusUnauthorized, err.Error())
	}

	tx := s.repo.GetRepo().Begin()
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
			ProductId:           item.ProductId,
			ProductName:         item.ProductName,
			Quantity:            item.Quantity,
			TotalAmount:         item.TotalAmount,
			SkuID:               item.SkuID,
			SkuCode:             item.SkuCode,
			Note:                item.Note,
			UOM:                 item.UOM,
			ProductNormalPrice:  utils.StrDelimitForSum(item.ProductNormalPrice, ""),
			ProductSellingPrice: utils.StrDelimitForSum(item.ProductNormalPrice, ""),
		}
		if len(item.ProductImages) > 0 {
			orderItem.ProductImages = item.ProductImages[0]
		}
		orderItems = append(orderItems, orderItem)
	}

	tmpBuyerInfo := order.BuyerInfo.RawMessage
	buyerInfo := model.BuyerInfo{}
	if err = json.Unmarshal(tmpBuyerInfo, &buyerInfo); err != nil {
		logrus.Errorf("Fail to Unmarshal contact : %v", err.Error())
		return nil, ginext.NewError(http.StatusInternalServerError, utils.MessageError()[http.StatusInternalServerError])
	}

	businessInfo, err := s.GetDetailBusiness(order.BusinessId.String())
	if err != nil {
		logrus.Errorf("Fail to get business detail due to %v", err)
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
		"PHONE_CUSTOMER":   s.RevertBeginPhone(buyerInfo.PhoneNumber),
		"EMAIL_CUSTOMER":   order.Email,
		// order
		"ORDER_NUMBER":        order.OrderNumber,
		"ORDERED_GRAND_TOTAL": utils.StrDelimitForSum(order.OrderedGrandTotal, ""),
		"PROMOTION_DISCOUNT":  utils.StrDelimitForSum(order.PromotionDiscount, ""),
		"DELIVERY_FEE":        utils.StrDelimitForSum(order.DeliveryFee, ""),
		"GRAND_TOTAL":         utils.StrDelimitForSum(order.GrandTotal, ""),
		"PAYMENT_METHOD":      order.PaymentMethod,
		"DELIVERY_METHOD":     order.DeliveryMethod,
		"ORDER_ITEMS":         orderItems,
		"TOTAL_ITEMS":         len(orderItems),
		// seller
		"NAME_BUSINESS":    businessInfo.Name,
		"ADDRESS_BUSINESS": businessInfo.Address,
		"PHONE_BUSINESS":   s.RevertBeginPhone(businessInfo.PhoneNumber),
		"DOMAIN_BUSINESS":  businessInfo.Domain,
	}
	if order.CreatorID != uuid.Nil {
		tParams["QRCODE"] = "https://" + businessInfo.Domain + "/o/" + order.OrderNumber + "?required-login=true"
	} else {
		tParams["QRCODE"] = "https://" + businessInfo.Domain + "/o/" + order.OrderNumber
	}

	if businessInfo.Avatar != "" {
		tParams["AVATAR_BUSINESS"] = businessInfo.Avatar
	} else {
		tParams["AVATAR_BUSINESS"] = "https://jx-central-media-stg.s3.ap-southeast-1.amazonaws.com/finan/default_image/default_avatar_shop.png"
	}
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
	default:
		return nil, nil
		break
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
	default:
		return nil, nil
		break
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

	return rs, err
}

func (s *OrderService) GetDetailBusiness(businessID string) (res model.BusinessMainInfo, err error) {
	bodyBusiness, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/business/"+businessID, rest.Get, nil, nil, nil)
	if err != nil {
		return res, err
	}
	tmpResBusiness := new(struct {
		Data model.BusinessMainInfo `json:"data"`
	})
	if err = json.Unmarshal([]byte(bodyBusiness), &tmpResBusiness); err != nil {
		return res, err
	}
	return tmpResBusiness.Data, nil
}

func (s *OrderService) ProcessConsumer(ctx context.Context, req model.ProcessConsumerRequest) (res interface{}, err error) {
	logger := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"body":  req.Payload,
		"topic": req.Topic,
	})
	switch req.Topic {
	case utils.TOPIC_SEND_EMAIL_ORDER:
		var sendEmailReq model.SendEmailRequest
		if err := json.Unmarshal([]byte(req.Payload), &sendEmailReq); err != nil {
			logger.Error("Error send email: %v", err.Error())
			return nil, err
		}
		var sendEmailOrderReq model.SendEmailRequest
		sendEmailOrderReq = sendEmailReq
		s.SendEmailOrder(ctx, sendEmailOrderReq)
		break
	default:
		return nil, fmt.Errorf("Topic not found in this service!")
	}
	return "Process consumer successfully", nil
}
