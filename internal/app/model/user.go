package model

import (
	"time"

	"gorm.io/gorm"
)

type UserRole string

const (
	RoleUser  UserRole = "user"
	RoleAdmin UserRole = "admin"
)

type User struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	Email        string         `gorm:"uniqueIndex;not null" json:"email"`
	PasswordHash string         `gorm:"not null" json:"-"`
	Name         string         `gorm:"not null" json:"name"`
	Phone        string         `json:"phone"`
	Role         UserRole       `gorm:"type:varchar(20);default:'user'" json:"role"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Orders    []Order    `gorm:"foreignKey:UserID" json:"orders,omitempty"`
	CartItems []CartItem `gorm:"foreignKey:UserID" json:"cart_items,omitempty"`
}

func (User) TableName() string {
	return "users"
}
