package repo

import (
	"context"
	"finan/ms-order-management/pkg/model"
	"gorm.io/gorm"
	"time"
)

func (r *RepoPG) LogHistory(ctx context.Context, history model.History, tx *gorm.DB) (rs model.History, err error) {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	if err = tx.Create(&history).Error; err != nil {
		return rs, err
	}

	return history, nil
}

func (r *RepoPG) DeleteLogHistory(ctx context.Context, tx *gorm.DB) error {
	var cancel context.CancelFunc
	if tx == nil {
		tx, cancel = r.DBWithTimeout(ctx)
		defer cancel()
	}

	//fmt.Printf("now: %s\n", time.Now().String())
	//fmt.Printf("before: %s\n", time.Now().Add(time.Duration(-30*24)*time.Hour))

	if err := tx.Unscoped().Where("created_at < ?", time.Now().Add(time.Duration(-30*24)*time.Hour)).Delete(&model.History{}).Error; err != nil {
		return err
	}

	return nil
}
