package handlers

import (
	"fmt"
	"log"
	"net/http"
	"profile-service/domain"
	"profile-service/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type ProfileHandler struct {
	profileService services.ProfileService
	Tracer         trace.Tracer
	logger         *logrus.Logger
}

func NewProfileHandler(profileService services.ProfileService, tr trace.Tracer, logger *logrus.Logger) ProfileHandler {
	return ProfileHandler{profileService, tr, logger}
}

func (ph *ProfileHandler) CreateProfile(ctx *gin.Context) {
	spanCtx, span := ph.Tracer.Start(ctx.Request.Context(), "ProfileHandler.CreateProfile")
	defer span.End()

	var user *domain.User

	if err := ctx.ShouldBindJSON(&user); err != nil {
		ph.logger.WithFields(logrus.Fields{"path": "profile/createProfile"}).Errorf("Error : %v", err.Error())
		span.SetStatus(codes.Error, err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	err := ph.profileService.Registration(user, spanCtx)
	if err != nil {
		ph.logger.WithFields(logrus.Fields{"path": "profile/createProfile"}).Errorf("Error: %v", err.Error())
		span.SetStatus(codes.Error, err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}
	ph.logger.WithFields(logrus.Fields{"path": "profile/createProfile"}).Info("Profile created successfully")
	span.SetStatus(codes.Ok, "Profile created successfully")
	ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "Profile created successfully"})
}

func (ph *ProfileHandler) DeleteProfile(ctx *gin.Context) {
	spanCtx, span := ph.Tracer.Start(ctx.Request.Context(), "ProfileHandler.DeleteProfile")
	defer span.End()

	email := ctx.Params.ByName("email")
	errP := ph.profileService.FindUserByEmail(email, spanCtx)
	if errP != nil {
		ph.logger.WithFields(logrus.Fields{"path": "profile/deleteProfile"}).Errorf("Error: %v", errP.Error())
		span.SetStatus(codes.Error, errP.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": errP.Error()})
		return
	}

	err := ph.profileService.DeleteUserProfile(email, spanCtx)

	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}
	ph.logger.WithFields(logrus.Fields{"path": "profile/deleteProfile"}).Info("Profile deleted successfully")
	span.SetStatus(codes.Ok, "Profile deleted successfully")
	ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "Profile deleted successfully"})
}
func (ph *ProfileHandler) UpdateUser(ctx *gin.Context) {
	spanCtx, span := ph.Tracer.Start(ctx.Request.Context(), "ProfileHandler.UpdateUser")
	defer span.End()

	var user *domain.User
	log.Println(user)
	if err := ctx.ShouldBindJSON(&user); err != nil {
		ph.logger.WithFields(logrus.Fields{"path": "profile/updateUser"}).Errorf("Error: %v", err.Error())
		span.SetStatus(codes.Error, err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	// Pozovi servis za unos korisnika
	err := ph.profileService.UpdateUser(user, spanCtx)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}
	ph.logger.WithFields(logrus.Fields{"path": "profile/updateUser"}).Info("Profile updated successfully.")
	span.SetStatus(codes.Ok, "Profile updated successfully")
	ctx.JSON(http.StatusOK, gin.H{"status": "success", "message": "Profile updated successfully"})
}
func (ph *ProfileHandler) FindUserByEmail(ctx *gin.Context) {
	spanCtx, span := ph.Tracer.Start(ctx.Request.Context(), "ProfileHandler.FindUserByEmail")
	defer span.End()
	email := ctx.Param("email")
	log.Println(email)
	if email == "" {
		ph.logger.WithFields(logrus.Fields{"path": "profile/findUserByEmail"}).Error("Email is required")
		span.SetStatus(codes.Error, "Email is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	user, err := ph.profileService.FindProfileByEmail(email, spanCtx)
	if err != nil {
		ph.logger.WithFields(logrus.Fields{"path": "profile/findUserByEmail"}).Error("User not found")
		span.SetStatus(codes.Error, "User not found")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	ph.logger.WithFields(logrus.Fields{"path": "profile/findUserByEmail"}).Info("Find user by email successfully.")
	span.SetStatus(codes.Ok, "Found user by email successfully")
	ctx.JSON(http.StatusOK, gin.H{"user": user})
}

// func (ph *ProfileHandler) CheckIsFeatured(ctx *gin.Context) {

// 	email := ctx.Param("email")

// 	if email == "" {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
// 		return
// 	}

// 	user, err := ph.profileService.FindProfileByEmail(email)
// 	if err != nil {
// 		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
// 		return
// 	}

// 	isFeatured, err := ph.profileService.CheckIsFeatured(user)
// 	if err != nil {
// 		ctx.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
// 		return
// 	}
// 	ctx.JSON(http.StatusOK, gin.H{"isFeatured": isFeatured})

// }

func (ph *ProfileHandler) IsFeatured(ctx *gin.Context) {
	hostID := ctx.Param("hostId")
	fmt.Println("hostId in handler: ", hostID)
	featured, err := ph.profileService.IsFeatured(hostID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"featured": featured})
}

func (ph *ProfileHandler) SetFeatured(ctx *gin.Context) {
	hostID := ctx.Param("hostId")
	err := ph.profileService.SetFeatured(hostID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "Host is now featured"})
}

func (ph *ProfileHandler) SetUnfeatured(ctx *gin.Context) {
	hostID := ctx.Param("hostId")
	err := ph.profileService.SetUnfeatured(hostID)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"message": "Host is no longer featured"})
}

func ExtractTraceInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
