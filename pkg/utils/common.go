package utils

import (
	"context"
	"encoding/json"
	"finan/ms-order-management/pkg/model"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/logger"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// check permission allow [seller | admin]
func CheckPermission(ctx context.Context, userID string, businessID string, role string) (err error) {
	log := logger.WithCtx(ctx, "CheckPermission")

	userRoles, _ := strconv.Atoi(role)
	if (userRoles&ADMIN_ROLE > 0) || (userRoles&ADMIN_ROLE == ADMIN_ROLE) {
		return nil
	}

	userHasBusiness, err := GetUserHasBusiness(userID, businessID)
	if err != nil {
		log.Errorf("Error CheckPermission GetUserHasBusiness ", err.Error())
		return err
	}

	// Check User has this business ?
	if len(userHasBusiness) == 0 {
		log.WithError(err).Error("Fail to get user has business")
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	return nil
}

// check permission allow [buyer | seller | admin]
func CheckPermissionV2(ctx context.Context, userRole string, userID uuid.UUID, businessID string, buyerID string) error {
	log := logger.WithCtx(ctx, "CheckPermissionV2")

	//Check roles
	userRoles, _ := strconv.Atoi(userRole)
	if (userRoles&ADMIN_ROLE > 0) || (userRoles&ADMIN_ROLE == ADMIN_ROLE) {
		return nil
	}

	// Buyer or Seller can get this order
	if buyerID != "" && userID.String() == buyerID {
		return nil
	}

	userHasBusiness, err := GetUserHasBusiness(userID.String(), businessID)
	if err != nil {
		log.Errorf("Error CheckSelectOrUpdateAnotherOrder GetUserHasBusiness ", err.Error())
		return err
	}

	if len(userHasBusiness) == 0 {
		log.Errorf("Fail to get user has business due to %v", err)
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	return nil
}

// 17/02/2022 - hieucn - check permission allow [buyer | seller], don't check admin role anymore
func CheckPermissionV3(ctx context.Context, userID uuid.UUID, businessID string, buyerID string) error {
	log := logger.WithCtx(ctx, "CheckPermissionV2")

	// Buyer or Seller can get this order
	if buyerID != "" && userID.String() == buyerID {
		return nil
	}

	userHasBusiness, err := GetUserHasBusiness(userID.String(), businessID)
	if err != nil {
		log.Errorf("Error CheckSelectOrUpdateAnotherOrder GetUserHasBusiness ", err.Error())
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	if len(userHasBusiness) == 0 {
		log.Errorf("Fail to get user has business due to %v", err)
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	return nil
}

// 17/02/2022 - hieucn - only allow [ seller ] to access
func CheckPermissionV4(ctx context.Context, userID string, businessID string) error {
	log := logger.WithCtx(ctx, "CheckPermissionV2")

	userHasBusiness, err := GetUserHasBusiness(userID, businessID)
	if err != nil {
		log.Errorf("Error CheckSelectOrUpdateAnotherOrder GetUserHasBusiness ", err.Error())
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	if len(userHasBusiness) == 0 {
		log.Errorf("Fail to get user has business due to %v", err)
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	return nil
}

// 17/02/2022 - hieucn - check permission allow [seller | buyer], don't check admin role anymore
func CheckPermissionV5(ctx context.Context, userID string, businessID string, buyerID string) (int, error) {
	log := logger.WithCtx(ctx, "CheckPermissionV5")

	role := BUYER_ROLE
	userHasBusiness, err := GetUserHasBusiness(userID, businessID)
	if err != nil {
		log.Errorf("Error CheckSelectOrUpdateAnotherOrder GetUserHasBusiness ", err.Error())
		return 0, ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	if len(userHasBusiness) > 0 {
		role = SELLER_ROLE
	}

	// Buyer or Seller can get this order
	if role != SELLER_ROLE && userID != buyerID {
		log.Error("Error CheckSelectOrUpdateAnotherOrder GetUserHasBusiness ")
		return 0, ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	return role, nil
}

func StrDelimitForSum(flt float64, currency string) string {
	str := strconv.FormatFloat(flt, 'f', 0, 64)

	pos := len(str) - 3
	for pos > 0 {
		str = str[:pos] + "." + str[pos:]
		pos = pos - 3
	}

	if currency != "" {
		return str + " " + currency
	}
	return str
}

func ParseIDFromUri(c *gin.Context) *uuid.UUID {
	tID := model.UriParse{}
	if err := c.ShouldBindUri(&tID); err != nil {
		_ = c.Error(err)
		return nil
	}
	if len(tID.ID) == 0 {
		_ = c.Error(fmt.Errorf("error: Empty when parse ID from URI"))
		return nil
	}
	if id, err := uuid.Parse(tID.ID[0]); err != nil {
		_ = c.Error(err)
		return nil
	} else {
		return &id
	}
}

func ParseStringIDFromUri(c *gin.Context) *string {
	tID := model.UriParse{}
	if err := c.ShouldBindUri(&tID); err != nil {
		_ = c.Error(err)
		return nil
	}
	if len(tID.ID) == 0 {
		_ = c.Error(fmt.Errorf("error: Empty when parse ID from URI"))
		return nil
	}
	return &tID.ID[0]
}

func ResizeImage(link string, w, h int) string {
	if link == "" || w == 0 || !strings.Contains(link, LINK_IMAGE_RESIZE) {
		return link
	}

	size := getSizeImage(w, h)

	env := "/finan-dev/"
	linkTemp := strings.Split(link, "/finan-dev/")
	if len(linkTemp) != 2 {
		linkTemp = strings.Split(link, "/finan/")
		env = "/finan/"
	}

	if len(linkTemp) == 2 {
		url := linkTemp[0] + "/v2/" + size + env + linkTemp[1]
		return strings.ReplaceAll(url, " ", "%20")
	}
	return strings.ReplaceAll(link, " ", "%20")
}

func getSizeImage(w, h int) string {
	if h == 0 {
		return "w" + strconv.Itoa(w)
	}
	return strconv.Itoa(w) + "x" + strconv.Itoa(h)
}

func ConvertVNPhoneFormat(phone string) string {
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

func ValidPhoneFormat(phone string) bool {
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

func RevertBeginPhone(phone string) string {
	if phone != "" {
		if strings.HasPrefix(phone, "+84") {
			phone = "0" + phone[3:]
		}
	}
	return phone
}

func PackHistoryModel(ctx context.Context, creatorID uuid.UUID, worker string, objectID uuid.UUID, objectTable string, action string, description string, data interface{}, reqData interface{}) (res model.History, err error) {
	log := logger.WithCtx(ctx, "RepoPG.PackHistoryModel")
	res = model.History{
		BaseModel: model.BaseModel{
			CreatorID: creatorID,
		},
		ObjectID:    objectID,
		ObjectTable: objectTable,
		Action:      action,
		Description: description,
		Worker:      worker,
	}

	tmpData, err := json.Marshal(data)
	if err != nil {
		log.WithError(err).Error("Error when parse order in PackHistoryModel func - utils")
		return
	}
	res.Data = tmpData
	if reqData != nil {
		requestData, err := json.Marshal(reqData)
		if err != nil {
			log.WithError(err).Error("Error when parse order request in PackHistoryModel - utils")
			return res, nil
		}
		res.DataRequest = requestData
	}

	return res, nil
}
