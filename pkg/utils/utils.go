package utils

import (
	"encoding/json"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/model"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"time"
	"unicode"

	"github.com/astaxie/beego/logs"
	"github.com/google/uuid"
	"github.com/praslar/lib/common"
	"github.com/sendgrid/rest"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type ConsumerRequest struct {
	Topic string `json:"topic"`
	Body  string `json:"body"`
}

type UserHasBusiness struct {
	UserID     uuid.UUID `json:"user_id"`
	BusinessID uuid.UUID `json:"business_id"`
	Domain     string    `json:"domain"`
}

func CheckCanPickQuantity(userID string, req []model.OrderItem, mapItem map[string]model.OrderItem) (res model.CheckValidOrderItemResponse, err error) {
	// Update req quantity
	var tReq []model.OrderItem
	for _, v := range req {
		if mapItem != nil {
			if item, ok := mapItem[v.SkuID.String()]; ok {
				v.Quantity = v.Quantity - item.Quantity
			}
		}
		tReq = append(tReq, v)
	}
	header := make(map[string]string)
	header["x-user-id"] = userID
	header["x-user-roles"] = strconv.Itoa(ADMIN_ROLE)
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v2/check-valid-order-items", rest.Post, header, nil, tReq)
	if err != nil {
		// parsing error
		tm := struct {
			Message string `json:"message"`
		}{}
		if err = json.Unmarshal([]byte(body), &tm); err != nil {
			return res, err
		}
		return res, fmt.Errorf(tm.Message)
	}
	tm := struct {
		Data model.CheckValidOrderItemResponse `json:"data"`
	}{}
	if err = json.Unmarshal([]byte(body), &tm); err != nil {
		return res, err
	}

	// set quantity
	for i, v := range tm.Data.ItemsInfo {
		if mapItem != nil {
			if _, ok := mapItem[v.Sku.ID.String()]; ok {
				tm.Data.ItemsInfo[i].Quantity = mapItem[v.Sku.ID.String()].Quantity
			}
		}
	}

	return tm.Data, nil
}

func CheckEmptyQuantity(quantity float64) error {
	logrus.Info("CheckEmptyQuantity")

	if quantity <= 0 {
		logrus.Error("Error when CheckEmptyQuantity")
		return fmt.Errorf("%s", "Lỗi: Số luợng sản phẩm phải lớn hơn 0")
	}
	return nil
}

func CheckCanPickQuantityV4(userID string, req []model.OrderItem, businessID string, mapItem map[string]model.OrderItem, createMethod string) (res model.CheckValidOrderItemResponse, err error) {
	// Update req quantity
	var tReq []model.OrderItem
	for _, v := range req {
		// check empty quantity
		if err := CheckEmptyQuantity(v.Quantity); err != nil {
			return res, err
		}

		if mapItem != nil {
			if item, ok := mapItem[v.SkuID.String()]; ok {
				v.Quantity = v.Quantity - item.Quantity
			}
		}
		tReq = append(tReq, v)
	}
	header := make(map[string]string)
	header["x-user-id"] = userID
	header["x-user-roles"] = strconv.Itoa(ADMIN_ROLE)
	header["x-business-id"] = businessID
	header["x-create-method"] = createMethod
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v4/check-valid-order-items", rest.Post, header, nil, tReq)
	if err != nil {
		// parsing error
		tm := struct {
			Message string `json:"message"`
		}{}
		if err = json.Unmarshal([]byte(body), &tm); err != nil {
			return res, err
		}
		return res, fmt.Errorf(tm.Message)
	}
	tm := struct {
		Data model.CheckValidOrderItemResponse `json:"data"`
	}{}
	if err = json.Unmarshal([]byte(body), &tm); err != nil {
		return res, err
	}

	// set quantity
	for i, v := range tm.Data.ItemsInfo {
		if mapItem != nil {
			if _, ok := mapItem[v.Sku.ID.String()]; ok {
				tm.Data.ItemsInfo[i].Quantity = mapItem[v.Sku.ID.String()].Quantity
			}
		}
	}

	return tm.Data, nil
}

func CheckValidStock(businessID uuid.UUID, orderItems []model.OrderItem) (res model.CheckValidOrderItemResponse, err error) {
	//Update req quantity
	header := make(map[string]string)
	header["x-user-roles"] = strconv.Itoa(ADMIN_ROLE)
	header["x-user-id"] = uuid.NewString()
	mapOrderItem := make(map[string]model.OrderItem)
	var strIDs []uuid.UUID
	for _, v := range orderItems {
		strIDs = append(strIDs, v.SkuID)
		mapOrderItem[v.SkuID.String()] = v
	}
	req := model.GetListStockRequest{
		ListSku:    strIDs,
		BusinessID: businessID,
		Page:       1,
		PageSize:   999,
	}
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSWarehouseManagement+"/api/v1/stock/get-list", rest.Post, header, nil, req)
	if err != nil {
		return res, err
	}
	tm := struct {
		Data []model.Stock `json:"data"`
	}{}
	if err = json.Unmarshal([]byte(body), &tm); err != nil {
		return res, err
	}

	var skuIDs []string
	for _, v := range tm.Data {
		if orderItem, ok := mapOrderItem[v.SkuID.String()]; ok {
			if orderItem.Quantity > v.TotalQuantity {
				skuIDs = append(skuIDs, v.SkuID.String())
			}
		}
	}
	if len(skuIDs) > 0 {
		listSKU, err := GetListSKU(skuIDs)
		if err != nil {
			return res, err
		}
		mapSKU := make(map[string]model.SkuDetail)
		for _, v := range listSKU {
			mapSKU[v.ID] = v
		}
		var itemInfo []model.CheckValidStockResponse
		for _, v := range tm.Data {
			if sku, ok := mapSKU[v.SkuID.String()]; ok {
				itemInfo = append(itemInfo, model.CheckValidStockResponse{
					Sku: model.Sku{
						ID:              uuid.MustParse(sku.ID),
						SkuName:         sku.Name,
						Media:           sku.Media,
						SellingPrice:    sku.SellingPrice,
						NormalPrice:     sku.NormalPrice,
						ProductID:       uuid.MustParse(sku.ProductID),
						ProductName:     mapOrderItem[v.SkuID.String()].ProductName,
						Uom:             mapOrderItem[v.SkuID.String()].UOM,
						SkuCode:         sku.SkuCode,
						Barcode:         sku.Barcode,
						CanPickQuantity: sku.CanPickQuantity,
						Type:            sku.Type,
						Quantity:        mapOrderItem[v.SkuID.String()].Quantity,
					},
					Stock: &model.StockForCheckValid{
						TotalQuantity:      v.TotalQuantity,
						DeliveringQuantity: v.DeliveringQuantity,
						BlockedQuantity:    v.BlockedQuantity,
						HistoricalCost:     v.HistoricalCost,
					},
				})
			}
		}
		res = model.CheckValidOrderItemResponse{
			Status:    SOLD_OUT,
			ItemsInfo: itemInfo,
		}
	} else {
		res.Status = STATUS_SUCCESS
	}
	return res, nil
}

// 02/03/2022 -hieucn - call to finan-product, update from CheckCanPickQuantityV4
func CheckCanPickQuantityV5(userID string, req []model.OrderItem, businessID string, mapItem map[string]model.OrderItem, createMethod string) (res model.CheckValidOrderItemResponse, err error) {
	// Update req quantity

	var tReq struct {
		Body       []model.OrderItem `json:"order_item"`
		BusinessID string            `json:"business_id"`
		Method     string            `json:"method"`
	}
	for _, v := range req {
		// check empty quantity
		if err := CheckEmptyQuantity(v.Quantity); err != nil {
			return res, err
		}

		//if mapItem != nil {
		//	if item, ok := mapItem[v.SkuID.String()]; ok {
		//		v.Quantity = v.Quantity - item.Quantity
		//	}
		//}
		tReq.Body = append(tReq.Body, v)
	}
	header := make(map[string]string)
	header["x-user-id"] = userID
	tReq.BusinessID = businessID
	tReq.Method = createMethod
	body, _, err := common.SendRestAPI(conf.LoadEnv().FinanProduct+"/api/v1/sku/check-valid-order-items", rest.Post, header, nil, tReq)
	if err != nil {
		// parsing error
		tm := struct {
			Message string `json:"message"`
		}{}
		if err = json.Unmarshal([]byte(body), &tm); err != nil {
			return res, err
		}
		return res, fmt.Errorf(tm.Message)
	}
	tm := struct {
		Data model.CheckValidOrderItemResponse `json:"data"`
	}{}
	if err = json.Unmarshal([]byte(body), &tm); err != nil {
		return res, err
	}

	// set quantity
	for i, v := range tm.Data.ItemsInfo {
		if mapItem != nil {
			if _, ok := mapItem[v.Sku.ID.String()]; ok {
				tm.Data.ItemsInfo[i].Quantity = mapItem[v.Sku.ID.String()].Quantity
			}
		}
	}

	return tm.Data, nil
}

func GetSkuDetail(skuIDs []string, businessID string) (res []model.SkuDetail, err error) {
	tBD := struct {
		ListSku    []string `json:"list_sku"`
		BusinessID string   `json:"business_id"`
	}{ListSku: skuIDs, BusinessID: businessID}

	body, _, err := common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/get-list-sku-detail", rest.Post, nil, nil, tBD)
	if err != nil {
		logrus.Errorf("Fail to GetProductInPo due to %v", err)
		return res, fmt.Errorf("Error when get product in po info")
	}
	tmp := new(struct {
		Data []model.SkuDetail `json:"data"`
	})
	if err = json.Unmarshal([]byte(body), &tmp); err != nil {
		return res, err
	}
	return tmp.Data, nil
}

func CurrentUser(c *http.Request) (uuid.UUID, error) {
	userIdStr := c.Header.Get("x-user-id")
	if strings.Contains(userIdStr, "|") {
		userIdStr = strings.Split(userIdStr, "|")[0]
	}
	res, err := uuid.Parse(userIdStr)
	if err != nil {
		return uuid.Nil, err
	}
	return res, nil
}

func String(in string) *string {
	return &in
}

func PushConsumer(consumer ConsumerRequest) (res []interface{}, err error) {
	_, _, err = common.SendRestAPI(conf.LoadEnv().MSConsumer+"/events", rest.Post, nil, nil, consumer)
	if err != nil {
		return res, err
	}
	return res, nil
}

func GetUserHasBusiness(userID string, businessID string) (res []UserHasBusiness, err error) {

	param := map[string]string{}
	if userID != "" {
		param["user_id"] = userID
	}
	if businessID != "" {
		param["business_id"] = businessID
	}
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/user-has-business", rest.Get, nil, param, nil)
	if err != nil {
		return res, err
	}
	tmp := new(struct {
		Data []UserHasBusiness `json:"data"`
	})
	if err = json.Unmarshal([]byte(body), &tmp); err != nil {
		return res, err
	}
	return tmp.Data, nil
}

func SendAutoChatWhenUpdateOrder(userID string, typeMess string, orderNumber string, messageContent string) {
	spBody := new(struct {
		Type           string `json:"type"`
		OrderNumber    string `json:"order_number"`
		MessageContent string `json:"message_content"`
	})
	spBody.Type = typeMess
	spBody.OrderNumber = orderNumber
	spBody.MessageContent = messageContent
	header := map[string]string{}
	header["x-user-id"] = userID
	if _, _, err := common.SendRestAPI(conf.LoadEnv().MSChat+"/api/notification/auto-reply", rest.Post, header, nil, spBody); err != nil {
		logrus.Errorf("Fail to send auto mess from support customer due to %v", err)
	}
}

func UUID(req *uuid.UUID) uuid.UUID {
	if req == nil {
		return uuid.Nil
	}
	return *req
}

func CheckSkuHasStock(userID string, req []model.OrderItem) (rs []string, err error) {
	// Update req quantity
	header := make(map[string]string)
	header["x-user-id"] = userID
	header["x-user-roles"] = strconv.Itoa(ADMIN_ROLE)
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/v1/check-sku-has-stock", rest.Post, header, nil, req)
	if err != nil {
		// parsing error
		tm := struct {
			Message string `json:"message"`
		}{}
		if err = json.Unmarshal([]byte(body), &tm); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf(tm.Message)
	}
	tm := struct {
		Data []string `json:"data"`
	}{}
	if err = json.Unmarshal([]byte(body), &tm); err != nil {
		return nil, err
	}
	return tm.Data, nil
}

func ConvertTimestampVN(dateTimeFrom *time.Time, dateTimeTo *time.Time) (string, string) {
	dateTimeFromStr := dateTimeFrom.Format("2006-01-02")
	dateTimeToStr := dateTimeTo.Format("2006-01-02")

	dateTimeFromStr = dateTimeFromStr + " 00:00:00+07"
	dateTimeToStr = dateTimeToStr + " 23:59:59+07"

	return dateTimeFromStr, dateTimeToStr
}

func TransformString(in string, uppercase bool) string {
	in = strings.TrimSpace(in)
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, err := transform.String(t, in)
	if err != nil {
		logs.Error("Failed to transform %s ", in)
		return ""
	}
	result = strings.ReplaceAll(result, "Đ", "D")
	result = strings.ReplaceAll(result, "đ", "d")
	if uppercase {
		return strings.ToUpper(result)
	}
	return strings.ToLower(result)
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r) // Mn: nonspacing marks
}

func ConvertTimeIntToString(in int) string {
	if in < 10 {
		return "0" + strconv.Itoa(in)
	}
	return strconv.Itoa(in)
}

func ConvertTimeFormatForReport(in time.Time) string {
	return fmt.Sprintf("%s/%s/%s - %s:%s",
		ConvertTimeIntToString(in.Day()),
		ConvertTimeIntToString(int(in.Month())),
		ConvertTimeIntToString(in.Year()),
		ConvertTimeIntToString(in.Hour()),
		ConvertTimeIntToString(in.Minute()),
	)
}

func RemoveSpace(str string) string {
	re := regexp.MustCompile(`\s+`)
	out := re.ReplaceAllString(str, " ")
	out = strings.TrimSpace(out)
	return out
}

func GetListSKU(skuIDs []string) (res []model.SkuDetail, err error) {
	header := make(map[string]string)
	header["x-user-roles"] = strconv.Itoa(ADMIN_ROLE)
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSProductManagement+"/api/get-list-sku", rest.Post, header, nil, skuIDs)
	if err != nil {
		return nil, err
	}
	tmp := new(struct {
		Data []model.SkuDetail `json:"data"`
	})
	if err = json.Unmarshal([]byte(body), &tmp); err != nil {
		return res, err
	}
	return tmp.Data, nil
}
