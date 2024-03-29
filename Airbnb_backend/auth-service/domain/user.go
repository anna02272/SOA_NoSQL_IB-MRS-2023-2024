package domain

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Password string             `bson:"password" json:"password"`
	Email    string             `bson:"email" json:"email" validate:"required,email"`
	Name     string             `bson:"name" json:"name"`
	Lastname string             `bson:"lastname" json:"lastname"`
	Address  Address            `bson:"address" json:"address"`
	Age      int                `bson:"age,omitempty" json:"age"`
	Gender   Gender             `bson:"gender,omitempty" json:"gender"`
	UserRole UserRole           `bson:"userRole" json:"userRole"`
}
type CurrentUser struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Username string             `bson:"username" json:"username"`
	Email    string             `bson:"email" json:"email" validate:"required,email"`
	Name     string             `bson:"name" json:"name"`
	Lastname string             `bson:"lastname" json:"lastname"`
	Address  Address            `bson:"address" json:"address"`
	Age      int                `bson:"age,omitempty" json:"age"`
	Gender   Gender             `bson:"gender,omitempty" json:"gender"`
	UserRole UserRole           `bson:"userRole" json:"userRole"`
}

type Credentials struct {
	ID                 primitive.ObjectID `bson:"_id" json:"id"`
	Username           string             `bson:"username" json:"username"`
	Password           string             `bson:"password" json:"password"`
	UserRole           UserRole           `bson:"userRole" json:"userRole"`
	Email              string             `bson:"email" json:"email" validate:"required,email"`
	VerificationCode   string             `bson:"verificationCode" json:"verificationCode"`
	VerifyAt           time.Time          `bson:"verifyAt" json:"verifyAt"`
	PasswordResetToken string             `bson:"passwordResetToken" json:"passwordResetToken"`
	PasswordResetAt    time.Time          `bson:"passwordResetAt" json:"passwordResetAt"`
	Verified           bool               `bson:"verified" json:"verified"`
}

type LoginInput struct {
	Email    string `json:"email" bson:"email" `
	Password string `json:"password" bson:"password"`
}
type PasswordChangeRequest struct {
	CurrentPassword    string `json:"current_password" bson:"password"`
	NewPassword        string `json:"new_password" bson:"password" binding:"required"`
	ConfirmNewPassword string `json:"confirm_new_password" binding:"required"`
}
type UserResponse struct {
	Username string   `bson:"username" json:"username"`
	Email    string   `bson:"email" json:"email" validate:"required,email"`
	UserRole UserRole `bson:"userRole" json:"userRole"`
}

type Address struct {
	Street  string `bson:"street,omitempty" json:"street"`
	City    string `bson:"city,omitempty" json:"city"`
	Country string `bson:"country,omitempty" json:"country"`
}

type Gender string

const (
	Male   = "Male"
	Female = "Female"
	Other  = "Other"
)

type UserRole string

const (
	Guest = "Guest"
	Host  = "Host"
)

type ForgotPasswordInput struct {
	Email string `bson:"email" json:"email" binding:"required"`
}

type ResetPasswordInput struct {
	Password        string `bson:"password" json:"password" binding:"required"`
	PasswordConfirm string `bson:"passwordConfirm" json:"passwordConfirm" binding:"required"`
}
