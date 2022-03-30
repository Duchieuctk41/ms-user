package model

type User struct {
	BaseModel
	FullName    string `json:"full_name" gorm:"column:full_name; type:varchar(255)"`
	PhoneNumber string `json:"phone_number" gorm:"column:phone_number; type:varchar(50);" sql:"index"`
	Email       string `json:"email" gorm:"column:email; type:varchar(500); not null" sql:"index"`
	DisplayName string `json:"display_name" gorm:"type:varchar(255);"`
	Bio         string `json:"bio" gorm:"type:varchar(500);"`
	AccountType string `json:"account_type" gorm:"type:varchar(50);"`
	Images      string `json:"images" gorm:"column:images; type:varchar(255);"`
	Link        string `json:"link" gorm:"type:varchar(500)"`
	Password    string `json:"password,omitempty" gorm:"column:password; type:varchar(255); not null"`
}

type CreateUserReq struct {
	Email           *string `json:"email" valid:"Required"`
	Password        *string `json:"password" valid:"Required"`
	ConfirmPassword *string `json:"confirm_password" valid:"Required"`
}
