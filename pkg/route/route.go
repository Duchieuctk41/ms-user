package route

import (
	"github.com/caarlos0/env/v6"
	"github.com/gin-contrib/cors"
	"gitlab.com/goxp/cloud0/ginext"
	"gitlab.com/goxp/cloud0/service"
	"ms-user/pkg/handlers"
	"ms-user/pkg/repo"
	service2 "ms-user/pkg/service"
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
	s.Router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000"},
		AllowMethods:     []string{"PUT", "PATCH", "GET", "DELETE", "POST"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	userService := service2.NewUserService(repoPG)
	userHandle := handlers.NewUserHandlers(userService)

	v1Api := s.Router.Group("/api/v1")

	v1Api.GET("/test", ginext.WrapHandler(userHandle.TestMsUser))

	// user
	v1Api.POST("user/create", ginext.WrapHandler(userHandle.CreateUser))
	v1Api.POST("user/login", ginext.WrapHandler(userHandle.Login))

	// Migrate
	migrateHandler := handlers.NewMigrationHandler(db)
	s.Router.POST("/internal/migrate", migrateHandler.Migrate)

	// middleware
	v1Api.Use(userHandle.VerifyTokenHandler())
	{
		v1Api.GET("/user/get-one/:id", ginext.WrapHandler(userHandle.GetOneUserByID))
	}

	return s
}
