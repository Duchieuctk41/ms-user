package route

import (
	"ms-user/pkg/handlers"
	"ms-user/pkg/repo"
	service2 "ms-user/pkg/service"

	"github.com/caarlos0/env/v6"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/service"
	"github.com/gin-contrib/cors"
	"time"
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
		service.NewApp("MS User Tutorial", "v1.0"),
		&extraSetting{},
	}

	// repo
	_ = env.Parse(s.setting)
	db := s.GetDB()
	if s.setting.DbDebugEnable {
		db = db.Debug()
	}
	repoPG := repo.NewPGRepo(db)

	userService := service2.NewUserService(repoPG)
	userHandle := handlers.NewUserHandlers(userService)

	v1Api := s.Router.Group("/api/v1")

	v1Api.GET("/test", ginext.WrapHandler(userHandle.TestMsUser))

	// Migrate
	migrateHandler := handlers.NewMigrationHandler(db)
	s.Router.POST("/internal/migrate", migrateHandler.Migrate)

	s.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3006"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "DELETE"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
				return origin == "http://localhost:3006"
		},
		MaxAge: 12 * time.Hour,
}))

	return s
}
