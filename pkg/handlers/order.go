package handlers
import (
	"finan/ms-order-management/pkg/service"
	"gitlab.com/goxp/cloud0/ginext"
	"net/http"
)

type OrderHandlers struct {
	service service.OrderServiceInterface
}

func NewPoCategoryHandlers(service service.OrderServiceInterface) *OrderHandlers {
	return &OrderHandlers{service: service}
}


func (h *OrderHandlers) GetOneOrder(r *ginext.Request) (*ginext.Response, error) {
	return ginext.NewResponseData(http.StatusOK, "hello world"), nil
}

