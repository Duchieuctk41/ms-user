package service

import (
	"finan/ms-order-management/pkg/repo"
)

type OrderService struct {
	repo repo.PGInterface
}

func NewOrderService(repo repo.PGInterface) OrderServiceInterface {
	return &OrderService{repo: repo}
}

type OrderServiceInterface interface {
}
