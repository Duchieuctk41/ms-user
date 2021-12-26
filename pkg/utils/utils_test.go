package utils

import (
	"encoding/json"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/valid"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"net/http"
	"reflect"
	"strconv"
	"testing"
)

func TestCheckCanPickQuantity(t *testing.T) {
	conf.SetEnv()

	ctr1 := gomock.NewController(t)
	defer ctr1.Finish()

	type args struct {
		userID  string
		req     []model.OrderItem
		mapItem map[string]model.OrderItem
	}

	tests := []struct {
		name    string
		args    args
		wantRes model.CheckValidOrderItemResponse
		wantErr bool
	}{
		// TODO: Add test cases
		{
			name: "happy flow CheckCanPickQuantity success",
			args: args{
				userID: "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
				req: []model.OrderItem{
					{
						SkuID:               uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
						ProductName:         "Wiegand Paul",
						SkuName:             "",
						SkuCode:             "SP70",
						ProductImages:       pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/fa825178-3d63-417d-9913-9fe5959b58df/image/67ce83e8-3a1e-4dbc-b5e1-75a704a818e4.jpeg"},
						ProductNormalPrice:  50000,
						ProductSellingPrice: 50000,
						Quantity:            1,
						Note:                "",
					},
				},
				mapItem: nil,
			},
			wantRes: model.CheckValidOrderItemResponse{
				Status: "success",
				ItemsInfo: []model.CheckValidStockResponse{
					{Sku: model.Sku{
						ID:              uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
						SkuName:         "",
						ProductName:     "Wiegand Paul",
						Quantity:        1, // so luong ma khach request len
						Media:           pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/fa825178-3d63-417d-9913-9fe5959b58df/image/67ce83e8-3a1e-4dbc-b5e1-75a704a818e4.jpeg"},
						SellingPrice:    50000,
						NormalPrice:     50000,
						OldSellingPrice: 0,
						OldNormalPrice:  0,
						Uom:             "Vé",
						SkuCode:         "z6",
						Barcode:         "8650",
						CanPickQuantity: 95,
						Type:            "stock_non_varriant",
					},
						Stock: nil,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "happy flow CheckCanPickQuantity price_change",
			args: args{
				userID: "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
				req: []model.OrderItem{
					{
						SkuID:               uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
						ProductName:         "Wiegand Paul",
						SkuName:             "",
						SkuCode:             "SP70",
						ProductImages:       pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/fa825178-3d63-417d-9913-9fe5959b58df/image/67ce83e8-3a1e-4dbc-b5e1-75a704a818e4.jpeg"},
						ProductNormalPrice:  60000,
						ProductSellingPrice: 60000,
						Quantity:            1,
						Note:                "",
					},
				},
				mapItem: nil,
			},
			wantRes: model.CheckValidOrderItemResponse{
				Status: "change_price",
				ItemsInfo: []model.CheckValidStockResponse{
					{Sku: model.Sku{
						ID:              uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
						SkuName:         "",
						ProductName:     "Wiegand Paul",
						Quantity:        1, // so luong ma khach request len
						Media:           pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/fa825178-3d63-417d-9913-9fe5959b58df/image/67ce83e8-3a1e-4dbc-b5e1-75a704a818e4.jpeg"},
						SellingPrice:    50000,
						NormalPrice:     50000,
						OldSellingPrice: 60000,
						OldNormalPrice:  60000,
						Uom:             "Vé",
						SkuCode:         "z6",
						Barcode:         "8650",
						CanPickQuantity: 95,
						Type:            "stock_non_varriant",
					},
						Stock: nil,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "happy flow CheckCanPickQuantity out_of_stock",
			args: args{
				userID: "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
				req: []model.OrderItem{
					{
						SkuID:               uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
						ProductName:         "Wiegand Paul",
						SkuName:             "",
						SkuCode:             "SP70",
						ProductImages:       pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/fa825178-3d63-417d-9913-9fe5959b58df/image/67ce83e8-3a1e-4dbc-b5e1-75a704a818e4.jpeg"},
						ProductNormalPrice:  50000,
						ProductSellingPrice: 50000,
						Quantity:            1000,
						Note:                "",
					},
				},
				mapItem: nil,
			},
			wantRes: model.CheckValidOrderItemResponse{
				Status: "out_of_stock",
				ItemsInfo: []model.CheckValidStockResponse{
					{Sku: model.Sku{
						ID:              uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
						SkuName:         "",
						ProductName:     "Wiegand Paul",
						Quantity:        1, // so luong ma khach request len
						Media:           pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/fa825178-3d63-417d-9913-9fe5959b58df/image/67ce83e8-3a1e-4dbc-b5e1-75a704a818e4.jpeg"},
						SellingPrice:    50000,
						NormalPrice:     50000,
						OldSellingPrice: 0,
						OldNormalPrice:  0,
						Uom:             "Vé",
						SkuCode:         "z6",
						Barcode:         "8650",
						CanPickQuantity: 95,
						Type:            "stock_non_varriant",
					},
						Stock: nil,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := CheckCanPickQuantity(tt.args.userID, tt.args.req, tt.args.mapItem)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckCanPickQuantity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("CheckCanPickQuantity() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestCheckSkuHasStock(t *testing.T) {
	conf.SetEnv()

	type args struct {
		userID string
		req    []model.OrderItem
	}
	tests := []struct {
		name    string
		args    args
		wantRes []string
		wantErr bool
	}{
		// TODO: Add test cases
		{
			name: "happy flow TestCheckSkuHasStock stock",
			args: args{
				userID: "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
				req: []model.OrderItem{
					{
						SkuID: uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
					},
				},
			},
			wantRes: []string{"a1513820-1f4d-4321-a25a-84fc09d9ae7d	"},
			wantErr: false,
		},
		{
			name: "happy flow TestCheckSkuHasStock non_stock",
			args: args{
				userID: "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
				req: []model.OrderItem{
					{
						SkuID: uuid.MustParse("4c7c9d50-2e25-4959-a5ab-8efded2336dc"),
					},
				},
			},
			wantRes: nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRs, err := CheckSkuHasStock(tt.args.userID, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckSkuHasStock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRs, tt.wantRes) {
				t.Errorf("CheckSkuHasStock() gotRs = %v, want %v", gotRs, tt.wantRes)
			}
		})
	}
}

func TestCurrentUser(t *testing.T) {
	type args struct {
		c *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    uuid.UUID
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CurrentUser(tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("CurrentUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CurrentUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserHasBusiness(t *testing.T) {
	type args struct {
		userId     string
		businessID string
	}
	tests := []struct {
		name    string
		args    args
		wantRes []UserHasBusiness
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := GetUserHasBusiness(tt.args.userId, tt.args.businessID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserHasBusiness() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("GetUserHasBusiness() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestPushConsumer(t *testing.T) {
	conf.SetEnv()

	type args struct {
		consumer ConsumerRequest
	}
	request := model.SendEmailRequest{
		ID:       "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
		State:    "complete",
		UserRole: strconv.Itoa(ADMIN_ROLE),
	}
	s, _ := json.Marshal(request)

	tests := []struct {
		name    string
		args    args
		wantRes []interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "happy flow TestPushConsumer",
			args: args{
				consumer: ConsumerRequest{
					Topic: "finan-order:send-email-order",
					Body:  string(s),
				},
			},
			wantRes: nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := PushConsumer(tt.args.consumer)
			if (err != nil) != tt.wantErr {
				t.Errorf("PushConsumer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("PushConsumer() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}

func TestSendAutoChatWhenUpdateOrder(t *testing.T) {
	type args struct {
		userID         string
		typeMess       string
		orderNumber    string
		messageContent string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases
		{
			name: "happy flow SendAutoChatWhenUpdateOrder",
			args: args{
				userID:         "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
				typeMess:       NOTI_TYPE_UPDATE_ORDER,
				orderNumber:    "4HPU9PAAE",
				messageContent: "test send auto chat when update Order",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func TestString(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name string
		args args
		want *string
	}{
		// TODO: Add test cases.
		{
			name: "happy flow convert string to pointer String",
			args: args{
				in: "test convert pointer string",
			},
			want: valid.StringPointer("test convert pointer string"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := String(tt.args.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUUID(t *testing.T) {
	type args struct {
		req *uuid.UUID
	}
	tests := []struct {
		name string
		args args
		want uuid.UUID
	}{
		// TODO: Add test cases.
		{
			name: "happy flow TestUUID",
			args: args{
				req: valid.UUIDPointer(uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d")),
			},
			want: uuid.MustParse("a1513820-1f4d-4321-a25a-84fc09d9ae7d"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UUID(tt.args.req); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UUID() = %v, want %v", got, tt.want)
			}
		})
	}
}
