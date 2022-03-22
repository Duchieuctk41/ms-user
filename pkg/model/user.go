package model

type User struct {
	BaseModel
	FullName    string `json:"full_name"`
	PhoneNumber string `json:"phone_number"`
	Address     string `json:"address"`
}
