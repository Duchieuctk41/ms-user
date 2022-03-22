package repo

import (
	"context"
	"gitlab.com/goxp/cloud0/logger"
)

func (r *RepoPG) TestMsUser(ctx context.Context) (err error) {
	log := logger.WithCtx(ctx, "RepoPG.TestMsUser")

	log.Info("RepoPG: Test ms-user success")

	return nil
}
