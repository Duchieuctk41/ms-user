package route

import (
	"finan/ms-order-management/pkg/handlers"
	"finan/ms-order-management/pkg/repo"
	service2 "finan/ms-order-management/pkg/service"

	"github.com/caarlos0/env/v6"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/service"
)

type extraSetting struct {
	DbDebugEnable bool `env:"DB_DEBUG_ENABLE" envDefault:"true"`
}

type Service struct {
	*service.BaseApp
	setting *extraSetting
}

func NewService() *Service {
	s := &Service{
		service.NewApp("MS Order Management", "v1.0"),
		&extraSetting{},
	}

	// repo
	_ = env.Parse(s.setting)
	db := s.GetDB()
	if s.setting.DbDebugEnable {
		db = db.Debug()
	}
	repoPG := repo.NewPGRepo(db)

	oderService := service2.NewOrderService(repoPG)
	orderHandle := handlers.NewOrderHandlers(oderService)

	orderTrackingService := service2.NewOrderTrackingService(repoPG)
	orderTrackingHandle := handlers.NewOrderTrackingHandlers(orderTrackingService)

	v1Api := s.Router.Group("/api/v1")

	// Order
	v1Api.GET("/get-one-order/:id", ginext.WrapHandler(orderHandle.GetOneOrder))
	v1Api.GET("/get-all-order", ginext.WrapHandler(orderHandle.GetAllOrder))
	v1Api.GET("/count-order-state", ginext.WrapHandler(orderHandle.CountOrderState))
	v1Api.GET("/get-order-by-contact", ginext.WrapHandler(orderHandle.GetOrderByContact))
	v1Api.GET("/get-contact-delivering", ginext.WrapHandler(orderHandle.GetContactDelivering))

	v1Api.POST("/create-order-for-seller", ginext.WrapHandler(orderHandle.CreateOrderFast))
	v1Api.PUT("/update-order/:id", ginext.WrapHandler(orderHandle.UpdateOrder))
	v1Api.PUT("/update-detail-order/:id", ginext.WrapHandler(orderHandle.UpdateDetailOrder))
	v1Api.POST("/export-order-report", ginext.WrapHandler(orderHandle.ExportOrderReport))

	// Order ecom
	v1Api.GET("/order-ecom/get-list", ginext.WrapHandler(orderHandle.GetListOrderEcom))

	// Order tracking
	v1Api.GET("/get-order-tracking", ginext.WrapHandler(orderTrackingHandle.GetOrderTracking))

	// Consumer - Receive message from rabbitmq - version app 1.0.34.1.1
	v1Api.POST("/consumer", ginext.WrapHandler(orderHandle.ProcessConsumer))

	// Migrate
	migrateHandler := handlers.NewMigrationHandler(db)
	s.Router.POST("/internal/migrate", migrateHandler.Migrate)
	return s
}
