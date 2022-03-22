package main

import (
	"context"
	"gitlab.com/goxp/cloud0/logger"
	"ms-user/conf"
	"ms-user/pkg/route"
	"ms-user/pkg/utils"
	"os"
)

const (
	APPNAME = "Order"
)

func main() {
	conf.SetEnv()
	logger.Init(APPNAME)
	utils.LoadMessageError()

	// TO DEBUG - No need config when deploy
	_ = os.Setenv("PORT", conf.LoadEnv().Port)
	_ = os.Setenv("DB_HOST", conf.LoadEnv().DBHost)
	_ = os.Setenv("DB_PORT", conf.LoadEnv().DBPort)
	_ = os.Setenv("DB_USER", conf.LoadEnv().DBUser)
	_ = os.Setenv("DB_PASS", conf.LoadEnv().DBPass)
	_ = os.Setenv("DB_NAME", conf.LoadEnv().DBName)
	_ = os.Setenv("ENABLE_DB", conf.LoadEnv().EnableDB)

	app := route.NewService()
	ctx := context.Background()
	err := app.Start(ctx)
	if err != nil {
		logger.Tag("main").Error(err)
	}
	os.Clearenv()
}
