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
	orderHandle := handlers.NewPoCategoryHandlers(oderService)

	v1Api := s.Router.Group("/api/v1")
	v1Api.GET("/get-one-oder", ginext.WrapHandler(orderHandle.GetOneOrder))

	// 08/12/21 - Create order fast & create product fast for seller - version app 1.0.34.1.1
	v1Api.POST("/create-order-fast", ginext.WrapHandler(orderHandle.CreateOrderFast))

	// Migrate
	migrateHandler := handlers.NewMigrationHandler(db)
	s.Router.POST("/internal/migrate", migrateHandler.Migrate)
	return s
}
