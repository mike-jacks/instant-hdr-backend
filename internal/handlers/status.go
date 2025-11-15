package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type StatusHandler struct {
	dbClient          *supabase.DatabaseClient
	autoenhanceClient *autoenhance.Client
}

func NewStatusHandler(dbClient *supabase.DatabaseClient, autoenhanceClient *autoenhance.Client) *StatusHandler {
	return &StatusHandler{
		dbClient:          dbClient,
		autoenhanceClient: autoenhanceClient,
	}
}

// GetStatus godoc
// @Summary     Get order status
// @Description Returns the current status and progress of an order. For real-time updates, connect to Supabase Realtime.
// @Tags        status
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Success     200 {object} models.StatusResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /orders/{order_id}/status [get]
func (h *StatusHandler) GetStatus(c *gin.Context) {
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

	order, err := h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	response := models.StatusResponse{
		OrderID:   orderID.String(),
		Status:    order.Status,
		Progress:  order.Progress,
		UpdatedAt: order.UpdatedAt,
	}

	// Fetch AutoEnhance data for real-time status
	if h.autoenhanceClient != nil {
		autoenhanceOrder, err := h.autoenhanceClient.GetOrder(order.ID.String())
		if err == nil {
			response.AutoEnhanceStatus = autoenhanceOrder.Status
			response.TotalImages = int(autoenhanceOrder.TotalImages)
			response.IsProcessing = autoenhanceOrder.IsProcessing
			response.IsMerging = autoenhanceOrder.IsMerging
			response.IsDeleted = autoenhanceOrder.IsDeleted
			
			// Include AutoEnhance's last updated timestamp
			if !autoenhanceOrder.LastUpdatedAt.Time.IsZero() {
				lastUpdated := autoenhanceOrder.LastUpdatedAt.Time
				response.AutoEnhanceLastUpdatedAt = &lastUpdated
			}

			// Convert images to generic map
			if len(autoenhanceOrder.Images) > 0 {
				response.Images = make([]map[string]interface{}, len(autoenhanceOrder.Images))
				for i, img := range autoenhanceOrder.Images {
					imgJSON, _ := json.Marshal(img)
					var imgMap map[string]interface{}
					json.Unmarshal(imgJSON, &imgMap)
					response.Images[i] = imgMap
				}
			}
		}

		// Get brackets info
		brackets, err := h.autoenhanceClient.GetOrderBrackets(order.ID.String())
		if err == nil {
			response.TotalBrackets = len(brackets.Brackets)
			uploadedCount := 0
			for _, bracket := range brackets.Brackets {
				if bracket.IsUploaded {
					uploadedCount++
				}
			}
			response.UploadedBrackets = uploadedCount
		}
	}

	c.JSON(http.StatusOK, response)
}
