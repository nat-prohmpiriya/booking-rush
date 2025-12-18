package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/service"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-booking/internal/worker"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/redis"
	"github.com/prohmpiriya/booking-rush-10k-rps/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// QueueHandler handles queue HTTP requests
type QueueHandler struct {
	queueService service.QueueService
	redisClient  *redis.Client // For Pub/Sub subscription in SSE
}

// NewQueueHandler creates a new queue handler
func NewQueueHandler(queueService service.QueueService, redisClient *redis.Client) *QueueHandler {
	return &QueueHandler{
		queueService: queueService,
		redisClient:  redisClient,
	}
}

// JoinQueue handles POST /queue/join
func (h *QueueHandler) JoinQueue(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.join")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.JoinQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid request",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
	)

	result, err := h.queueService.JoinQueue(ctx, userID, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusCreated, result)
}

// GetPosition handles GET /queue/position/:event_id
func (h *QueueHandler) GetPosition(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.position")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	eventID := c.Param("event_id")
	if eventID == "" {
		span.SetStatus(codes.Error, "event_id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "event_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	result, err := h.queueService.GetPosition(ctx, userID, eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// LeaveQueue handles DELETE /queue/leave
func (h *QueueHandler) LeaveQueue(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.leave")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	var req dto.LeaveQueueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "invalid request",
			Code:    "INVALID_REQUEST",
			Message: err.Error(),
		})
		return
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", req.EventID),
	)

	result, err := h.queueService.LeaveQueue(ctx, userID, &req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// GetQueueStatus handles GET /queue/status/:event_id
func (h *QueueHandler) GetQueueStatus(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.status")
	defer span.End()
	c.Request = c.Request.WithContext(ctx)

	eventID := c.Param("event_id")
	if eventID == "" {
		span.SetStatus(codes.Error, "event_id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "event_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(attribute.String("event_id", eventID))

	result, err := h.queueService.GetQueueStatus(ctx, eventID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		h.handleError(c, err)
		return
	}

	span.SetStatus(codes.Ok, "")
	c.JSON(http.StatusOK, result)
}

// StreamPosition handles GET /queue/position/:event_id/stream (SSE)
// This endpoint uses Redis Pub/Sub to receive real-time queue pass notifications.
// Instead of polling every 500ms (which causes 2000 req/s for 1000 connections),
// it subscribes to a channel and only receives updates when queue passes are issued.
// This reduces Redis load from ~2000 queries/s to ~10 publishes/s (50-200x reduction).
func (h *QueueHandler) StreamPosition(c *gin.Context) {
	ctx, span := telemetry.StartSpan(c.Request.Context(), "handler.queue.stream_position")
	defer span.End()

	userID := c.GetString("user_id")
	if userID == "" {
		span.SetStatus(codes.Error, "unauthorized")
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error: "unauthorized",
			Code:  "UNAUTHORIZED",
		})
		return
	}

	eventID := c.Param("event_id")
	if eventID == "" {
		span.SetStatus(codes.Error, "event_id required")
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: "event_id required",
			Code:  "INVALID_REQUEST",
		})
		return
	}

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.String("event_id", eventID),
	)

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // Disable nginx buffering

	// FAST PATH: Check if user already has queue pass
	result, err := h.queueService.GetPosition(ctx, userID, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrNotInQueue) {
			data, _ := json.Marshal(map[string]interface{}{
				"event":   "not_in_queue",
				"message": "User is not in queue",
			})
			c.Writer.WriteString(fmt.Sprintf("event: error\ndata: %s\n\n", data))
			c.Writer.Flush()
			span.SetStatus(codes.Error, "not_in_queue")
			return
		}
		// Other error - return error response
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	// If already has queue pass, send and close immediately
	if result.IsReady && result.QueuePass != "" {
		data, _ := json.Marshal(result)
		c.Writer.WriteString(fmt.Sprintf("event: position\ndata: %s\n\n", data))
		c.Writer.Flush()
		span.SetStatus(codes.Ok, "already_ready")
		return
	}

	// Send initial position
	data, _ := json.Marshal(result)
	c.Writer.WriteString(fmt.Sprintf("event: position\ndata: %s\n\n", data))
	c.Writer.Flush()

	// Use Pub/Sub if Redis client is available, otherwise fallback to polling
	if h.redisClient != nil {
		h.streamWithPubSub(c, ctx, userID, eventID)
	} else {
		h.streamWithPolling(c, ctx, userID, eventID)
	}

	span.SetStatus(codes.Ok, "")
}

// streamWithPubSub uses Redis Pub/Sub to wait for queue pass notification
// Uses per-user channel for targeted delivery - no broadcast amplification
func (h *QueueHandler) streamWithPubSub(c *gin.Context, ctx context.Context, userID, eventID string) {
	// Subscribe to queue pass channel for this USER (targeted delivery)
	// Trade-off: More Redis connections but no broadcast storm
	channel := worker.QueuePassChannelKey(eventID, userID)
	pubsub := h.redisClient.Subscribe(ctx, channel)
	defer pubsub.Close()

	// Get the channel for receiving messages
	msgChan := pubsub.Channel()

	// Create keepalive ticker (send position every 15 seconds to prevent timeout)
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	// Maximum wait time (5 minutes - should match queue pass TTL)
	maxWait := time.NewTimer(5 * time.Minute)
	defer maxWait.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return

		case msg := <-msgChan:
			// Received queue pass notification - this is already for this user (per-user channel)
			var queuePassMsg worker.QueuePassReadyMessage
			if err := json.Unmarshal([]byte(msg.Payload), &queuePassMsg); err != nil {
				// Invalid message, continue waiting
				continue
			}

			// No filtering needed - per-user channel guarantees this is for us

			// Send queue pass to client
			result := &dto.QueuePositionResponse{
				Position:           0,
				TotalInQueue:       0,
				IsReady:            true,
				QueuePass:          queuePassMsg.QueuePass,
				QueuePassExpiresAt: time.Unix(queuePassMsg.ExpiresAt, 0),
			}
			data, _ := json.Marshal(result)
			c.Writer.WriteString(fmt.Sprintf("event: position\ndata: %s\n\n", data))
			c.Writer.Flush()
			return // Done, close connection

		case <-keepalive.C:
			// Send keepalive with current position (low frequency)
			result, err := h.queueService.GetPosition(ctx, userID, eventID)
			if err != nil {
				if errors.Is(err, domain.ErrNotInQueue) {
					data, _ := json.Marshal(map[string]interface{}{
						"event":   "not_in_queue",
						"message": "User is not in queue",
					})
					c.Writer.WriteString(fmt.Sprintf("event: error\ndata: %s\n\n", data))
					c.Writer.Flush()
					return
				}
				// Send keepalive heartbeat
				c.Writer.WriteString(":keepalive\n\n")
				c.Writer.Flush()
				continue
			}

			// If got queue pass (race condition - might have been set between publishes)
			if result.IsReady && result.QueuePass != "" {
				data, _ := json.Marshal(result)
				c.Writer.WriteString(fmt.Sprintf("event: position\ndata: %s\n\n", data))
				c.Writer.Flush()
				return
			}

			// Send position update
			data, _ := json.Marshal(result)
			c.Writer.WriteString(fmt.Sprintf("event: position\ndata: %s\n\n", data))
			c.Writer.Flush()

		case <-maxWait.C:
			// Timeout - close connection
			data, _ := json.Marshal(map[string]interface{}{
				"event":   "timeout",
				"message": "Queue wait timeout",
			})
			c.Writer.WriteString(fmt.Sprintf("event: error\ndata: %s\n\n", data))
			c.Writer.Flush()
			return
		}
	}
}

// streamWithPolling is the fallback method using polling (for when Redis Pub/Sub is unavailable)
func (h *QueueHandler) streamWithPolling(c *gin.Context, ctx context.Context, userID, eventID string) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			result, err := h.queueService.GetPosition(ctx, userID, eventID)
			if err != nil {
				if errors.Is(err, domain.ErrNotInQueue) {
					data, _ := json.Marshal(map[string]interface{}{
						"event":   "not_in_queue",
						"message": "User is not in queue",
					})
					fmt.Fprintf(w, "event: error\ndata: %s\n\n", data)
					c.Writer.Flush()
					return false
				}
				return true
			}

			data, _ := json.Marshal(result)
			fmt.Fprintf(w, "event: position\ndata: %s\n\n", data)
			c.Writer.Flush()

			if result.IsReady && result.QueuePass != "" {
				return false
			}
			return true
		}
	})
}

// handleError converts domain errors to HTTP responses
func (h *QueueHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotInQueue):
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "NOT_IN_QUEUE",
		})
	case errors.Is(err, domain.ErrAlreadyInQueue):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "ALREADY_IN_QUEUE",
		})
	case errors.Is(err, domain.ErrQueueFull):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "QUEUE_FULL",
		})
	case errors.Is(err, domain.ErrQueueNotOpen):
		c.JSON(http.StatusConflict, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "QUEUE_NOT_OPEN",
		})
	case errors.Is(err, domain.ErrInvalidQueueToken):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_TOKEN",
		})
	case errors.Is(err, domain.ErrInvalidUserID):
		c.JSON(http.StatusForbidden, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "FORBIDDEN",
		})
	case errors.Is(err, domain.ErrInvalidEventID):
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error: err.Error(),
			Code:  "INVALID_EVENT_ID",
		})
	default:
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error: "internal server error",
			Code:  "INTERNAL_ERROR",
		})
	}
}
