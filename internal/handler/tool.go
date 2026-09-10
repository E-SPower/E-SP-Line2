package handler

import (
	"net/http"

	"github.com/e-spl/e-sp-line2/internal/adaptergateway"
	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/gin-gonic/gin"
)

// ListTools returns the adapter's registered "inactive field" tool catalog.
//
//	GET /api/v1/tools?enabled=true
func ListTools(gateway *adaptergateway.Gateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		if gateway == nil {
			c.JSON(http.StatusOK, gin.H{"data": []v3.ToolDefinition{}, "total": 0})
			return
		}
		onlyEnabled := c.Query("enabled") == "true"
		var tools []v3.ToolDefinition
		if onlyEnabled {
			tools = gateway.ListEnabledTools()
		} else {
			tools = gateway.ListTools()
		}
		if tools == nil {
			tools = []v3.ToolDefinition{}
		}
		c.JSON(http.StatusOK, gin.H{"data": tools, "total": len(tools)})
	}
}

// SetToolEnabled toggles a tool's enabled flag (permission layer 1).
//
//	PUT /api/v1/tools/:name/enabled  {"enabled": true|false}
func SetToolEnabled(gateway *adaptergateway.Gateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		if gateway == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gateway unavailable"})
			return
		}
		name := c.Param("name")
		var body struct {
			Enabled *bool `json:"enabled"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Enabled == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "enabled is required"})
			return
		}
		if !gateway.SetToolEnabled(name, *body.Enabled) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tool not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"name": name, "enabled": *body.Enabled}})
	}
}

// CallTool invokes an inactive-field tool synchronously and returns its result.
//
//	POST /api/v1/tools/call
//	{
//	  "tool": "get_logistics",
//	  "instance_id": "xianyu-1",
//	  "arguments": {"order_id": "123"},
//	  "confirm": false
//	}
//
// The tool may also be selected by semantic_id instead of tool.
func CallTool(gateway *adaptergateway.Gateway) gin.HandlerFunc {
	return func(c *gin.Context) {
		if gateway == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "gateway unavailable"})
			return
		}

		var req v3.ToolCallRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
		if req.Tool == "" && req.SemanticID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "tool or semantic_id is required"})
			return
		}
		if req.InstanceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "instance_id is required"})
			return
		}

		resp := gateway.CallTool(&req)
		// Tool-level failures are reported in the body with HTTP 200 so callers
		// can distinguish transport errors from tool errors; the response
		// always carries success/error_code.
		c.JSON(http.StatusOK, gin.H{"data": resp})
	}
}
