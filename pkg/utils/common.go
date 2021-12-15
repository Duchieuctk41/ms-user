package utils

import (
	"context"
	"encoding/json"
	"finan/ms-order-management/conf"
	"github.com/praslar/lib/common"
	"github.com/sendgrid/rest"
	"github.com/sirupsen/logrus"
	"gitlab.com/goxp/cloud0/ginext"
	"net/http"
	"strconv"
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
		log.WithError(err).
			Error("Error when call func Unmarshal")
		return ginext.NewError(http.StatusInternalServerError, MessageError()[http.StatusInternalServerError])
	}

	// Check User has this business ?
	if len(tmp.Data) == 0 {
		logrus.Errorf("Fail to get user has business due to %v", err)
		return ginext.NewError(http.StatusUnauthorized, MessageError()[http.StatusUnauthorized])
	}

	return nil
}
