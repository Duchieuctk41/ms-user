package utils

import (
	"context"
	"encoding/json"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/model"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/praslar/lib/common"
	"github.com/sendgrid/rest"
	"github.com/sirupsen/logrus"
	"gitlab.com/goxp/cloud0/ginext"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

func CheckPermission(ctx context.Context, userId string, businessID string, role string) (err error) {
	log := logrus.WithContext(ctx).WithField("business ID", businessID)

	userRoles, _ := strconv.Atoi(role)
	if (userRoles&ADMIN_ROLE > 0) || (userRoles&ADMIN_ROLE == ADMIN_ROLE) {
		return nil
	}

	param := map[string]string{}
	param["user_id"] = userId
	param["business_id"] = businessID
	body, _, err := common.SendRestAPI(conf.LoadEnv().MSBusinessManagement+"/api/user-has-business", rest.Get, nil, param, nil)
	if err != nil {
		log.WithError(err).
			Error("Error when call func SendRestAPI")
		return ginext.NewError(http.StatusInternalServerError, MessageError()[http.StatusInternalServerError])
	}
	tmp := new(struct {
		Data []UserHasBusiness `json:"data"`
	})
	if err = json.Unmarshal([]byte(body), &tmp); err != nil {
		log.WithError(err).Error("Error when call func Unmarshal")
		return ginext.NewError(http.StatusInternalServerError, MessageError()[http.StatusInternalServerError])
	}

	// Check User has this business ?
	if len(tmp.Data) == 0 {
		log.WithError(err).Error("Fail to get user has business")
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	return nil
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
