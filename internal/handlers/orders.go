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
// @Description Creates a new AutoEnhance AI order for a listing (real estate shoot). Returns the order ID and AutoEnhance order ID.
// @Tags        orders
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       request body models.CreateOrderRequest false "Order metadata (optional)"
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
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no JSON body, use empty metadata
		req.Metadata = make(map[string]interface{})
	}

	// Generate order ID
	orderID := uuid.New().String()

	// Create AutoEnhance order with retry
	var autoenhanceOrder *autoenhance.OrderOut
	err = h.autoenhanceClient.RetryWithBackoff(func() error {
		var err error
		autoenhanceOrder, err = h.autoenhanceClient.CreateOrder(orderID, "")
		return err
	}, 3)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to create autoenhance order",
			Message: err.Error(),
		})
		return
	}

	// Create order in database
	order, err := h.dbClient.CreateOrder(userID, autoenhanceOrder.OrderID, req.Metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "failed to create order",
			Message: err.Error(),
		})
		return
	}

	var metadata map[string]interface{}
	if len(order.Metadata) > 0 {
		json.Unmarshal(order.Metadata, &metadata)
	}

	c.JSON(http.StatusOK, models.OrderResponse{
		ID:                 order.ID.String(),
		AutoEnhanceOrderID: order.AutoEnhanceOrderID,
		Status:             order.Status,
		Progress:           order.Progress,
		Metadata:           metadata,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	})
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
		summaries[i] = models.OrderSummary{
			ID:        o.ID.String(),
			Status:    o.Status,
			Progress:  o.Progress,
			CreatedAt: o.CreatedAt,
			UpdatedAt: o.UpdatedAt,
		}
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
		ID:                 order.ID.String(),
		AutoEnhanceOrderID: order.AutoEnhanceOrderID,
		Status:             order.Status,
		Progress:          order.Progress,
		Metadata:           metadata,
		CreatedAt:          order.CreatedAt,
		UpdatedAt:          order.UpdatedAt,
	}

	if order.ErrorMessage.Valid {
		response.ErrorMessage = order.ErrorMessage.String
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

	// Get order to get autoenhance_order_id
	order, err := h.dbClient.GetOrder(orderID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Error:   "order not found",
			Message: err.Error(),
		})
		return
	}

	// Delete from AutoEnhance with retry
	err = h.autoenhanceClient.RetryWithBackoff(func() error {
		return h.autoenhanceClient.DeleteOrder(order.AutoEnhanceOrderID)
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

