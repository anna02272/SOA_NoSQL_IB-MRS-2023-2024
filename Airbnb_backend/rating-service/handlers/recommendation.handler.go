package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	logger "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"rating-service/domain"
	"rating-service/services"
)

type RecommendationHandler struct {
	rec    services.RecommendationService
	driver neo4j.DriverWithContext
	Tracer trace.Tracer
	logger *logger.Logger
}
type KeyProduct struct{}

func NewRecommendationHandler(recommendationService services.RecommendationService, driver neo4j.DriverWithContext, tr trace.Tracer, logger *logger.Logger) RecommendationHandler {
	return RecommendationHandler{recommendationService, driver, tr, logger}
}
func (r *RecommendationHandler) CreateUser(c *gin.Context) {
	var user domain.NeoUser
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	r.CreateUserNext(c.Writer, c.Request, &user)

}
func (r *RecommendationHandler) CreateUserNext(rw http.ResponseWriter, h *http.Request, user *domain.NeoUser) {

	if user == nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateUserNext"}).Info("User not found in the context or not of type *domain.NeoUser2")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	err := r.rec.CreateUser(user)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateUserNext"}).Errorf("Database exception: %s", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusCreated)

}
func (r *RecommendationHandler) CreateReservation(c *gin.Context) {
	var reservation domain.ReservationByGuest
	if err := c.ShouldBindJSON(&reservation); err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateReservation"}).Errorf("Error to get reservation info(createReservation): %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r.CreateReservationNext(c.Writer, c.Request, &reservation)

}
func (r *RecommendationHandler) CreateReservationNext(rw http.ResponseWriter, h *http.Request, reservation *domain.ReservationByGuest) {

	if reservation == nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateReservationNext"}).Info("Resevation not found in the context or not of type *domain.Reservation", http.StatusBadRequest)
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	err := r.rec.CreateReservation(reservation)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateReservationNext"}).Errorf("Database exception: %s", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	r.logger.WithFields(logger.Fields{"path": "rating/CreateReservationNext"}).Info("Reservation created ", http.StatusCreated)
	rw.WriteHeader(http.StatusCreated)

}
func (r *RecommendationHandler) CreateAccommodation(c *gin.Context) {

	var accommodation domain.AccommodationRec
	if err := c.ShouldBindJSON(&accommodation); err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateAccommodation"}).Errorf("Error in createAccommodation method %s", err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	r.CreateAccommodationNext(c.Writer, c.Request, &accommodation)

}
func (r *RecommendationHandler) CreateAccommodationNext(rw http.ResponseWriter, h *http.Request, accommodation *domain.AccommodationRec) {

	if accommodation == nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateAccommodationNext"}).Error("Accommodation not found in the context or not of type *domain.AccommodationRec")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	err := r.rec.CreateAccommodation(accommodation)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateAccommodationNext"}).Errorf("Database exception: %s", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusCreated)

}
func (r *RecommendationHandler) CreateRecomRate(c *gin.Context) {

	var rate domain.RateAccommodationRec
	if err := c.ShouldBindJSON(&rate); err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateRecomRate"}).Error("Error in CreateRate method:")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	r.CreateRecomRateNext(c.Writer, c.Request, &rate)

}
func (r *RecommendationHandler) CreateRecomRateNext(rw http.ResponseWriter, h *http.Request, rate *domain.RateAccommodationRec) {

	if rate == nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateRecomRateNext"}).Info("Rate not found in the context or not of type *domain.RateAccommodationRec")
		rw.WriteHeader(http.StatusBadRequest)
		return
	}
	err := r.rec.CreateRate(rate)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/CreateRecomRateNext"}).Errorf("Database exception: %s", err)
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusCreated)

}
func (r *RecommendationHandler) GetRecommendation(ctx *gin.Context) {
	_, span := r.Tracer.Start(ctx.Request.Context(), "RecommendationHandler.GetRecommendation")
	defer span.End()
	id := ctx.Param("id")
	if id == "" {
		r.logger.WithFields(logger.Fields{"path": "rating/GetRecommendationg"}).Error("Id is required")
		span.SetStatus(codes.Error, "Id is required")
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Id is required"})
		return
	}

	acc, result := r.rec.GetRecommendation(id)
	if result != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/GetRecommendationg"}).Error("Accommodation not found")
		span.SetStatus(codes.Error, "Accommodation not found")
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Accommodation not found"})
		return
	}
	r.logger.WithFields(logger.Fields{"path": "rating/GetRecommendationg"}).Info("Found accommodation by id successfully")
	span.SetStatus(codes.Ok, "Found accommodation by id successfully")
	ctx.JSON(http.StatusOK, acc)
}
func (r *RecommendationHandler) DeleteReservation(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	guestId, exists := requestData["guestId"].(string)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guestId is missing or not a string"})
		return
	}
	accommodationId, existss := requestData["accommodationId"].(string)
	if !existss {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accommodationId is missing or not a string"})
		return
	}
	err := r.rec.DeleteReservation(accommodationId, guestId)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/DeleteReservation"}).Errorf("Database exception: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting reservation"})
		return
	}
	r.logger.WithFields(logger.Fields{"path": "rating/DeleteReservation"}).Info("Reservation deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Reservation deleted successfully"})
}

func (r *RecommendationHandler) DeleteRate(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	guestId, exists := requestData["guestId"].(string)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "guestId is missing or not a string"})
		return
	}
	accommodation, existss := requestData["accommodation"].(string)
	if !existss {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accommodation is missing or not a string"})
		return
	}
	err := r.rec.DeleteRate(accommodation, guestId)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/DeleteRate"}).Errorf("Database exception: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting rate"})
		return
	}
	r.logger.WithFields(logger.Fields{"path": "rating/DeleteRate"}).Info("Rate deleted successfully")

	c.JSON(http.StatusOK, gin.H{"message": "Rate deleted successfully"})
}

func (r *RecommendationHandler) DeleteUser(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userId, exists := requestData["userId"].(string)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is missing or not a string"})
		return
	}
	err := r.rec.DeleteUser(userId)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/DeleteUser"}).Errorf("Database exception: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting user"})
		return
	}
	r.logger.WithFields(logger.Fields{"path": "rating/DeleteUser"}).Info("User deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
func (r *RecommendationHandler) DeleteAccommodation(c *gin.Context) {
	var requestData map[string]interface{}
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accommodationId, exists := requestData["accommodationId"].(string)
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "accommodationId is missing or not a string"})
		return
	}
	err := r.rec.DeleteAccommodation(accommodationId)
	if err != nil {
		r.logger.WithFields(logger.Fields{"path": "rating/DeleteAccommodation"}).Errorf("Database exception: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting accommodation"})
		return
	}
	r.logger.WithFields(logger.Fields{"path": "rating/DeleteReservation"}).Info("Accommodation deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Accommodation deleted successfully"})
}
