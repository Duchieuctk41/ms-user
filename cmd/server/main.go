package main

import (
	"context"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/route"
	"finan/ms-order-management/pkg/utils"
	"gitlab.com/goxp/cloud0/logger"
	"os"
)

const (
	APPNAME  = "Order"
)

func main() {
	conf.SetEnv()
	logger.Init(APPNAME)
	utils.LoadMessageError()

	app := route.NewService()
	ctx := context.Background()
	err := app.Start(ctx)
	if err != nil {
		logger.Tag("main").Error(err)
	}
	os.Clearenv()
}
