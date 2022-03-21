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

	historyService := service2.NewHistoryService(repoPG)

	oderService := service2.NewOrderService(repoPG, historyService)
	ProfitAndLossService := service2.NewProfitAndLossService(repoPG)
	//orderHandle := handlers.NewPoCategoryHandlers(oderService)
	ProfitAndLossHandle := handlers.NewProfitAndLossHandlers(ProfitAndLossService)
	orderHandle := handlers.NewOrderHandlers(oderService)

	orderTrackingService := service2.NewOrderTrackingService(repoPG)
	orderTrackingHandle := handlers.NewOrderTrackingHandlers(orderTrackingService)

	paymentOrderHistoryService := service2.NewPaymentOrderHistoryService(repoPG, historyService)
	paymentOrderHistoryHandle := handlers.NewPaymentOrderHistoryHandlers(paymentOrderHistoryService)

	v1Api := s.Router.Group("/api/v1")
	v2Api := s.Router.Group("/api/v2")
	v3Api := s.Router.Group("/api/v3")

	// Order
	v1Api.GET("/get-one-order/:id", ginext.WrapHandler(orderHandle.GetOneOrder))
	v1Api.GET("/buyer/get-one-order/:id", ginext.WrapHandler(orderHandle.GetOneOrderBuyer))
	v1Api.GET("/get-all-order", ginext.WrapHandler(orderHandle.GetAllOrder))
	v1Api.GET("/count-order-state", ginext.WrapHandler(orderHandle.CountOrderState))
	v1Api.GET("/get-order-by-contact", ginext.WrapHandler(orderHandle.GetOrderByContact))
	v1Api.GET("/get-contact-delivering", ginext.WrapHandler(orderHandle.GetContactDelivering))
	v1Api.GET("/get-number-delivering", ginext.WrapHandler(orderHandle.GetNumberDelivering))
	v1Api.GET("/get-total-contact-delivery", ginext.WrapHandler(orderHandle.GetTotalContactDelivery))
	v1Api.GET("/get-sum-order-complete-contact", ginext.WrapHandler(orderHandle.GetSumOrderCompleteContact))

	v1Api.POST("/create-order-for-seller", ginext.WrapHandler(orderHandle.CreateOrderFast))
	v1Api.PUT("/update-order/:id", ginext.WrapHandler(orderHandle.UpdateOrder))
	v1Api.PUT("/update-detail-order/:id", ginext.WrapHandler(orderHandle.UpdateDetailOrder))
	v1Api.PUT("/seller/update-detail-order/:id", ginext.WrapHandler(orderHandle.UpdateDetailOrderSeller))
	v1Api.POST("/export-order-report", ginext.WrapHandler(orderHandle.ExportOrderReport))
	v1Api.POST("/count-quantity-in-order", ginext.WrapHandler(orderHandle.CountDeliveringQuantity))

	// Order ecom
	v1Api.GET("/order-ecom/get-list", ginext.WrapHandler(orderHandle.GetListOrderEcom))

	// Order tracking
	v1Api.GET("/get-order-tracking", ginext.WrapHandler(orderTrackingHandle.GetOrderTracking))

	// hieucn -06/03/2022 - test local
	//v1Api.POST("/send-email-order", ginext.WrapHandler(orderHandle.SendEmailOrder))
	v1Api.DELETE("/delete-log-history", ginext.WrapHandler(orderHandle.DeleteLogHistory))

	//ProfitAndLoss
	v1Api.GET("/get-list-profit-and-loss", ginext.WrapHandler(ProfitAndLossHandle.GetListProfitAndLoss))
	v1Api.GET("/overview-profit-and-loss", ginext.WrapHandler(ProfitAndLossHandle.GetOverviewPandL))

	// Consumer
	// 15/12/21 - Receive message from rabbitmq - version app 1.0.34.1.1
	v1Api.POST("/consumer", ginext.WrapHandler(orderHandle.ProcessConsumer))

	// Payment order history
	v1Api.POST("/payment-order-history/create", ginext.WrapHandler(paymentOrderHistoryHandle.CreatePaymentOrderHistory))
	v1Api.GET("/payment-order-history/get-list", ginext.WrapHandler(paymentOrderHistoryHandle.GetListPaymentOrderHistory))

	// version 2
	v2Api.POST("/create-order", ginext.WrapHandler(orderHandle.CreateOrderV2))
	v2Api.PUT("/seller/update-detail-order/:id", ginext.WrapHandler(orderHandle.UpdateDetailOrderSellerV2))
	v2Api.POST("/seller/create-order", ginext.WrapHandler(orderHandle.CreateOrderSeller))
	v2Api.PUT("/update-order/:id", ginext.WrapHandler(orderHandle.UpdateOrderV2))
	v2Api.GET("/get-list-order", ginext.WrapHandler(orderHandle.GetlistOrderV2))

	// version 3
	v3Api.POST("/seller/create-order", ginext.WrapHandler(orderHandle.CreateOrderSellerV3))

	//pro seller
	v1Api.GET("pro-seller/get-overview", ginext.WrapHandler(orderHandle.OverviewOrder))
	v1Api.GET("pro-seller/get-list-top-sales", ginext.WrapHandler(orderHandle.GetOrderItemRevenueAnalytics))

	// analytic
	v1Api.GET("/get-daily-visit-analytics", ginext.WrapHandler(orderHandle.GetDailyViewAnalytics))
	v1Api.GET("/get-order-analytics", ginext.WrapHandler(orderHandle.GetOrderAnalytics))

	// Migrate
	migrateHandler := handlers.NewMigrationHandler(db)
	s.Router.POST("/internal/migrate", migrateHandler.Migrate)
	return s
}
