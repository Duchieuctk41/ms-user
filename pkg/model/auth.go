package model

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type CreateTokenRequest struct {
	UserID  string `json:"user_id"`
	NumHour int    `json:"num_hour"`
	Extra   string `json:"extra"`
}

type TokenRepository struct {
	db           *gorm.DB
	expiryInDays int
	nowFunc      func() time.Time
	secret       []byte
}

type RefreshToken struct {
	BaseModel

	UserID     uuid.UUID `gorm:"index"`
	DeviceID   string    `gorm:"index"`
	Sign       string    `gorm:"index"`
	BusinessID uuid.UUID `json:"business_id" gorm:"index;type:uuid"`
	ExpiredAt  time.Time
}

type AccessTokenClaims struct {
	DeviceID       string `json:"device_id"`
	BusinessID     string `json:"business_id"`
	PermissionKeys string `json:"permission_keys"`
	jwt.StandardClaims
}

type OAuthVerifyRequest struct {
	Token               string `json:"token"`
	Method              string `json:"method"`
	URL                 string `json:"url"`
	SecurityRoles       uint64 `json:"security_roles"`
	SecurityPermissions string `json:"security_permissions"`
}

type OAuthVerifyResponseData struct {
	Error   string            `json:"error"`
	UserID  string            `json:"user_id"`
	Headers map[string]string `json:"headers"`
}
