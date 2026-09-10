package adaptergateway

import (
	"encoding/json"
	"fmt"
	"time"

	v3 "github.com/e-spl/e-sp-line2/internal/protocol/v3"
	"github.com/e-spl/e-sp-line2/pkg/logger"
)

// DefaultToolCallTimeout is how long a tool call waits for a bridge reply
// before failing with ToolErrTimeout.
const DefaultToolCallTimeout = 15 * time.Second

// pendingToolCall tracks an in-flight tool invocation awaiting its result from
// the bridge.
type pendingToolCall struct {
	req    *v3.ToolCallRequest
	result chan *v3.ToolCallResponse
}

// ToolRegistry returns the gateway's tool registry (the adapter's declared
// "inactive field" tools).
func (g *Gateway) ToolRegistry() *v3.ToolRegistry {
	return g.tools
}

// RegisterTools adds tool definitions to the gateway registry. It is used by
// adapters (or the built-in catalog) to declare callable inactive fields.
func (g *Gateway) RegisterTools(defs []v3.ToolDefinition) {
	if g.tools == nil {
		return
	}
	g.tools.RegisterAll(defs)
}

// RegisterToolsFromCatalogFrame parses a "tool_catalog" frame and registers the
// declared tools with the gateway. It accepts the tool list either at the top
// level ("tools") or nested inside "payload" ("tools"). Both bridges
// (/ws/adapter) and external frameworks in server mode (/ws/adapter-gateway)
// may send this frame to declare the inactive-field tools they can serve.
//
// Accepted tool shape (JSON):
//
//	{
//	  "name": "get_product",
//	  "semantic_id": "get_product",
//	  "category": "query_only",
//	  "description": "...",
//	  "parameters": [{"name":"item_id","type":"string","required":true}],
//	  "enabled": true,
//	  "require_confirm": false,
//	  "returns": "...",
//	  "platform": "xianyu"
//	}
func (g *Gateway) RegisterToolsFromCatalogFrame(frame map[string]interface{}) int {
	if g == nil || frame == nil {
		return 0
	}
	src := frame
	if p, ok := frame["payload"].(map[string]interface{}); ok {
		if _, ok := p["tools"]; ok {
			src = p
		}
	}
	rawTools, ok := src["tools"].([]interface{})
	if !ok {
		return 0
	}

	defs := make([]v3.ToolDefinition, 0, len(rawTools))
	for _, rt := range rawTools {
		m, ok := rt.(map[string]interface{})
		if !ok {
			continue
		}
		def := v3.ToolDefinition{}
		def.Name, _ = m["name"].(string)
		if def.Name == "" {
			continue
		}
		if s, ok := m["semantic_id"].(string); ok {
			def.SemanticID = v3.SemanticID(s)
		}
		if cat, ok := m["category"].(string); ok && cat != "" {
			def.Category = v3.ToolCategory(cat)
		}
		def.Description, _ = m["description"].(string)
		def.Returns, _ = m["returns"].(string)
		def.Platform, _ = m["platform"].(string)
		def.Enabled = true
		if v, ok := m["enabled"].(bool); ok {
			def.Enabled = v
		}
		if v, ok := m["require_confirm"].(bool); ok {
			def.RequireConfirm = v
		}
		if params, ok := m["parameters"].([]interface{}); ok {
			for _, rp := range params {
				pm, ok := rp.(map[string]interface{})
				if !ok {
					continue
				}
				param := v3.ToolParameter{}
				param.Name, _ = pm["name"].(string)
				param.Description, _ = pm["description"].(string)
				if t, ok := pm["type"].(string); ok && t != "" {
					param.Type = v3.ToolParameterType(t)
				}
				if v, ok := pm["required"].(bool); ok {
					param.Required = v
				}
				if v, ok := pm["default"]; ok {
					param.Default = v
				}
				if enums, ok := pm["enum"].([]interface{}); ok {
					for _, e := range enums {
						if s, ok := e.(string); ok {
							param.Enum = append(param.Enum, s)
						}
					}
				}
				def.Parameters = append(def.Parameters, param)
			}
		}
		defs = append(defs, def)
	}
	if len(defs) == 0 {
		return 0
	}
	g.RegisterTools(defs)
	logger.Info("Adapter gateway registered tools from catalog frame",
		logger.Int("count", len(defs)))
	return len(defs)
}

// ListTools returns all registered tool definitions.
func (g *Gateway) ListTools() []v3.ToolDefinition {
	if g.tools == nil {
		return nil
	}
	return g.tools.List()
}

// ListEnabledTools returns only enabled tool definitions.
func (g *Gateway) ListEnabledTools() []v3.ToolDefinition {
	if g.tools == nil {
		return nil
	}
	return g.tools.ListEnabled()
}

// SetToolEnabled toggles a tool's enabled flag (layer 1 of the permission
// model). It reports whether the tool exists.
func (g *Gateway) SetToolEnabled(name string, enabled bool) bool {
	if g.tools == nil {
		return false
	}
	return g.tools.SetEnabled(name, enabled)
}

// resolveTool resolves a request to a tool definition, preferring the explicit
// tool name and falling back to the semantic ID.
func (g *Gateway) resolveTool(req *v3.ToolCallRequest) (*v3.ToolDefinition, string) {
	if g.tools == nil {
		return nil, v3.ToolErrNotFound
	}
	if req.Tool != "" {
		def, ok := g.tools.Get(req.Tool)
		if !ok {
			return nil, v3.ToolErrNotFound
		}
		return def, ""
	}
	if req.SemanticID != "" {
		def, ok := g.tools.GetBySemanticID(req.SemanticID)
		if !ok {
			return nil, v3.ToolErrNotFound
		}
		return def, ""
	}
	return nil, v3.ToolErrNotFound
}

// ToolRequiresWrite reports whether the resolved tool performs a state change
// (query_action), which requires the adapter to hold write permission.
func (g *Gateway) ToolRequiresWrite(req *v3.ToolCallRequest) bool {
	def, _ := g.resolveTool(req)
	return def != nil && def.Category == v3.ToolCategoryQueryAction
}

// validateArguments checks that all required parameters are present.
func validateArguments(def *v3.ToolDefinition, args map[string]interface{}) bool {
	for _, p := range def.Parameters {
		if !p.Required {
			continue
		}
		if args == nil {
			return false
		}
		v, ok := args[p.Name]
		if !ok || v == nil || v == "" {
			return false
		}
	}
	return true
}

// CallTool invokes an inactive-field tool against the bridge that owns the
// request's instance_id and blocks until the bridge replies or the timeout
// elapses.
//
// It enforces the permission model:
//   - the tool must exist and be enabled (40401 / 40408)
//   - query_action tools (and tools flagged RequireConfirm) need Confirm=true
//     (40407)
//   - required arguments must be present (40404)
//   - a live bridge must exist for the instance (40402)
func (g *Gateway) CallTool(req *v3.ToolCallRequest) *v3.ToolCallResponse {
	start := time.Now()

	if req == nil {
		return &v3.ToolCallResponse{
			Success:   false,
			Error:     v3.ToolErrorMessage(v3.ToolErrInvalidArgs),
			ErrorCode: v3.ToolErrInvalidArgs,
		}
	}
	if req.CallID == "" {
		req.CallID = v3.GenerateEventID()
	}

	fail := func(code string, detail string) *v3.ToolCallResponse {
		msg := v3.ToolErrorMessage(code)
		if detail != "" {
			msg = msg + ": " + detail
		}
		return &v3.ToolCallResponse{
			CallID:     req.CallID,
			Tool:       req.Tool,
			SemanticID: req.SemanticID,
			Success:    false,
			Error:      msg,
			ErrorCode:  code,
			DurationMS: time.Since(start).Milliseconds(),
		}
	}

	def, errCode := g.resolveTool(req)
	if def == nil {
		return fail(errCode, "")
	}
	if !def.Enabled {
		return fail(v3.ToolErrDisabled, def.Name)
	}
	if (def.Category == v3.ToolCategoryQueryAction || def.RequireConfirm) && !req.Confirm {
		return fail(v3.ToolErrConfirmRequired, def.Name)
	}
	if !validateArguments(def, req.Arguments) {
		return fail(v3.ToolErrInvalidArgs, def.Name)
	}
	if req.InstanceID == "" {
		return fail(v3.ToolErrBridgeOffline, "missing instance_id")
	}

	// Register the pending call BEFORE dispatching so a very fast bridge reply
	// can never be missed.
	key := pendingKey(req.InstanceID, req.CallID)
	pc := &pendingToolCall{
		req:    req,
		result: make(chan *v3.ToolCallResponse, 1),
	}
	g.pendingMu.Lock()
	g.pendingTools[key] = pc
	g.pendingMu.Unlock()

	cleanup := func() {
		g.pendingMu.Lock()
		delete(g.pendingTools, key)
		g.pendingMu.Unlock()
	}

	if !g.routeToolCallToBridge(def, req) {
		cleanup()
		return fail(v3.ToolErrBridgeOffline, req.InstanceID)
	}

	timeout := DefaultToolCallTimeout
	if req.TimeoutMS > 0 {
		timeout = time.Duration(req.TimeoutMS) * time.Millisecond
	}

	select {
	case resp := <-pc.result:
		if resp == nil {
			return fail(v3.ToolErrUpstream, def.Name)
		}
		if resp.CallID == "" {
			resp.CallID = req.CallID
		}
		if resp.Tool == "" {
			resp.Tool = def.Name
		}
		if resp.SemanticID == "" {
			resp.SemanticID = def.SemanticID
		}
		resp.DurationMS = time.Since(start).Milliseconds()
		return resp
	case <-time.After(timeout):
		cleanup()
		return fail(v3.ToolErrTimeout, def.Name)
	}
}

// routeToolCallToBridge delivers a tool_call frame to the bridge owning the
// instance. It reports whether the frame was buffered for delivery.
func (g *Gateway) routeToolCallToBridge(def *v3.ToolDefinition, req *v3.ToolCallRequest) bool {
	g.bridgeMu.RLock()
	bc, ok := g.bridgeConns[req.InstanceID]
	g.bridgeMu.RUnlock()
	if !ok {
		logger.Warn("Tool call: no bridge connection for instance",
			logger.String("instance_id", req.InstanceID),
			logger.String("tool", def.Name))
		return false
	}

	payload := map[string]interface{}{
		"call_id":         req.CallID,
		"tool":            def.Name,
		"semantic_id":     def.SemanticID,
		"category":        def.Category,
		"instance_id":     req.InstanceID,
		"conversation_id": req.ConversationID,
		"arguments":       req.Arguments,
		"trace_id":        req.TraceID,
	}

	frame := map[string]interface{}{
		"type":         "tool_call",
		"command_type": v3.CommandTypeToolCall,
		"id":           req.CallID,
		"call_id":      req.CallID,
		"instance_id":  req.InstanceID,
		"timestamp":    time.Now().UnixMilli(),
		"payload":      payload,
	}

	data, err := json.Marshal(frame)
	if err != nil {
		logger.Error("Tool call marshal failed", logger.String("error", err.Error()))
		return false
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Warn("Tool call bridge send panicked",
				logger.String("instance_id", req.InstanceID),
				logger.String("error", fmt.Sprint(r)))
		}
	}()

	select {
	case bc.send <- data:
		return true
	default:
		logger.Warn("Tool call bridge send buffer full",
			logger.String("instance_id", req.InstanceID))
		return false
	}
}

// HandleToolResult delivers a bridge tool result to the waiting CallTool
// invocation. It is called by the bridge WebSocket handler (/ws/adapter) when
// it receives a frame of type "tool_result".
//
// It reports whether a matching pending call was found.
func (g *Gateway) HandleToolResult(instanceID string, payload map[string]interface{}) bool {
	if payload == nil {
		return false
	}

	callID, _ := payload["call_id"].(string)
	if callID == "" {
		callID, _ = payload["id"].(string)
	}
	if callID == "" {
		return false
	}

	// The bridge may report the instance under either key.
	if instanceID == "" {
		if v, ok := payload["instance_id"].(string); ok {
			instanceID = v
		}
	}

	g.pendingMu.Lock()
	pc, ok := g.pendingTools[pendingKey(instanceID, callID)]
	g.pendingMu.Unlock()
	if !ok {
		logger.Debug("Tool result for unknown call",
			logger.String("call_id", callID),
			logger.String("instance_id", instanceID))
		return false
	}

	resp := &v3.ToolCallResponse{
		CallID: callID,
	}
	if v, ok := payload["success"].(bool); ok {
		resp.Success = v
	}
	if v, ok := payload["tool"].(string); ok {
		resp.Tool = v
	}
	if v, ok := payload["semantic_id"].(string); ok {
		resp.SemanticID = v3.SemanticID(v)
	}
	if v, ok := payload["error"].(string); ok {
		resp.Error = v
	}
	if v, ok := payload["error_code"].(string); ok {
		resp.ErrorCode = v
	}
	resp.Data = payload["data"]

	select {
	case pc.result <- resp:
		return true
	default:
		return false
	}
}

// pendingKey builds the lookup key for an in-flight tool call.
func pendingKey(instanceID, callID string) string {
	return instanceID + "|" + callID
}

// buildToolResultFrame converts a ToolCallResponse into a wire frame addressed
// back to the requesting adapter gateway client.
func buildToolResultFrame(resp *v3.ToolCallResponse) map[string]interface{} {
	return map[string]interface{}{
		"type":      "tool_result",
		"id":        resp.CallID,
		"timestamp": time.Now().UnixMilli(),
		"payload": map[string]interface{}{
			"call_id":     resp.CallID,
			"tool":        resp.Tool,
			"semantic_id": resp.SemanticID,
			"success":     resp.Success,
			"data":        resp.Data,
			"error":       resp.Error,
			"error_code":  resp.ErrorCode,
			"duration_ms": resp.DurationMS,
		},
	}
}

// buildToolCallRequestFromFrame extracts a ToolCallRequest from an inbound
// frame sent by a downstream framework (LangBot). The frame may carry the
// request fields either at the top level or inside "payload".
func buildToolCallRequestFromFrame(frame map[string]interface{}) *v3.ToolCallRequest {
	src := frame
	if p, ok := frame["payload"].(map[string]interface{}); ok {
		src = p
	}

	req := &v3.ToolCallRequest{}
	req.CallID, _ = src["call_id"].(string)
	if req.CallID == "" {
		req.CallID, _ = frame["id"].(string)
	}
	req.Tool, _ = src["tool"].(string)
	if v, ok := src["semantic_id"].(string); ok {
		req.SemanticID = v3.SemanticID(v)
	}
	req.InstanceID, _ = src["instance_id"].(string)
	if req.InstanceID == "" {
		req.InstanceID, _ = frame["instance_id"].(string)
	}
	req.ConversationID, _ = src["conversation_id"].(string)
	req.TraceID, _ = src["trace_id"].(string)
	if v, ok := src["arguments"].(map[string]interface{}); ok {
		req.Arguments = v
	}
	if v, ok := src["timeout_ms"].(float64); ok {
		req.TimeoutMS = int64(v)
	}
	if v, ok := src["confirm"].(bool); ok {
		req.Confirm = v
	}
	return req
}

// BroadcastEvent fans out an ESPL v3 envelope (e.g. an e-commerce lifecycle or
// activity event) to all matching adapter gateway clients. It reuses the
// inbound broadcast path so server-mode and client-mode adapters both receive
// the event.
func (g *Gateway) BroadcastEvent(envelope map[string]interface{}) {
	g.BroadcastInbound(envelope)
}
