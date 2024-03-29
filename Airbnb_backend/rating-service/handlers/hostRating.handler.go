package handlers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	logger "github.com/sirupsen/logrus"

	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sony/gobreaker"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"rating-service/domain"
	"rating-service/services"
	"strconv"
	"time"
)

type HostRatingHandler struct {
	hostRatingService services.HostRatingService
	DB                *mongo.Collection
	Tracer            trace.Tracer
	CircuitBreaker    *gobreaker.CircuitBreaker
	logger            *logger.Logger
}

func NewHostRatingHandler(hostRatingService services.HostRatingService, db *mongo.Collection, tr trace.Tracer, logger *logger.Logger) HostRatingHandler {
	circuitBreaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name: "HTTPSRequest",
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			fmt.Printf("Circuit Breaker state changed from %s to %s\n", from, to)
		},
	})
	return HostRatingHandler{
		hostRatingService: hostRatingService,
		DB:                db,
		Tracer:            tr,
		CircuitBreaker:    circuitBreaker,
		logger:            logger,
	}

}

func (s *HostRatingHandler) RateHost(c *gin.Context) {
	spanCtx, span := s.Tracer.Start(c.Request.Context(), "HostRatingHandler.RateHost")
	defer span.End()

	hostID := c.Param("hostId")

	token := c.GetHeader("Authorization")
	currentUser, err := s.getCurrentUserFromAuthService(token, spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Failed to obtain current user information. Try again later")
		span.SetStatus(codes.Error, "Failed to obtain current user information. Try again later")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to obtain current user information. Try again later"})
		return
	}

	hostUser, err := s.getUserByIDFromAuthService(hostID, spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Failed to obtain host information. Try again later")
		span.SetStatus(codes.Error, "Failed to obtain host information.Try again later.")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to obtain host information.Try again later."})
		return
	}

	urlCheckReservations := "https://res-server:8082/api/reservations/getAll"

	timeout := 2000 * time.Second
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	respRes, errRes := s.HTTPSperformAuthorizationRequestWithCircuitBreaker(spanCtx, token, urlCheckReservations)
	if errRes != nil {

		if errors.Is(err, gobreaker.ErrOpenState) {
			// Circuit is open
			s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Circuit is open. Auth service is not available.")
			span.SetStatus(codes.Error, "Circuit is open. Auth service is not available.")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Auth service is not available. Try again later."})
		}

		if spanCtx.Err() == context.DeadlineExceeded {
			span.SetStatus(codes.Error, "Request timed out")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get reservations. Try again later"})
			return
		}
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Failed to get reservations. Try again later")
		span.SetStatus(codes.Error, "Failed to get reservations. Try again later.")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get reservations. Try again later."})
		return
	}

	defer respRes.Body.Close()

	if respRes.StatusCode != http.StatusOK {
		span.SetStatus(codes.Error, "You cannot rate this host. You don't have reservations from him")
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot rate this host. You don't have reservations from him"})
		return
	}

	decoder := json.NewDecoder(respRes.Body)
	var reservations []domain.ReservationByGuest
	if err := decoder.Decode(&reservations); err != nil {
		fmt.Println(err)
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Failed to decode reservations")
		span.SetStatus(codes.Error, "Failed to decode reservations")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to decode reservations"})
		return
	}

	if len(reservations) == 0 {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("You cannot rate this host. You don't have reservations from him")
		span.SetStatus(codes.Error, "You cannot rate this host. You don't have reservations from him")
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot rate this host. You don't have reservations from him"})
		return
	}

	canRate := false
	for _, reservation := range reservations {
		if reservation.AccommodationHostId == hostID {
			canRate = true
			break
		}
	}

	if !canRate {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("You cannot rate this host. You don't have reservations from him")
		span.SetStatus(codes.Error, "You cannot rate this host. You don't have reservations from him")
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot rate this host. You don't have reservations from him"})
		return
	}

	var requestBody struct {
		Rating int `json:"rating"`
	}

	if err := c.BindJSON(&requestBody); err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Invalid JSON request")
		span.SetStatus(codes.Error, "Invalid JSON request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON request"})
		return
	}

	currentDateTime := primitive.NewDateTimeFromTime(time.Now())
	id := primitive.NewObjectID()

	newRateHost := &domain.RateHost{
		ID:          id,
		Host:        hostUser,
		Guest:       currentUser,
		DateAndTime: currentDateTime,
		Rating:      domain.Rating(requestBody.Rating),
	}

	err = s.hostRatingService.SaveRating(newRateHost, spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Failed to save rating")
		span.SetStatus(codes.Error, "Failed to save rating")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to save rating"})
		return
	}

	hostIDString := hostUser.ID.Hex()

	notificationPayload := map[string]interface{}{
		"host_id":           hostIDString,
		"host_email":        hostUser.Email,
		"notification_text": "Dear " + hostUser.Username + "\n you have been rated. You got " + strconv.Itoa(requestBody.Rating) + " stars from " + currentUser.Username + "!",
	}

	notificationPayloadJSON, err := json.Marshal(notificationPayload)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Error creating notification payload")
		span.SetStatus(codes.Error, "Error creating notification payload")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error creating notification payload"})
		return
	}

	notificationURL := "https://notifications-server:8089/api/notifications/create"

	timeout = 2000 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := s.HTTPSperformAuthorizationRequestWithContextAndBodyAccCircuitBreaker(spanCtx, token, notificationURL, "POST", notificationPayloadJSON)
	if err != nil {
		if errors.Is(err, gobreaker.ErrOpenState) {
			// Circuit is open
			s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Circuit is open. Auth service is not available.")
			span.SetStatus(codes.Error, "Circuit is open. Auth service is not available.")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Auth service is not available. Try again later."})
		}
		s.logger.WithError(err).Error("Error creating notification request")
		span.SetStatus(codes.Error, "Error creating notification request")
		if ctx.Err() == context.DeadlineExceeded {
			s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Error("Request timed out")
			span.SetStatus(codes.Error, "Request timed out")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Notification service not available"})
			return
		}
		s.logger.WithError(err).Error("Notification service not available.")
		span.SetStatus(codes.Error, "Notification service not available.")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Notification service not available."})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Errorf("Error creating notification. Status code: %d", resp.StatusCode)
		span.SetStatus(codes.Error, "Error creating notification")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error creating notification"})
		return
	}
	s.logger.WithFields(logger.Fields{"path": "rating/RateHost"}).Info("Rating successfully saved")
	span.SetStatus(codes.Ok, "Rating successfully saved")
	c.JSON(http.StatusCreated, gin.H{"message": "Rating successfully saved", "rating": newRateHost})
}

func (s *HostRatingHandler) DeleteRating(c *gin.Context) {
	spanCtx, span := s.Tracer.Start(c.Request.Context(), "HostRatingHandler.DeleteRating")
	defer span.End()

	hostID := c.Param("hostId")

	token := c.GetHeader("Authorization")
	currentUser, err := s.getCurrentUserFromAuthService(token, spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/deleteRating"}).Error("Failed to obtain current user information. Try again later.")
		span.SetStatus(codes.Error, "Failed to obtain current user information.  Try again later.")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to obtain current user information.  Try again later."})
		return
	}
	guestID := currentUser.ID.Hex()

	err = s.hostRatingService.DeleteRating(hostID, guestID, spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/deleteRating"}).Error("Failed to delete rating")
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.logger.WithFields(logger.Fields{"path": "rating/deleteRating"}).Info("Rating successfully deleted")
	span.SetStatus(codes.Ok, "Rating successfully deleted")
	c.JSON(http.StatusOK, gin.H{"message": "Rating successfully deleted"})
}

func (s *HostRatingHandler) GetAllRatings(c *gin.Context) {
	spanCtx, span := s.Tracer.Start(c.Request.Context(), "HostRatingHandler.GetAllRatings")
	defer span.End()

	ratings, averageRating, err := s.hostRatingService.GetAllRatings(spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/getAllRating"}).Error("Failed to get all ratings")
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"ratings":       ratings,
		"averageRating": averageRating,
	}
	s.logger.WithFields(logger.Fields{"path": "rating/getAllRating"}).Info("Got all ratings successfully")
	span.SetStatus(codes.Ok, "Got all ratings successfully")
	c.JSON(http.StatusOK, response)
}

func (s *HostRatingHandler) GetByHostAndGuest(c *gin.Context) {
	spanCtx, span := s.Tracer.Start(c.Request.Context(), "HostRatingHandler.GetByHostAndGuest")
	defer span.End()

	token := c.GetHeader("Authorization")
	currentUser, err := s.getCurrentUserFromAuthService(token, spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/getByHostAndGuest"}).Error("Failed to obtain current user information")
		span.SetStatus(codes.Error, "Failed to obtain current user information")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to obtain current user information"})
		return
	}
	guestID := currentUser.ID.Hex()

	hostID := c.Param("hostId")

	ratings, err := s.hostRatingService.GetByHostAndGuest(hostID, guestID, spanCtx)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/getByHostAndGuest"}).Error("Failed to get ratings by host and guest")
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.logger.WithFields(logger.Fields{"path": "rating/getByHostAndGuest"}).Info("Got ratings by host and guest successfully")
	span.SetStatus(codes.Ok, "Got ratings by host and guest successfully")
	c.JSON(http.StatusOK, gin.H{"ratings": ratings})
}

func (s *HostRatingHandler) getUserByIDFromAuthService(userID string, c context.Context) (*domain.User, error) {
	spanCtx, span := s.Tracer.Start(c, "HostRatingHandler.getUserByIDFromAuthService")
	defer span.End()
	url := "https://auth-server:8080/api/users/getById/" + userID

	timeout := 2000 * time.Second
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := s.HTTPSperformAuthorizationRequestWithCircuitBreaker(spanCtx, "", url)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/getUserByIDFromAuthService"}).Error("Failed to get user by ID from auth service")

		if errors.Is(err, gobreaker.ErrOpenState) {
			// Circuit is open
			span.SetStatus(codes.Error, "Circuit is open. Accommodation service is not available.")
			return nil, errors.New("Accommodation service is not available")
		}

		if spanCtx.Err() == context.DeadlineExceeded {
			span.SetStatus(codes.Error, "Accommodation service not available")
			return nil, errors.New("Accommodation service is not available")
		}

		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		span.SetStatus(codes.Error, "User not found")
		return nil, errors.New("User not found")
	}

	var userResponse domain.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResponse); err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/getUserByIDFromAuthService"}).Error("Failed to decode user response from auth service")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	s.logger.WithFields(logger.Fields{"path": "rating/getUserByIDFromAuthService"}).Info("Got user by id from auth service")
	span.SetStatus(codes.Ok, "Got user by id from auth service")
	user := domain.ConvertToDomainUser(userResponse)
	return &user, nil
}

func (s *HostRatingHandler) getCurrentUserFromAuthService(token string, c context.Context) (*domain.User, error) {
	spanCtx, span := s.Tracer.Start(c, "HostRatingHandler.getCurrentUserFromAuthService")
	defer span.End()
	url := "https://auth-server:8080/api/users/currentUser"
	//handler := AccommodationRatingHandler{}

	timeout := 2000 * time.Second
	_, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resp, err := s.HTTPSperformAuthorizationRequestWithCircuitBreaker(spanCtx, token, url)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/getCurrentUserFromAuthService"}).Error("Failed to get current user from auth service")

		if errors.Is(err, gobreaker.ErrOpenState) {
			// Circuit is open
			span.SetStatus(codes.Error, "Circuit is open. Accommodation service is not available.")
			return nil, errors.New("Accommodation service is not available")
		}

		if spanCtx.Err() == context.DeadlineExceeded {
			span.SetStatus(codes.Error, "Accommodation service not available")
			return nil, errors.New("Accommodation service is not available")
		}

		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		span.SetStatus(codes.Error, "Unauthorized")
		return nil, errors.New("Unauthorized")
	}

	var userResponse domain.UserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResponse); err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/getCurrentUserFromAuthService"}).Error("Failed to decode current user response from auth service")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	s.logger.WithFields(logger.Fields{"path": "rating/getCurrentUserFromAuthService"}).Info("Got current user from auth service")
	span.SetStatus(codes.Ok, "Got current user from auth service")
	user := domain.ConvertToDomainUser(userResponse)
	return &user, nil
}

func (s *HostRatingHandler) HTTPSPerformAuthorizationRequestWithContext(ctx context.Context, token string, url string) (*http.Response, error) {
	_, span := s.Tracer.Start(ctx, "HostRatingHandler.HTTPSPerformAuthorizationRequestWithContext")
	defer span.End()
	s.logger.WithFields(logger.Fields{"path": "rating/HTTPSPerformAuthorizationRequestWithContext"}).Info("Sending HTTP request")
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/HTTPSPerformAuthorizationRequestWithContext"}).Error("Failed to create HTTP request")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	req.Header.Set("Authorization", token)
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
	client := &http.Client{Transport: tr}
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		s.logger.WithFields(logger.Fields{"path": "rating/HTTPSPerformAuthorizationRequestWithContext"}).Error("Failed to perform HTTP request")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	s.logger.WithFields(logger.Fields{"path": "rating/HTTPSPerformAuthorizationRequestWithContext"}).Info("Received HTTP response")
	return resp, nil
}

func (s *HostRatingHandler) HTTPSperformAuthorizationRequestWithContextAndBodyAccCircuitBreaker(
	ctx context.Context, token string, url string, method string, requestBody []byte,
) (*http.Response, error) {
	_, span := s.Tracer.Start(ctx, "HostRatingHandler.HTTPSperformAuthorizationRequestWithContextAndBodyAccCircuitBreaker")
	maxRetries := 3

	// Define a retry operation function
	retryOperation := func() (interface{}, error) {
		// Use the Circuit Breaker to execute the request function
		result, err := s.CircuitBreaker.Execute(func() (interface{}, error) {
			tr := http.DefaultTransport.(*http.Transport).Clone()
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

			req, err := http.NewRequest(method, url, bytes.NewBuffer(requestBody))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Authorization", token)
			otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
			client := &http.Client{Transport: tr}
			resp, err := client.Do(req.WithContext(ctx))
			if err != nil {
				return nil, err
			}

			return resp, nil
		})

		// If there is an error, propagate it
		if err != nil {
			return nil, err
		}

		// Check the type of the result
		resp, ok := result.(*http.Response)
		if !ok {
			return nil, errors.New("unexpected response type from Circuit Breaker")
		}

		return resp, nil
	}

	// Use the retry mechanism
	result, err := s.CircuitBreaker.Execute(func() (interface{}, error) {
		return retryOperationWithExponentialBackoff(ctx, maxRetries, retryOperation)
	})
	if err != nil {
		return nil, err
	}

	resp, ok := result.(*http.Response)
	if !ok {
		s.logger.WithFields(logger.Fields{"path": "rating/HTTPSperformAuthorizationRequestWithContextAndBodyAccCircuitBreaker"}).Error("Unexpected response type from retry operation")
		err := errors.New("unexpected response type from retry operation")
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return resp, nil
}

func (s *HostRatingHandler) HTTPSperformAuthorizationRequestWithCircuitBreaker(ctx context.Context, token string, url string) (*http.Response, error) {
	maxRetries := 3
	type retryOperationFunc func() (interface{}, error)

	retryOperation := retryOperationFunc(func() (interface{}, error) {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", token)
		otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

		client := &http.Client{Transport: tr}
		resp, err := client.Do(req.WithContext(ctx))
		if err != nil {
			return nil, err
		}

		return resp, nil // Return the response as the first value
	})

	// Use an anonymous function to convert the result to the expected type
	result, err := s.CircuitBreaker.Execute(func() (interface{}, error) {
		return retryOperationWithExponentialBackoff(ctx, maxRetries, retryOperation)
	})
	if err != nil {
		// Handle or return the error
		return nil, err
	}

	resp, ok := result.(*http.Response)
	if !ok {
		s.logger.WithFields(logger.Fields{"path": "rating/HTTPSperformAuthorizationRequestWithCircuitBreaker"}).Error("Unexpected response type from Circuit Breaker")
		return nil, errors.New("unexpected response type from Circuit Breaker")
	}

	return resp, nil
}

func ExtractTraceInfoMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
