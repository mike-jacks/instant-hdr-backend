package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type ProcessHandler struct {
	autoenhanceClient *autoenhance.Client
	dbClient          *supabase.DatabaseClient
	realtimeClient    *supabase.RealtimeClient
}

func NewProcessHandler(autoenhanceClient *autoenhance.Client, dbClient *supabase.DatabaseClient, realtimeClient *supabase.RealtimeClient) *ProcessHandler {
	return &ProcessHandler{
		autoenhanceClient: autoenhanceClient,
		dbClient:          dbClient,
		realtimeClient:    realtimeClient,
	}
}

// Process godoc
// @Summary     Process images with HDR merge
// @Description Initiates HDR processing and merging of uploaded images using AutoEnhance AI. Groups brackets into images and processes them with the specified enhancement options.
// @Tags        process
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Param       request body models.ProcessRequest false "Processing options. enhance_type defaults to 'property' for real estate photography."
// @Success     200 {object} models.ProcessResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/process [post]
func (h *ProcessHandler) Process(c *gin.Context) {
	if h.dbClient == nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "database not available"})
		return
	}

	userIDStr, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "user id not found"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid user id"})
		return
	}

	orderIDStr := c.Param("order_id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid order id"})
		return
	}

	// Verify order belongs to user
	order, err := h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	var req models.ProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid request body",
			Message: err.Error(),
		})
		return
	}

	// Get brackets for this order
	brackets, err := h.dbClient.GetBracketsByOrderID(orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get brackets",
			Message: err.Error(),
		})
		return
	}

	if len(brackets) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "no brackets found",
			Message: "please upload images before processing",
		})
		return
	}

	// Group brackets into images (for now, group all brackets into one image)
	// In the future, this could be smarter about grouping by shot
	bracketIDs := make([]string, len(brackets))
	for i, bracket := range brackets {
		bracketIDs[i] = bracket.BracketID
	}

	// Set default enhance_type if not provided
	enhanceType := req.EnhanceType
	if enhanceType == "" {
		enhanceType = "property" // Default for real estate
	}

	// Build process request
	processReq := autoenhance.OrderHDRProcessIn{
		EnhanceType: enhanceType,
		Images: []autoenhance.OrderImageIn{
			{
				BracketIDs: bracketIDs,
			},
		},
	}

	// Set optional fields
	if req.SkyReplacement != nil {
		processReq.SkyReplacement = req.SkyReplacement
	} else {
		skyReplacement := true // Default for real estate
		processReq.SkyReplacement = &skyReplacement
	}

	if req.VerticalCorrection != nil {
		processReq.VerticalCorrection = req.VerticalCorrection
	} else {
		verticalCorrection := true // Default
		processReq.VerticalCorrection = &verticalCorrection
	}

	if req.LensCorrection != nil {
		processReq.LensCorrection = req.LensCorrection
	} else {
		lensCorrection := true // Default
		processReq.LensCorrection = &lensCorrection
	}

	if req.WindowPullType != "" {
		processReq.WindowPullType = &req.WindowPullType
	} else {
		windowPullType := "ONLY_WINDOWS" // Default for real estate
		processReq.WindowPullType = &windowPullType
	}

	if req.Upscale != nil {
		processReq.Upscale = req.Upscale
	}

	if req.Privacy != nil {
		processReq.Privacy = req.Privacy
	}

	// Initiate processing with retry
	err = h.autoenhanceClient.RetryWithBackoff(func() error {
		_, err := h.autoenhanceClient.ProcessOrder(order.ID.String(), processReq)
		return err
	}, 3)
	if err != nil {
		h.dbClient.UpdateOrderError(orderID, err.Error())
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to initiate processing",
			Message: err.Error(),
		})
		return
	}

	// Update order status
	h.dbClient.UpdateOrderStatus(orderID, "processing", 0)

	// Publish processing_started event
	h.realtimeClient.PublishOrderEvent(orderID, "processing_started",
		supabase.ProcessingStartedPayload(orderID, ""))

	// Build processing params for response
	processingParams := map[string]interface{}{
		"enhance_type":        processReq.EnhanceType,
		"sky_replacement":     processReq.SkyReplacement,
		"vertical_correction": processReq.VerticalCorrection,
		"lens_correction":     processReq.LensCorrection,
		"window_pull_type":    processReq.WindowPullType,
		"upscale":             processReq.Upscale,
		"privacy":             processReq.Privacy,
		"total_brackets":      len(bracketIDs),
	}

	response := models.ProcessResponse{
		OrderID:          orderID.String(),
		Status:           "processing",
		Message:          "Order processing started successfully",
		ProcessingParams: processingParams,
	}

	c.JSON(http.StatusOK, response)
}
