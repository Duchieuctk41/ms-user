package utils

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"ms-user/conf"
	"ms-user/pkg/model"
	"strings"
	"time"
)

func CreateToken(req model.CreateTokenRequest) (res string, err error) {
	// Create the Claims
	claims := &jwt.StandardClaims{
		//Subject:   Encrypt([]byte(LoadEnv().PassEncrypt), utils.AppEnv+"|"+platformName),
		Audience:  req.Extra,
		ExpiresAt: time.Now().Add(time.Hour * time.Duration(req.NumHour)).Unix(),
		//Id:        Encrypt([]byte(LoadEnv().PassEncrypt), userID),
		//Issuer:  req.PlatformKey,
		Subject: req.UserID, // later we should use this field for userID instead of Issuer
	}
	mainClain := model.AccessTokenClaims{
		StandardClaims: *claims,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, mainClain)
	tokenString, err := token.SignedString([]byte(conf.LoadEnv().JWTSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// ExtractUserID extracts userID part from token payload
// since we store userID+device in token as format "userID|deviceID"
func ExtractUserID(tokenUserPayload string) (uuid.UUID, error) {
	parts := strings.Split(tokenUserPayload, "|")
	userID := tokenUserPayload
	if len(parts) > 1 {
		userID = parts[0]
	}
	return uuid.Parse(userID)
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
