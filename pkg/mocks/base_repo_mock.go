// Code generated by MockGen. DO NOT EDIT.
// Source: base_repo.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	model "finan/ms-order-management/pkg/model"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	uuid "github.com/google/uuid"
	gorm "gorm.io/gorm"
)

// MockPGInterface is a mock of PGInterface interface.
type MockPGInterface struct {
	ctrl     *gomock.Controller
	recorder *MockPGInterfaceMockRecorder
}

func (m *MockPGInterface) OverviewSales(ctx context.Context, req model.OrverviewPandLRequest, tx *gorm.DB) (model.OverviewPandLResponse, error) {
	panic("implement me")
}

func (m *MockPGInterface) OverviewCost(ctx context.Context, req model.OrverviewPandLRequest, overviewPandL model.OverviewPandLResponse, tx *gorm.DB) (model.OverviewPandLResponse, error) {
	panic("implement me")
}

func (m *MockPGInterface) GetListProfitAndLoss(ctx context.Context, req model.ProfitAndLossRequest, tx *gorm.DB) (model.GetListProfitAndLossResponse, error) {
	panic("implement me")
}

func (m *MockPGInterface) GetCountQuantityInOrder(ctx context.Context, req model.CountQuantityInOrderRequest, tx *gorm.DB) (rs model.CountQuantityInOrderResponse, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetSumOrderCompleteContact(ctx context.Context, req model.GetTotalOrderByBusinessRequest, tx *gorm.DB) ([]model.GetTotalOrderByBusinessResponse, error) {
	panic("implement me")
}

func (m *MockPGInterface) CountOrderForTutorial(ctx context.Context, creatorID uuid.UUID, tx *gorm.DB) (count int, err error) {
	panic("implement me")
}

func (m *MockPGInterface) LogHistory(ctx context.Context, history model.History, tx *gorm.DB) (rs model.History, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetListOrderEcom(ctx context.Context, req model.OrderEcomRequest, tx *gorm.DB) (rs model.ListOrderEcomResponse, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetAllOrder(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ListOrderResponse, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetCompleteOrders(ctx context.Context, contactID uuid.UUID, tx *gorm.DB) (res model.GetCompleteOrdersResponse, err error) {
	panic("implement me")
}

func (m *MockPGInterface) UpdateDetailOrder(ctx context.Context, order model.Order, mapItem map[string]model.OrderItem, tx *gorm.DB) (rs model.Order, stocks []model.StockRequest, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetOrderTracking(ctx context.Context, req model.OrderTrackingRequest, tx *gorm.DB) (rs model.OrderTrackingResponse, err error) {
	panic("implement me")
}

func (m *MockPGInterface) CountOrderState(ctx context.Context, req model.RevenueBusinessParam, tx *gorm.DB) (res model.CountOrderState, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetOrderByContact(ctx context.Context, req model.OrderByContactParam, tx *gorm.DB) (rs model.ListOrderResponse, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetAllOrderForExport(ctx context.Context, req model.ExportOrderReportRequest, tx *gorm.DB) (orders []model.Order, err error) {
	panic("implement me")
}

func (m *MockPGInterface) GetContactDelivering(ctx context.Context, req model.OrderParam, tx *gorm.DB) (rs model.ContactDeliveringResponse, err error) {
	panic("implement me")
}

// MockPGInterfaceMockRecorder is the mock recorder for MockPGInterface.
type MockPGInterfaceMockRecorder struct {
	mock *MockPGInterface
}

// NewMockPGInterface creates a new mock instance.
func NewMockPGInterface(ctrl *gomock.Controller) *MockPGInterface {
	mock := &MockPGInterface{ctrl: ctrl}
	mock.recorder = &MockPGInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPGInterface) EXPECT() *MockPGInterfaceMockRecorder {
	return m.recorder
}

// CountOneStateOrder mocks base method.
func (m *MockPGInterface) CountOneStateOrder(ctx context.Context, businessId uuid.UUID, state string, tx *gorm.DB) int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CountOneStateOrder", ctx, businessId, state, tx)
	ret0, _ := ret[0].(int)
	return ret0
}

// CountOneStateOrder indicates an expected call of CountOneStateOrder.
func (mr *MockPGInterfaceMockRecorder) CountOneStateOrder(ctx, businessId, state, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CountOneStateOrder", reflect.TypeOf((*MockPGInterface)(nil).CountOneStateOrder), ctx, businessId, state, tx)
}

// CreateOrder mocks base method.
func (m *MockPGInterface) CreateOrder(ctx context.Context, order model.Order, tx *gorm.DB) (model.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrder", ctx, order, tx)
	ret0, _ := ret[0].(model.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOrder indicates an expected call of CreateOrder.
func (mr *MockPGInterfaceMockRecorder) CreateOrder(ctx, order, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrder", reflect.TypeOf((*MockPGInterface)(nil).CreateOrder), ctx, order, tx)
}

// CreateOrderItem mocks base method.
func (m *MockPGInterface) CreateOrderItem(ctx context.Context, orderItem model.OrderItem, tx *gorm.DB) (model.OrderItem, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrderItem", ctx, orderItem, tx)
	ret0, _ := ret[0].(model.OrderItem)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CreateOrderItem indicates an expected call of CreateOrderItem.
func (mr *MockPGInterfaceMockRecorder) CreateOrderItem(ctx, orderItem, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrderItem", reflect.TypeOf((*MockPGInterface)(nil).CreateOrderItem), ctx, orderItem, tx)
}

// CreateOrderTracking mocks base method.
func (m *MockPGInterface) CreateOrderTracking(ctx context.Context, orderTracking model.OrderTracking, tx *gorm.DB) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateOrderTracking", ctx, orderTracking, tx)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateOrderTracking indicates an expected call of CreateOrderTracking.
func (mr *MockPGInterfaceMockRecorder) CreateOrderTracking(ctx, orderTracking, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateOrderTracking", reflect.TypeOf((*MockPGInterface)(nil).CreateOrderTracking), ctx, orderTracking, tx)
}

// DBWithTimeout mocks base method.
func (m *MockPGInterface) DBWithTimeout(ctx context.Context) (*gorm.DB, context.CancelFunc) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DBWithTimeout", ctx)
	ret0, _ := ret[0].(*gorm.DB)
	ret1, _ := ret[1].(context.CancelFunc)
	return ret0, ret1
}

// DBWithTimeout indicates an expected call of DBWithTimeout.
func (mr *MockPGInterfaceMockRecorder) DBWithTimeout(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DBWithTimeout", reflect.TypeOf((*MockPGInterface)(nil).DBWithTimeout), ctx)
}

// GetContactHaveOrder mocks base method.
func (m *MockPGInterface) GetContactHaveOrder(ctx context.Context, businessId uuid.UUID, tx *gorm.DB) (string, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetContactHaveOrder", ctx, businessId, tx)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetContactHaveOrder indicates an expected call of GetContactHaveOrder.
func (mr *MockPGInterfaceMockRecorder) GetContactHaveOrder(ctx, businessId, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetContactHaveOrder", reflect.TypeOf((*MockPGInterface)(nil).GetContactHaveOrder), ctx, businessId, tx)
}

// GetOneOrder mocks base method.
func (m *MockPGInterface) GetOneOrder(ctx context.Context, id string, tx *gorm.DB) (model.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOneOrder", ctx, id, tx)
	ret0, _ := ret[0].(model.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOneOrder indicates an expected call of GetOneOrder.
func (mr *MockPGInterfaceMockRecorder) GetOneOrder(ctx, id, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOneOrder", reflect.TypeOf((*MockPGInterface)(nil).GetOneOrder), ctx, id, tx)
}

// GetOneOrderRecent mocks base method.
func (m *MockPGInterface) GetOneOrderRecent(ctx context.Context, buyerID string, tx *gorm.DB) (model.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetOneOrderRecent", ctx, buyerID, tx)
	ret0, _ := ret[0].(model.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetOneOrderRecent indicates an expected call of GetOneOrderRecent.
func (mr *MockPGInterfaceMockRecorder) GetOneOrderRecent(ctx, buyerID, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetOneOrderRecent", reflect.TypeOf((*MockPGInterface)(nil).GetOneOrderRecent), ctx, buyerID, tx)
}

// RevenueBusiness mocks base method.
func (m *MockPGInterface) RevenueBusiness(ctx context.Context, req model.RevenueBusinessParam, tx *gorm.DB) (model.RevenueBusiness, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RevenueBusiness", ctx, req, tx)
	ret0, _ := ret[0].(model.RevenueBusiness)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RevenueBusiness indicates an expected call of RevenueBusiness.
func (mr *MockPGInterfaceMockRecorder) RevenueBusiness(ctx, req, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RevenueBusiness", reflect.TypeOf((*MockPGInterface)(nil).RevenueBusiness), ctx, req, tx)
}

// UpdateOrder mocks base method.
func (m *MockPGInterface) UpdateOrder(ctx context.Context, order model.Order, tx *gorm.DB) (model.Order, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateOrder", ctx, order, tx)
	ret0, _ := ret[0].(model.Order)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateOrder indicates an expected call of UpdateOrder.
func (mr *MockPGInterfaceMockRecorder) UpdateOrder(ctx, order, tx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateOrder", reflect.TypeOf((*MockPGInterface)(nil).UpdateOrder), ctx, order, tx)
}
