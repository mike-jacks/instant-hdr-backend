package handlers

import (
	"encoding/json"
	"fmt"
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
// @Description Initiates HDR processing and merging of uploaded images using AutoEnhance AI.
// @Description
// @Description **Processing Options:**
// @Description
// @Description **enhance_type** (default: "property"):
// @Description - "property": Best for real estate - balanced enhancement
// @Description - "warm": Warm color grading for cozy feel (AI >= 4.0)
// @Description - "neutral": Neutral natural look (AI >= 4.0)
// @Description - "modern": Contemporary enhancement (AI >= 4.0)
// @Description
// @Description **sky_replacement** (default: true):
// @Description - Replaces dull skies with attractive blue skies
// @Description
// @Description **cloud_type** (optional):
// @Description - "CLEAR": Clear blue sky, "LOW_CLOUD": Subtle clouds, "HIGH_CLOUD": Dramatic clouds
// @Description
// @Description **window_pull_type** (default: "WINDOWS_WITH_SKIES"):
// @Description - "NONE": No window enhancement
// @Description - "ONLY_WINDOWS": Enhance window views only
// @Description - "WINDOWS_WITH_SKIES": Enhance windows + replace exterior skies (AI >= 5.2) - BEST RESULTS
// @Description
// @Description **vertical_correction** (default: true):
// @Description - Straightens tilted walls and vertical lines
// @Description
// @Description **lens_correction** (default: true):
// @Description - Removes wide-angle lens distortion
// @Description
// @Description **upscale** (default: false):
// @Description - AI upscaling to double resolution (increases processing time)
// @Description
// @Description **privacy** (default: false):
// @Description - Blurs faces and license plates
// @Description
// @Description **bracket_grouping** (default: "by_upload_group"):
// @Description - "by_upload_group": Use groups from upload
// @Description - "auto": Sequential sets (every N brackets = 1 HDR, where N = brackets_per_image)
// @Description - "all": One mega-HDR from all brackets
// @Description - "individual": Separate images (no HDR)
// @Description
// @Description **brackets_per_image** (default: 3, only for "auto" mode):
// @Description - How many consecutive brackets to merge into one HDR image
// @Description - Example: 6 brackets + brackets_per_image=3 → 2 HDR images ([1,2,3] and [4,5,6])
// @Description - 3 = Standard HDR, 5 = High dynamic range, 7 = Extreme lighting
// @Tags        process
// @Accept      json
// @Produce     json
// @Security    Bearer
// @Param       order_id path string true "Order ID (UUID)"
// @Param       request body models.ProcessRequest false "Processing options with defaults shown in model"
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

	// Organize brackets into image groups based on BracketGrouping strategy
	imageGroups := organizeBracketsIntoGroups(brackets, req.BracketGrouping, req.BracketsPerImage)

	if len(imageGroups) == 0 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "failed to organize brackets",
			Message: "no valid bracket groups created",
		})
		return
	}

	// Set default enhance_type if not provided
	enhanceType := req.EnhanceType
	if enhanceType == "" {
		enhanceType = "property" // Default for real estate
	}

	// Build process request with organized image groups
	processReq := autoenhance.OrderHDRProcessIn{
		EnhanceType: enhanceType,
		Images:      imageGroups,
	}

	// Set optional fields with defaults for real estate photography

	// Sky Replacement (default: true)
	if req.SkyReplacement != nil {
		processReq.SkyReplacement = req.SkyReplacement
	} else {
		skyReplacement := true // Default for real estate
		processReq.SkyReplacement = &skyReplacement
	}

	// Cloud Type (optional, AutoEnhance chooses if not specified)
	if req.CloudType != "" {
		processReq.CloudType = &req.CloudType
	}

	// Vertical Correction (default: true)
	if req.VerticalCorrection != nil {
		processReq.VerticalCorrection = req.VerticalCorrection
	} else {
		verticalCorrection := true // Default
		processReq.VerticalCorrection = &verticalCorrection
	}

	// Lens Correction (default: true)
	if req.LensCorrection != nil {
		processReq.LensCorrection = req.LensCorrection
	} else {
		lensCorrection := true // Default
		processReq.LensCorrection = &lensCorrection
	}

	// Window Pull Type (default: WINDOWS_WITH_SKIES)
	if req.WindowPullType != "" {
		processReq.WindowPullType = &req.WindowPullType
	} else {
		windowPullType := "WINDOWS_WITH_SKIES" // Default for best results (AI >= 5.2)
		processReq.WindowPullType = &windowPullType
	}

	// Upscale (default: false)
	if req.Upscale != nil {
		processReq.Upscale = req.Upscale
	}

	// Privacy (default: false)
	if req.Privacy != nil {
		processReq.Privacy = req.Privacy
	}

	// AI Version (optional, uses latest if not specified)
	if req.AIVersion != "" {
		processReq.AIVersion = req.AIVersion
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

	// Calculate total brackets from all image groups
	totalBrackets := 0
	for _, group := range imageGroups {
		totalBrackets += len(group.BracketIDs)
	}

	// Build processing params for response (show all settings used)
	processingParams := map[string]interface{}{
		"enhance_type":        processReq.EnhanceType,
		"sky_replacement":     processReq.SkyReplacement,
		"vertical_correction": processReq.VerticalCorrection,
		"lens_correction":     processReq.LensCorrection,
		"window_pull_type":    processReq.WindowPullType,
		"upscale":             processReq.Upscale,
		"privacy":             processReq.Privacy,
		"total_brackets":      totalBrackets,
		"total_images":        len(imageGroups),
		"bracket_grouping":    req.BracketGrouping,
	}

	// Add optional parameters if they were specified
	if processReq.CloudType != nil {
		processingParams["cloud_type"] = *processReq.CloudType
	}
	if processReq.AIVersion != "" {
		processingParams["ai_version"] = processReq.AIVersion
	}
	if req.BracketsPerImage > 0 {
		processingParams["brackets_per_image"] = req.BracketsPerImage
	}

	response := models.ProcessResponse{
		OrderID:          orderID.String(),
		Status:           "processing",
		Message:          fmt.Sprintf("Order processing started successfully - Creating %d HDR image(s) from %d bracket(s)", len(imageGroups), totalBrackets),
		ProcessingParams: processingParams,
	}

	c.JSON(http.StatusOK, response)
}

// organizeBracketsIntoGroups organizes brackets into image groups for HDR processing
// Supports multiple strategies: "by_upload_group", "auto", "all", "individual", or custom groups
func organizeBracketsIntoGroups(brackets []models.Bracket, grouping interface{}, bracketsPerImage int) []autoenhance.OrderImageIn {
	// Default: use upload groups if available, otherwise auto
	if grouping == nil {
		grouping = "by_upload_group"
	}
	
	// Default brackets per image
	if bracketsPerImage == 0 {
		bracketsPerImage = 3
	}

	var imageGroups []autoenhance.OrderImageIn
	
	// Handle string strategies
	if groupingStr, ok := grouping.(string); ok {
		switch groupingStr {
		case "by_upload_group":
			// Group brackets by the group_id in their metadata
			groupMap := make(map[string][]string) // group_id -> []bracket_id
			ungrouped := []string{} // brackets without a group_id
			
			for _, bracket := range brackets {
				// Try to extract group_id from metadata
				var groupID string
				if len(bracket.Metadata) > 0 {
					var metadata map[string]interface{}
					if err := json.Unmarshal(bracket.Metadata, &metadata); err == nil {
						if gid, ok := metadata["group_id"].(string); ok && gid != "" {
							groupID = gid
						}
					}
				}
				
				if groupID != "" {
					groupMap[groupID] = append(groupMap[groupID], bracket.BracketID)
				} else {
					ungrouped = append(ungrouped, bracket.BracketID)
				}
			}
			
			// Create image groups from grouped brackets
			for _, bracketIDs := range groupMap {
				if len(bracketIDs) > 0 {
					imageGroups = append(imageGroups, autoenhance.OrderImageIn{
						BracketIDs: bracketIDs,
					})
				}
			}
			
			// If there are ungrouped brackets, fall back to auto-grouping for them
			if len(ungrouped) > 0 {
				for i := 0; i < len(ungrouped); i += bracketsPerImage {
					end := i + bracketsPerImage
					if end > len(ungrouped) {
						end = len(ungrouped)
					}
					imageGroups = append(imageGroups, autoenhance.OrderImageIn{
						BracketIDs: ungrouped[i:end],
					})
				}
			}
			
			// If no groups were created, fall back to auto
			if len(imageGroups) == 0 {
				return organizeBracketsIntoGroups(brackets, "auto", bracketsPerImage)
			}

		case "all":
			// All brackets into ONE HDR image
			bracketIDs := make([]string, len(brackets))
			for i, bracket := range brackets {
				bracketIDs[i] = bracket.BracketID
			}
			imageGroups = []autoenhance.OrderImageIn{
				{BracketIDs: bracketIDs},
			}

		case "individual":
			// Each bracket becomes its own image (no HDR merge)
			for _, bracket := range brackets {
				imageGroups = append(imageGroups, autoenhance.OrderImageIn{
					BracketIDs: []string{bracket.BracketID},
				})
			}

		case "auto":
			fallthrough
		default:
			// Auto-group by sequential sets
			// E.g., 6 brackets with bracketsPerImage=3 → 2 groups of 3
			for i := 0; i < len(brackets); i += bracketsPerImage {
				end := i + bracketsPerImage
				if end > len(brackets) {
					end = len(brackets)
				}
				
				bracketIDs := make([]string, end-i)
				for j := i; j < end; j++ {
					bracketIDs[j-i] = brackets[j].BracketID
				}
				
				imageGroups = append(imageGroups, autoenhance.OrderImageIn{
					BracketIDs: bracketIDs,
				})
			}
		}
		return imageGroups
	}

	// Handle custom array grouping: [[id1,id2,id3],[id4,id5,id6]]
	// First, try to parse as JSON array
	jsonBytes, err := json.Marshal(grouping)
	if err != nil {
		// Fallback to auto mode
		return organizeBracketsIntoGroups(brackets, "auto", bracketsPerImage)
	}

	var customGroups [][]string
	if err := json.Unmarshal(jsonBytes, &customGroups); err != nil {
		// Fallback to auto mode
		return organizeBracketsIntoGroups(brackets, "auto", bracketsPerImage)
	}

	// Create a map of bracket IDs for validation
	bracketMap := make(map[string]bool)
	for _, bracket := range brackets {
		bracketMap[bracket.BracketID] = true
	}

	// Validate and build image groups
	for _, group := range customGroups {
		// Validate all bracket IDs exist
		validGroup := true
		for _, bracketID := range group {
			if !bracketMap[bracketID] {
				validGroup = false
				break
			}
		}

		if validGroup && len(group) > 0 {
			imageGroups = append(imageGroups, autoenhance.OrderImageIn{
				BracketIDs: group,
			})
		}
	}

	// If no valid groups, fallback to auto
	if len(imageGroups) == 0 {
		return organizeBracketsIntoGroups(brackets, "auto", bracketsPerImage)
	}

	return imageGroups
}
