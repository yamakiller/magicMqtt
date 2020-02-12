package authdb

import "time"

//AuthUser 授权用户表
type AuthUser struct {
	ClientID string `gorm:"primary_key;type:varchar(64);not null;"`
	UserName string `gorm:"type:varchar(32);not null;index:user_idx;"`
	Password string `gorm:"type:varchar(32);not null;"`
	CreateAt time.Time
	UpdateAt time.Time
}
