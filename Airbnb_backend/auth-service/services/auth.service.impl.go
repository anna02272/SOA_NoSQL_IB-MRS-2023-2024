package services

import (
	"auth-service/domain"
	"auth-service/utils"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"net/http"
	"strings"
)

type AuthServiceImpl struct {
	collection  *mongo.Collection
	ctx         context.Context
	userService UserService
}

func NewAuthService(collection *mongo.Collection, ctx context.Context, userService UserService) AuthService {
	return &AuthServiceImpl{collection, ctx, userService}
}

func (uc *AuthServiceImpl) Login(*domain.LoginInput) (*domain.User, error) {
	return nil, nil
}

func (uc *AuthServiceImpl) Registration(user *domain.User) (*domain.UserResponse, error) {
	hashedPassword, _ := utils.HashPassword(user.Password)
	user.Password = hashedPassword
	code := randstr.String(20)
	verificationCode := utils.Encode(code)

	credentials := &domain.Credentials{
		ID:               primitive.NewObjectID(),
		Username:         user.Username,
		Password:         hashedPassword,
		UserRole:         user.UserRole,
		Email:            user.Email,
		Verified:         false,
		VerificationCode: verificationCode,
	}
	res, err := uc.collection.InsertOne(uc.ctx, credentials)
	if err != nil {
		return nil, err
	}

	err = uc.userService.SendUserToProfileService(user)
	if err != nil {
		return nil, err
	}

	// Send Verification Email
	if err := uc.SendVerificationEmail(credentials); err != nil {
		log.Printf("Error sending verification email: %v", err)
		return nil, err
	}

	var newUser *domain.UserResponse
	query := bson.M{"_id": res.InsertedID}

	err = uc.collection.FindOne(uc.ctx, query).Decode(&newUser)
	if err != nil {
		return nil, err
	}
	return newUser, nil

}

func (uc *AuthServiceImpl) SendVerificationEmail(credentials *domain.Credentials) error {
	var username = credentials.Username
	if strings.Contains(username, " ") {
		username = strings.Split(username, " ")[1]
	}

	emailData := utils.EmailData{
		URL:      "localhost:8080/api/auth/verifyEmail/" + credentials.VerificationCode,
		Username: username,
		Subject:  "Your account verification code",
	}

	return utils.SendEmail(credentials, &emailData)
}

func (uc *AuthServiceImpl) SendPasswordResetToken(credentials *domain.Credentials) error {
	var username = credentials.Username
	if strings.Contains(username, " ") {
		username = strings.Split(username, " ")[1]
	}

	emailData := utils.EmailData{
		URL:      "localhost:8080/api/auth/resetPassword/" + credentials.PasswordResetToken,
		Username: username,
		Subject:  "Your account password reset code (valid for 10min)",
	}

	return utils.SendEmail(credentials, &emailData)
}

func (uc *AuthServiceImpl) ResendVerificationEmail(ctx *gin.Context) {
	email := ctx.Param("email")

	user, err := uc.userService.FindCredentialsByEmail(email)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": "User not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "fail", "message": "Internal Server Error"})
		return
	}

	// Generate a new verification code
	code := randstr.String(20)
	verificationCode := utils.Encode(code)

	// Update the user in the database with the new verification code
	_, err = uc.collection.UpdateOne(
		context.TODO(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"verificationCode": verificationCode, "verified": false}},
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "fail", "message": "Internal Server Error"})
		return
	}

	user.VerificationCode = verificationCode

	// Send the verification email
	if err := uc.SendVerificationEmail(user); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"status": "fail", "message": "Error sending verification email"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "Verification email resent successfully"})
}
