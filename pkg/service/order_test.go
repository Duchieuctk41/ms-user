package service

import (
	"context"
	"finan/ms-order-management/conf"
	"finan/ms-order-management/pkg/mocks"
	"finan/ms-order-management/pkg/model"
	"finan/ms-order-management/pkg/repo"
	"finan/ms-order-management/pkg/valid"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"reflect"
	"testing"
)

func TestOrderService_CreateOrder(t *testing.T) {
	conf.SetEnv()

	ctr1 := gomock.NewController(t)
	defer ctr1.Finish()

	type service struct {
		repo repo.PGInterface
	}
	type args struct {
		ctx context.Context
		req model.OrderBody
	}

	// mock data order request
	buyerInfo := model.BuyerInfo{
		PhoneNumber: "+84792452548",
		Name:        "Minh Nhựa",
		Address:     "q9",
	}

	// req service
	createOrderReq := model.OrderBody{
		UserID:            uuid.MustParse("1b0c26d6-e53f-4326-a5c7-8076c24b530d"),
		BusinessID:        valid.UUIDPointer(uuid.MustParse("ad5c698f-8ec4-44d2-98f1-c8df052c8c3b")),
		PromotionCode:     "",
		OrderedGrandTotal: 55000,
		PromotionDiscount: 0,
		DeliveryFee:       0,
		GrandTotal:        55000,
		State:             "waiting_confirm",
		PaymentMethod:     "COD",
		Email:             "howhowhow@gmail.com",
		Note:              "",
		DeliveryMethod:    valid.StringPointer("buyer_pick_up"),
		BuyerReceived:     true,
		CreateMethod:      "buyer",
		Debit: &model.Debit{
			BuyerPay: valid.Float64Pointer(55000),
			Note:     "khong cos note",
			Images:   pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/1624322515880.png"},
		},
		BuyerInfo: &model.BuyerInfo{
			PhoneNumber: "0792452548",
			Name:        "Minh Nhựa",
			Address:     "q9",
		},
		ListOrderItem: []model.OrderItem{
			{
				SkuID:       *valid.StringToPointerUUID("f18e996f-e3e8-4202-891d-0b2906683f22"),
				ProductName: "Cốc uống nước",
				SkuName:     "Cốc uống nước",
				SkuCode:     "SP70",
				ProductImages: pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/1625046188665.png",
					"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/2BDAD91B-4DDD-4C44-BDF2-8DB0619264D2.jpg",
					"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/2C5AAB9D-507F-4509-8921-13FDAA7F593B.jpg",
					"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/EDBCED52-0E4B-43FE-83A7-EAAD0B49F46B.jpg"},
				ProductNormalPrice:  55000,
				ProductSellingPrice: 45000,
				Quantity:            1,
				Note:                "",
			},
		},
		ListProductFast: []model.Product{
			{
				Name:        "hieu thu 3",
				Description: "hihihi",
				IsActive:    true,
				Images: pq.StringArray{"https://d3hr4eej8cfgwy.cloudfront.net/finan/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/1625046188665.png",
					"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/2BDAD91B-4DDD-4C44-BDF2-8DB0619264D2.jpg",
					"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/2C5AAB9D-507F-4509-8921-13FDAA7F593B.jpg",
					"https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/4a954e41-2198-4024-9bd6-a8ac3a0e52a3/image/EDBCED52-0E4B-43FE-83A7-EAAD0B49F46B.jpg"},
				Priority:       1,
				SellingPrice:   10000,
				HistoricalCost: 1000,
				NormalPrice:    30000,
				Quantity:       1,
				Uom:            "1 thung 5 gói",
				IsProductFast:  true,
			},
		},
	}

	// order request
	orderReq := model.OrderBody{
		BusinessID:        valid.UUIDPointer(uuid.MustParse("ad5c698f-8ec4-44d2-98f1-c8df052c8c3b")),
		ContactID:         valid.UUIDPointer(uuid.MustParse("fb9ecb53-cc87-4fa4-81db-798d031105a7")),
		PromotionCode:     "",
		PromotionDiscount: 0,
		DeliveryFee:       0,
		OrderedGrandTotal: 55000,
		GrandTotal:        55000,
		State:             "complete",
		PaymentMethod:     "COD",
		DeliveryMethod:    valid.StringPointer("buyer_pick_up"),
		Note:              "",
		CreateMethod:      "buyer",
		OtherDiscount:     0,
		Email:             "missem9999@gmail.com",
		BuyerInfo:         &buyerInfo,
	}

	// order response
	orderRes := model.Order{
		BusinessID:        uuid.MustParse("ad5c698f-8ec4-44d2-98f1-c8df052c8c3b"),
		ContactID:         uuid.MustParse("fb9ecb53-cc87-4fa4-81db-798d031105a7"),
		PromotionDiscount: 0,
		DeliveryFee:       0,
		OrderedGrandTotal: 55000,
		GrandTotal:        55000,
		State:             "complete",
		PaymentMethod:     "COD",
		DeliveryMethod:    "buyer_pick_up",
		Note:              "",
		CreateMethod:      "buyer",
		BuyerId:           valid.UUIDPointer(uuid.MustParse("a354186a-8b2c-43f9-9ff0-3c404833d5a1")),
		OtherDiscount:     0,
		Email:             "missem9999@gmail.com",
	}
	var tx *gorm.DB

	tests := []struct {
		name    string
		service *service
		args    args
		wantRes interface{}
		wantErr bool
	}{
		// TODO: Add test cases
		{
			name: "happy flow: CreateOrder",
			service: &service{
				repo: func() repo.PGInterface {
					mockIRepo := mocks.NewMockPGInterface(ctr1)
					mockIRepo.EXPECT().CreateOrder(context.Background(), orderReq, tx).Return(orderRes, nil)

					return mockIRepo
				}(),
			},
			args: args{
				ctx: context.Background(),
				req: createOrderReq,
			},
			wantRes: orderRes,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRes, err := tt.service.CreateOrder(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotRes, tt.wantRes) {
				t.Errorf("CreateOrder() gotRes = %v, want %v", gotRes, tt.wantRes)
			}
		})
	}
}
