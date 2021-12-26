package utils

import (
	"context"
	"finan/ms-order-management/pkg/valid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"reflect"
	"testing"
)

func TestCheckPermission(t *testing.T) {
	type args struct {
		ctx        context.Context
		userId     string
		businessID string
		role       string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: UserID & businessID in
		{
			// user request is owner shop
			name: "happy flow userID has businessID",
			args: args{
				ctx:        context.Background(),
				userId:     "a354186a-8b2c-43f9-9ff0-3c404833d5a1",
				businessID: "460d9e32-b7fc-47b9-81fa-9aef6f2fce38",
				role:       "64",
			},
			wantErr: false,
		},
		{
			// user request isn't owner shop
			name: "sad flow userID hasn't businessID",
			args: args{
				ctx:        context.Background(),
				userId:     "27302455-9327-44ab-bb59-36e9b4ebea21",
				businessID: "460d9e32-b7fc-47b9-81fa-9aef6f2fce38",
				role:       "64",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CheckPermission(tt.args.ctx, tt.args.userId, tt.args.businessID, tt.args.role); (err != nil) != tt.wantErr {
				t.Errorf("CheckPermission() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseIDFromUri(t *testing.T) {
	type args struct {
		c *gin.Context
	}

	var c *gin.Context
	tests := []struct {
		name string
		args args
		want *uuid.UUID
	}{
		// TODO: Add test cases
		{
			name: "happy flow: TestParseIDFromUri",
			args: args{
				c: c,
			},
			want: valid.UUIDPointer(uuid.MustParse("27302455-9327-44ab-bb59-36e9b4ebea21")),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseIDFromUri(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseIDFromUri() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResizeImage(t *testing.T) {
	type args struct {
		link string
		w    int
		h    int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "happy flow TestResizeImage",
			args: args{
				link: "https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/1d78990d-33ef-4278-94a9-881c7c57d4ae/image/default_avatar_shop.png",
				w:    128,
				h:    128,
			},
			want: "https://d3hr4eej8cfgwy.cloudfront.net/v2/128x128/finan-dev/1d78990d-33ef-4278-94a9-881c7c57d4ae/image/default_avatar_shop.png",
		},
		{
			name: "happy flow TestResizeImage",
			args: args{
				link: "https://d3hr4eej8cfgwy.cloudfront.net/finan-dev/1d78990d-33ef-4278-94a9-881c7c57d4ae/image/default_avatar_shop.png",
				w:    240,
				h:    0,
			},
			want: "https://d3hr4eej8cfgwy.cloudfront.net/v2/w240/finan-dev/1d78990d-33ef-4278-94a9-881c7c57d4ae/image/default_avatar_shop.png",
		},
		{
			name: "happy flow TestResizeImage",
			args: args{
				link: "https://internal-api-lark-file.larksuite.com/api/image/keys/img_v2_10b34b00-cce9-44d9-af59-a10557b1742h?message_id=7044941922342813701",
				w:    128,
				h:    128,
			},
			want: "https://internal-api-lark-file.larksuite.com/api/image/keys/img_v2_10b34b00-cce9-44d9-af59-a10557b1742h?message_id=7044941922342813701",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResizeImage(tt.args.link, tt.args.w, tt.args.h); got != tt.want {
				t.Errorf("ResizeImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStrDelimitForSum(t *testing.T) {
	type args struct {
		flt      float64
		currency string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{
			name: "happy flow: enough argument float & currency ",
			args: args{
				flt:      5000,
				currency: "vnd",
			},
			want: "5.000 vnd",
		},
		{
			name: "happy flow: have flt but not have currency",
			args: args{
				flt:      5000,
				currency: "",
			},
			want: "5.000",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StrDelimitForSum(tt.args.flt, tt.args.currency); got != tt.want {
				t.Errorf("StrDelimitForSum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getSizeImage(t *testing.T) {
	type args struct {
		w int
		h int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases
		{
			name: "happy flow Test_getSizeImage",
			args: args{
				w: 128,
				h: 128,
			},
			want: "128x128",
		},
		{
			name: "happy flow Test_getSizeImage",
			args: args{
				w: 240,
				h: 0,
			},
			want: "w240",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSizeImage(tt.args.w, tt.args.h); got != tt.want {
				t.Errorf("getSizeImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
