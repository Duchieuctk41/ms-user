package handlers

import (
	"finan/ms-order-management/pkg/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MigrationHandler struct {
	db *gorm.DB
}

func NewMigrationHandler(db *gorm.DB) *MigrationHandler {
	return &MigrationHandler{db: db}
}

func (h *MigrationHandler) Migrate(ctx *gin.Context) {

	_ = h.db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"")

	models := []interface{}{
		&model.Order{},
		&model.OrderItem{},
		&model.OrderTracking{},
		&model.OrderEcom{},
		&model.History{},
		&model.PaymentOrderHistory{},
		&model.EcomOrder{},
		&model.EcomOrderItem{},
	}
	for _, m := range models {
		err := h.db.AutoMigrate(m)
		if err != nil {
			_ = ctx.Error(err)
			return
		}
	}
}
