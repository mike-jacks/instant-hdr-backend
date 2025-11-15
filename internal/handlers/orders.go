package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"instant-hdr-backend/internal/autoenhance"
	"instant-hdr-backend/internal/middleware"
	"instant-hdr-backend/internal/models"
	"instant-hdr-backend/internal/supabase"
)

type OrdersHandler struct {
	autoenhanceClient *autoenhance.Client
	dbClient          *supabase.DatabaseClient
	storageClient     *supabase.StorageClient
}

func NewOrdersHandler(autoenhanceClient *autoenhance.Client, dbClient *supabase.DatabaseClient, storageClient *supabase.StorageClient) *OrdersHandler {
	return &OrdersHandler{
		autoenhanceClient: autoenhanceClient,
		dbClient:          dbClient,
		storageClient:     storageClient,
	}
}

// CreateOrder godoc
// @Summary     Create a new order
// @Description Creates a new AutoEnhance AI order for a listing (real estate shoot). You can optionally provide a custom name for the order.
// @Tags        orders
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       request body models.CreateOrderRequest false "Order name (optional, defaults to 'Order')"
// @Success     200 {object} models.OrderResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders [post]
func (h *OrdersHandler) CreateOrder(c *gin.Context) {
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

	var req models.CreateOrderRequest
	// JSON body is optional - if not provided or invalid, req will just have empty Name
	_ = c.ShouldBindJSON(&req)

	// Use provided name or default
	orderName := req.Name
	if orderName == "" {
		orderName = "Order" // Default name
	}

	// Create AutoEnhance order - let them generate the order_id
	// We'll use that order_id as our primary key
	var autoenhanceOrder *autoenhance.OrderOut
	err = h.autoenhanceClient.RetryWithBackoff(func() error {
		var err error
		// Don't pass order_id (empty string) - let AutoEnhance generate it
		// But do pass the order name
		autoenhanceOrder, err = h.autoenhanceClient.CreateOrder("", orderName)
		return err
	}, 3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to create autoenhance order",
			Message: err.Error(),
		})
		return
	}

	// Parse AutoEnhance's order_id as UUID (it should be a UUID string)
	orderID, err := uuid.Parse(autoenhanceOrder.OrderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "invalid order id from AutoEnhance",
			Message: fmt.Sprintf("AutoEnhance returned invalid UUID: %s", autoenhanceOrder.OrderID),
		})
		return
	}

	// Create order in database using AutoEnhance's generated order_id as our primary key
	order, err := h.dbClient.CreateOrder(orderID, userID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to create order",
			Message: err.Error(),
		})
		return
	}

	// Sync AutoEnhance data (name, status, etc.) to database
	if h.autoenhanceClient != nil && autoenhanceOrder != nil {
		var lastUpdated *time.Time
		if !autoenhanceOrder.LastUpdatedAt.Time.IsZero() {
			lastUpdated = &autoenhanceOrder.LastUpdatedAt.Time
		}
		_ = h.dbClient.SyncAutoEnhanceOrderData(
			orderID,
			autoenhanceOrder.Name,
			autoenhanceOrder.Status,
			autoenhanceOrder.IsProcessing,
			autoenhanceOrder.IsMerging,
			autoenhanceOrder.IsDeleted,
			int(autoenhanceOrder.TotalImages),
			lastUpdated,
		)
		// Refresh order to get synced data
		order, _ = h.dbClient.GetOrder(orderID, userID)
	}

	var metadata map[string]interface{}
	if len(order.Metadata) > 0 {
		json.Unmarshal(order.Metadata, &metadata)
	}

	response := models.OrderResponse{
		ID:        order.ID.String(),
		Status:    order.Status,
		Progress:  order.Progress,
		Metadata:  metadata,
		CreatedAt: order.CreatedAt,
		UpdatedAt: order.UpdatedAt,
	}

	// Include cached AutoEnhance data
	if order.Name.Valid {
		response.Name = order.Name.String
	}
	if order.AutoEnhanceStatus.Valid {
		response.AutoEnhanceStatus = order.AutoEnhanceStatus.String
	}
	response.IsProcessing = order.IsProcessing
	response.IsMerging = order.IsMerging
	response.IsDeleted = order.IsDeleted
	response.TotalImages = order.TotalImages
	if order.AutoEnhanceLastUpdatedAt.Valid {
		response.AutoEnhanceLastUpdatedAt = &order.AutoEnhanceLastUpdatedAt.Time
	}

	c.JSON(http.StatusOK, response)
}

// ListOrders godoc
// @Summary     List all orders
// @Description Returns a list of all orders for the authenticated user
// @Tags        orders
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Success     200 {object} models.OrderListResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders [get]
func (h *OrdersHandler) ListOrders(c *gin.Context) {
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

	orders, err := h.dbClient.ListOrders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to list orders",
			Message: err.Error(),
		})
		return
	}

	summaries := make([]models.OrderSummary, len(orders))
	for i, o := range orders {
		summary := models.OrderSummary{
			ID:        o.ID.String(),
			Status:    o.Status,
			Progress:  o.Progress,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		}

		// Use cached name from database if available and not empty
		if o.Name.Valid && o.Name.String != "" {
			summary.Name = o.Name.String
		} else if h.autoenhanceClient != nil {
			// If name not cached or is empty, fetch from AutoEnhance and sync to DB
			autoenhanceOrder, err := h.autoenhanceClient.GetOrder(o.ID.String())
			if err == nil && autoenhanceOrder != nil && autoenhanceOrder.Name != "" {
				summary.Name = autoenhanceOrder.Name
				// Sync to database for future requests
				var lastUpdated *time.Time
				if !autoenhanceOrder.LastUpdatedAt.Time.IsZero() {
					lastUpdated = &autoenhanceOrder.LastUpdatedAt.Time
				}
				go func(orderID uuid.UUID) {
					_ = h.dbClient.SyncAutoEnhanceOrderData(
						orderID,
						autoenhanceOrder.Name,
						autoenhanceOrder.Status,
						autoenhanceOrder.IsProcessing,
						autoenhanceOrder.IsMerging,
						autoenhanceOrder.IsDeleted,
						int(autoenhanceOrder.TotalImages),
						lastUpdated,
					)
				}(o.ID)
			}
		}

		summaries[i] = summary
	}

	c.JSON(http.StatusOK, models.OrderListResponse{Orders: summaries})
}

// GetOrder godoc
// @Summary     Get order details
// @Description Returns detailed information about a specific order
// @Tags        orders
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Success     200 {object} models.OrderResponse
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /orders/{order_id} [get]
func (h *OrdersHandler) GetOrder(c *gin.Context) {
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

	var metadata map[string]interface{}
	if len(order.Metadata) > 0 {
		json.Unmarshal(order.Metadata, &metadata)
	}

	response := models.OrderResponse{
		ID:        order.ID.String(),
		Status:    order.Status,
		Progress:  order.Progress,
		Metadata:  metadata,
		CreatedAt: order.CreatedAt,
		UpdatedAt: order.UpdatedAt,
	}

	if order.ErrorMessage.Valid {
		response.ErrorMessage = order.ErrorMessage.String
	}

	// Use cached AutoEnhance data from database (fast!)
	if order.Name.Valid {
		response.Name = order.Name.String
	}
	if order.AutoEnhanceStatus.Valid {
		response.AutoEnhanceStatus = order.AutoEnhanceStatus.String
	}
	response.IsProcessing = order.IsProcessing
	response.IsMerging = order.IsMerging
	response.IsDeleted = order.IsDeleted
	response.TotalImages = order.TotalImages
	if order.AutoEnhanceLastUpdatedAt.Valid {
		response.AutoEnhanceLastUpdatedAt = &order.AutoEnhanceLastUpdatedAt.Time
	}

	// Optionally refresh from AutoEnhance in background (for real-time data like images)
	// But return cached data immediately for fast response
	if h.autoenhanceClient != nil {
		// Fetch fresh data in background and sync to DB
		go func() {
			autoenhanceOrder, err := h.autoenhanceClient.GetOrder(order.ID.String())
			if err == nil {
				var lastUpdated *time.Time
				if !autoenhanceOrder.LastUpdatedAt.Time.IsZero() {
					lastUpdated = &autoenhanceOrder.LastUpdatedAt.Time
				}
				_ = h.dbClient.SyncAutoEnhanceOrderData(
					order.ID,
					autoenhanceOrder.Name,
					autoenhanceOrder.Status,
					autoenhanceOrder.IsProcessing,
					autoenhanceOrder.IsMerging,
					autoenhanceOrder.IsDeleted,
					int(autoenhanceOrder.TotalImages),
					lastUpdated,
				)
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
		}()

		// For images, we still need to fetch from AutoEnhance (not cached)
		autoenhanceOrder, err := h.autoenhanceClient.GetOrder(order.ID.String())
		if err == nil {
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

		// Get brackets info synchronously for response
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

// VerifyOrderUploads godoc
// @Summary     Verify order has uploaded images in AutoEnhance
// @Description Checks with AutoEnhance API to verify that an order has uploaded brackets/images
// @Tags        orders
// @Accept     json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Success     200 {object} map[string]interface{} "verification result"
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id}/verify [get]
func (h *OrdersHandler) VerifyOrderUploads(c *gin.Context) {
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

	// Get order details from AutoEnhance
	autoenhanceOrder, err := h.autoenhanceClient.GetOrder(order.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get order from AutoEnhance",
			Message: err.Error(),
		})
		return
	}

	// Get brackets from AutoEnhance
	brackets, err := h.autoenhanceClient.GetOrderBrackets(order.ID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to get brackets from AutoEnhance",
			Message: err.Error(),
		})
		return
	}

	// Count uploaded brackets
	uploadedCount := 0
	for _, bracket := range brackets.Brackets {
		if bracket.IsUploaded {
			uploadedCount++
		}
	}

	// Build verification response
	response := map[string]interface{}{
		"order_id":           orderID.String(),
		"autoenhance_order_id": autoenhanceOrder.OrderID,
		"order_status":       autoenhanceOrder.Status,
		"total_brackets":     len(brackets.Brackets),
		"uploaded_brackets":  uploadedCount,
		"total_images":       autoenhanceOrder.TotalImages,
		"has_uploaded_images": uploadedCount > 0 || autoenhanceOrder.TotalImages > 0,
		"brackets":           brackets.Brackets,
		"images":             autoenhanceOrder.Images,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteOrder godoc
// @Summary     Delete an order
// @Description Deletes an order, including associated AutoEnhance AI order and files from Supabase Storage
// @Tags        orders
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Success     200 {object} map[string]string "message"
// @Failure     400 {object} models.ErrorResponse
// @Failure     401 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Failure     500 {object} models.ErrorResponse
// @Router      /orders/{order_id} [delete]
func (h *OrdersHandler) DeleteOrder(c *gin.Context) {
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

	// Verify order exists
	_, err = h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	// Delete from AutoEnhance with retry - use the same order_id
	err = h.autoenhanceClient.RetryWithBackoff(func() error {
		return h.autoenhanceClient.DeleteOrder(orderID.String())
	}, 3)
	if err != nil {
		// Log error but continue with database deletion
	}

	// Delete files from storage
	if err := h.storageClient.DeleteOrderFiles(userID, orderID); err != nil {
		// Log error but continue
	}

	// Delete from database (cascade will delete files)
	if err := h.dbClient.DeleteOrder(orderID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to delete order",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "order deleted successfully"})
}

