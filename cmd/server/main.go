package main

import (
	"context"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/route"
	"gitlab.com/goxp/cloud0/logger"
	"os"
)

const (
	APPNAME  = "Order"
)
func main() {
	conf.SetEnv()
	logger.Init(APPNAME)

	app := route.NewService()
	ctx := context.Background()
	err := app.Start(ctx)
	if err != nil {
		logger.Tag("main").Error(err)
	}
	os.Clearenv()
}
