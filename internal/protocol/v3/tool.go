package v3

import (
	"sort"
	"sync"
)

// ToolCategory classifies how a tool behaves, mirroring LangBot's "非活动字段
// 注册机制" (Inactive Field Registration Mechanism) tool taxonomy.
//
//   - query_only: idempotent, side-effect free reads (e.g. 商品名称/简介/图片,
//     订单信息, 物流单号). Safe to auto-invoke without user confirmation.
//   - query_action: performs a state change (e.g. 发货, 更新订单状态, 收藏,
//     点赞, 发布). Requires confirmation before execution.
type ToolCategory string

const (
	ToolCategoryQueryOnly   ToolCategory = "query_only"
	ToolCategoryQueryAction ToolCategory = "query_action"
)

// SemanticID is a protocol-agnostic identifier from LangBot's official semantic
// library. Downstream frameworks map a natural-language intent to one of these
// IDs instead of binding to a vendor-specific tool name, which keeps ESPL
// adapters portable across platforms.
type SemanticID string

// Semantic ID constants (LangBot official semantic library + ESPL2 extensions).
const (
	// Generic social/utility semantics.
	SemanticFavorite        SemanticID = "favorite"
	SemanticLike            SemanticID = "like"
	SemanticPublishArticle  SemanticID = "publish_article"
	SemanticGetUserProfile  SemanticID = "get_user_profile"
	SemanticGetGroupMembers SemanticID = "get_group_members"

	// E-commerce semantics implemented by ESPL2.
	SemanticGetProduct     SemanticID = "get_product"      // 商品名称/简介/图片/规格
	SemanticGetProductList SemanticID = "get_product_list" // 商品列表/搜索
	SemanticGetOrder       SemanticID = "get_order"        // 订单信息
	SemanticGetLogistics   SemanticID = "get_logistics"    // 物流单号/发货轨迹
	SemanticShipOrder      SemanticID = "ship_order"       // 发货（有副作用）
	SemanticUpdateOrder    SemanticID = "update_order"     // 更新订单状态（有副作用）
	SemanticGetActivity    SemanticID = "get_activity"     // 活动/事件信息

	// ── XianYuApis 原版兼容语义（HTTP 接口）───────────────────────────────
	// 发布商品：mtop.idle.pc.idleitem.publish
	SemanticPublishItem SemanticID = "publish_item"
	// 更新商品：mtop.taobao.idle.item.update（含改价）
	SemanticUpdateItem SemanticID = "update_item"
	// 单独改价（商品下架/改价）
	SemanticUpdateItemPrice SemanticID = "update_item_price"
	// 分类推荐：mtop.taobao.idle.kgraph.property.recommend
	SemanticGetCategoryRecommend SemanticID = "get_category_recommend"
	// 默认收货地址：mtop.taobao.idle.local.poi.get
	SemanticGetDefaultAddress SemanticID = "get_default_address"
	// 媒体上传：stream-upload.goofish.com/api/upload.api
	SemanticUploadMedia SemanticID = "upload_media"
	// 刷新登录态：mtop.taobao.idlemessage.pc.loginuser.get
	SemanticRefreshToken SemanticID = "refresh_token"
	// 获取登录 token：mtop.taobao.idlemessage.pc.login.token
	SemanticGetToken SemanticID = "get_token"
	// 扫码登录
	SemanticLoginQRCode SemanticID = "login_qrcode"

	// ── XianYuApis 原版兼容语义（WebSocket lwp 命令）──────────────────────
	// 会话历史：/r/MessageManager/listUserMessages
	SemanticGetConversationHistory SemanticID = "get_conversation_history"
	// 创建/打开会话：/r/SingleChatConversation/create
	SemanticCreateConversation SemanticID = "create_conversation"
	// 发送消息：/r/MessageSend/sendByReceiverScope
	SemanticSendMessage SemanticID = "send_message"

	// ── TaoBaoApis 专用语义 ─────────────────────────────────────────
	// 从商品 URL 解析用户信息：userId / encryptUid
	SemanticGetGoodsInfo SemanticID = "get_goods_info"
)

// ToolParameterType enumerates the JSON-schema-like parameter types a tool
// argument may take. Kept intentionally small so both Go and Python adapters
// can validate arguments without a full JSON-schema dependency.
type ToolParameterType string

const (
	ToolParamString ToolParameterType = "string"
	ToolParamNumber ToolParameterType = "number"
	ToolParamBool   ToolParameterType = "boolean"
	ToolParamObject ToolParameterType = "object"
	ToolParamArray  ToolParameterType = "array"
)

// ToolParameter describes a single argument accepted by a tool.
type ToolParameter struct {
	Name        string            `json:"name"`
	Type        ToolParameterType `json:"type"`
	Description string            `json:"description"`
	Required    bool              `json:"required,omitempty"`
	Default     interface{}       `json:"default,omitempty"`
	Enum        []string          `json:"enum,omitempty"`
}

// ToolDefinition is the declarative description of an "inactive field" tool.
// Adapters register these so downstream frameworks (LangBot) can expose them as
// Agent Tools / Plugin Keywords / ActionFlow Cards.
type ToolDefinition struct {
	// Name is the adapter-local tool name (stable identifier).
	Name string `json:"name"`
	// SemanticID links the tool to LangBot's semantic library so intent
	// resolution works across platforms.
	SemanticID SemanticID `json:"semantic_id"`
	// Category decides whether confirmation is required before invocation.
	Category ToolCategory `json:"category"`
	// Description is a human/LLM readable summary (default locale).
	Description string `json:"description"`
	// DescriptionI18n maps locale -> description (e.g. zh_Hans, en_US).
	DescriptionI18n map[string]string `json:"description_i18n,omitempty"`
	// Parameters declares the accepted arguments.
	Parameters []ToolParameter `json:"parameters,omitempty"`
	// Returns documents the shape of a successful response data payload.
	Returns string `json:"returns,omitempty"`
	// Enabled reports whether the tool is currently callable. It mirrors the
	// first layer of LangBot's 4-layer permission model (tool enable switch).
	Enabled bool `json:"enabled"`
	// RequireConfirm forces a confirmation step even for query tools.
	RequireConfirm bool `json:"require_confirm,omitempty"`
	// Platform optionally restricts the tool to a specific platform.
	Platform string `json:"platform,omitempty"`
}

// ToolRegistry holds the set of tools an adapter exposes. It is safe for
// concurrent use because the gateway may list tools while handling calls.
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolDefinition // key: tool name
}

// NewToolRegistry creates an empty tool registry.
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{tools: make(map[string]*ToolDefinition)}
}

// Register adds or replaces a tool definition. Tools default to enabled unless
// explicitly set otherwise; callers can flip Enabled afterwards.
func (r *ToolRegistry) Register(def ToolDefinition) {
	if def.Name == "" {
		return
	}
	if def.Category == "" {
		def.Category = ToolCategoryQueryOnly
	}
	cp := def
	r.mu.Lock()
	r.tools[def.Name] = &cp
	r.mu.Unlock()
}

// RegisterAll registers a slice of tool definitions.
func (r *ToolRegistry) RegisterAll(defs []ToolDefinition) {
	for _, d := range defs {
		r.Register(d)
	}
}

// Unregister removes a tool by name.
func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	delete(r.tools, name)
	r.mu.Unlock()
}

// Get returns a tool definition by name.
func (r *ToolRegistry) Get(name string) (*ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// GetBySemanticID returns the first enabled tool matching a semantic ID.
func (r *ToolRegistry) GetBySemanticID(id SemanticID) (*ToolDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, t := range r.tools {
		if t.SemanticID == id && t.Enabled {
			return t, true
		}
	}
	return nil, false
}

// List returns all tools sorted by name (stable output for API responses).
func (r *ToolRegistry) List() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, *t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ListEnabled returns only enabled tools, sorted by name.
func (r *ToolRegistry) ListEnabled() []ToolDefinition {
	all := r.List()
	out := make([]ToolDefinition, 0, len(all))
	for _, t := range all {
		if t.Enabled {
			out = append(out, t)
		}
	}
	return out
}

// SetEnabled toggles a tool's enabled flag. It reports whether the tool exists.
func (r *ToolRegistry) SetEnabled(name string, enabled bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.tools[name]
	if !ok {
		return false
	}
	t.Enabled = enabled
	return true
}

// Clear removes all tools.
func (r *ToolRegistry) Clear() {
	r.mu.Lock()
	r.tools = make(map[string]*ToolDefinition)
	r.mu.Unlock()
}

// ── Tool call request / response ───────────────────────────────────────────

// ToolCallRequest is the frame sent from a downstream framework (LangBot) to
// E-SP-Line2 asking it to invoke an "inactive field" tool against a bridge.
type ToolCallRequest struct {
	// CallID correlates the request with its response.
	CallID string `json:"call_id"`
	// Tool is the adapter-local tool name (preferred).
	Tool string `json:"tool,omitempty"`
	// SemanticID may be supplied instead of Tool; the gateway resolves it to
	// an enabled tool.
	SemanticID SemanticID `json:"semantic_id,omitempty"`
	// InstanceID selects the bridge instance that owns the data.
	InstanceID string `json:"instance_id"`
	// ConversationID is optional context (e.g. to resolve the current order).
	ConversationID string `json:"conversation_id,omitempty"`
	// Arguments holds the tool arguments (validated against the definition).
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	// TraceID propagates request tracing.
	TraceID string `json:"trace_id,omitempty"`
	// TimeoutMS overrides the default invocation timeout.
	TimeoutMS int64 `json:"timeout_ms,omitempty"`
	// Confirm records that the caller (or user) confirmed a query_action tool.
	Confirm bool `json:"confirm,omitempty"`
}

// ToolCallResponse is the result of a tool invocation.
type ToolCallResponse struct {
	CallID     string      `json:"call_id"`
	Tool       string      `json:"tool"`
	SemanticID SemanticID  `json:"semantic_id,omitempty"`
	Success    bool        `json:"success"`
	Data       interface{} `json:"data,omitempty"`
	Error      string      `json:"error,omitempty"`
	ErrorCode  string      `json:"error_code,omitempty"`
	DurationMS int64       `json:"duration_ms,omitempty"`
}

// Tool error codes. They follow the ESPL2 convention of a 5-digit code whose
// prefix mirrors the HTTP-ish family (40x = client/caller, 50x = upstream).
const (
	// ToolErrNotFound means the requested tool is not registered.
	ToolErrNotFound = "40401"
	// ToolErrBridgeOffline means no live bridge connection exists for the
	// instance.
	ToolErrBridgeOffline = "40402"
	// ToolErrTimeout means the bridge did not answer within the deadline.
	ToolErrTimeout = "40403"
	// ToolErrInvalidArgs means arguments failed validation.
	ToolErrInvalidArgs = "40404"
	// ToolErrPermission means the adapter scope forbids the operation.
	ToolErrPermission = "40405"
	// ToolErrUpstream means the bridge reported a failure.
	ToolErrUpstream = "40406"
	// ToolErrConfirmRequired means a query_action tool was called without
	// confirmation.
	ToolErrConfirmRequired = "40407"
	// ToolErrDisabled means the tool exists but is disabled.
	ToolErrDisabled = "40408"
)

// ToolErrorMessage returns a human-readable message for a tool error code.
func ToolErrorMessage(code string) string {
	switch code {
	case ToolErrNotFound:
		return "tool not found"
	case ToolErrBridgeOffline:
		return "bridge instance offline"
	case ToolErrTimeout:
		return "tool call timeout"
	case ToolErrInvalidArgs:
		return "invalid tool arguments"
	case ToolErrPermission:
		return "adapter has no permission for this tool"
	case ToolErrUpstream:
		return "upstream bridge error"
	case ToolErrConfirmRequired:
		return "tool requires confirmation"
	case ToolErrDisabled:
		return "tool is disabled"
	default:
		return "unknown tool error"
	}
}

// ── Default ESPL2 tool catalog ─────────────────────────────────────────────

// DefaultESPL2Tools returns the built-in tool catalog every ESPL2 bridge
// supports. It exposes the e-commerce "inactive fields" the platform can
// actively query: 商品名称/简介/图片, 订单信息, 发货/物流单号, 活动信息.
func DefaultESPL2Tools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "get_product",
			SemanticID:  SemanticGetProduct,
			Category:    ToolCategoryQueryOnly,
			Description: "查询商品详情：名称、简介、图片、价格、规格、库存。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "查询商品详情：名称、简介、图片、价格、规格、库存。",
				"en_US":   "Get product detail: title, description, images, price, specs, stock.",
			},
			Parameters: []ToolParameter{
				{Name: "item_id", Type: ToolParamString, Description: "商品 ID（item_id）", Required: true},
				{Name: "with_images", Type: ToolParamBool, Description: "是否返回图片列表", Default: true},
				{Name: "with_description", Type: ToolParamBool, Description: "是否返回详细简介", Default: true},
			},
			Returns: "ProductDetail{ item_id,title,description,images[],price,specs[],stock,rating,review_count }",
			Enabled: true,
		},
		{
			Name:        "get_product_list",
			SemanticID:  SemanticGetProductList,
			Category:    ToolCategoryQueryOnly,
			Description: "查询/搜索商品列表。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "查询/搜索商品列表。",
				"en_US":   "List or search products.",
			},
			Parameters: []ToolParameter{
				{Name: "keyword", Type: ToolParamString, Description: "搜索关键词"},
				{Name: "category", Type: ToolParamString, Description: "分类"},
				{Name: "page", Type: ToolParamNumber, Description: "页码（从 1 开始）", Default: 1},
				{Name: "page_size", Type: ToolParamNumber, Description: "每页数量", Default: 20},
			},
			Returns: "ProductList{ total, items[] }",
			Enabled: true,
		},
		{
			Name:        "get_order",
			SemanticID:  SemanticGetOrder,
			Category:    ToolCategoryQueryOnly,
			Description: "查询订单信息：状态、金额、商品明细、支付信息。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "查询订单信息：状态、金额、商品明细、支付信息。",
				"en_US":   "Get order information: status, amount, items, payment.",
			},
			Parameters: []ToolParameter{
				{Name: "order_id", Type: ToolParamString, Description: "订单号", Required: true},
				{Name: "with_items", Type: ToolParamBool, Description: "是否返回商品明细", Default: true},
			},
			Returns: "OrderDetail{ order_id,status,amount,items[],payment,buyer_info,seller_info }",
			Enabled: true,
		},
		{
			Name:        "get_logistics",
			SemanticID:  SemanticGetLogistics,
			Category:    ToolCategoryQueryOnly,
			Description: "查询物流/发货信息：物流单号、承运商、轨迹。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "查询物流/发货信息：物流单号、承运商、轨迹。",
				"en_US":   "Get logistics: tracking number, carrier, traces.",
			},
			Parameters: []ToolParameter{
				{Name: "order_id", Type: ToolParamString, Description: "订单号", Required: true},
				{Name: "tracking_no", Type: ToolParamString, Description: "物流单号（可选，优先使用）"},
			},
			Returns: "LogisticsInfo{ order_id,tracking_no,carrier,status,traces[] }",
			Enabled: true,
		},
		{
			Name:        "ship_order",
			SemanticID:  SemanticShipOrder,
			Category:    ToolCategoryQueryAction,
			Description: "对订单发货并回填物流单号（有副作用）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "对订单发货并回填物流单号（有副作用）。",
				"en_US":   "Ship an order and attach the tracking number (side effect).",
			},
			Parameters: []ToolParameter{
				{Name: "order_id", Type: ToolParamString, Description: "订单号", Required: true},
				{Name: "tracking_no", Type: ToolParamString, Description: "物流单号", Required: true},
				{Name: "carrier", Type: ToolParamString, Description: "承运商代码（如 sf, yto, zto）"},
			},
			Returns:        "Shipping{ method,tracking_no,carrier,shipped_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "update_order",
			SemanticID:  SemanticUpdateOrder,
			Category:    ToolCategoryQueryAction,
			Description: "更新订单状态（如取消、退款、确认收货，有副作用）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "更新订单状态（如取消、退款、确认收货，有副作用）。",
				"en_US":   "Update order status (cancel/refund/confirm, side effect).",
			},
			Parameters: []ToolParameter{
				{Name: "order_id", Type: ToolParamString, Description: "订单号", Required: true},
				{Name: "status", Type: ToolParamString, Description: "目标状态", Required: true,
					Enum: []string{OrderStatusCancelled, OrderStatusRefunded, OrderStatusDelivered}},
				{Name: "remark", Type: ToolParamString, Description: "备注"},
			},
			Returns:        "OrderInfo{ order_id,status,updated_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "get_activity",
			SemanticID:  SemanticGetActivity,
			Category:    ToolCategoryQueryOnly,
			Description: "查询活动/事件信息（促销、秒杀、平台活动）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "查询活动/事件信息（促销、秒杀、平台活动）。",
				"en_US":   "Get activity/event information (promotions, flash sales).",
			},
			Parameters: []ToolParameter{
				{Name: "activity_id", Type: ToolParamString, Description: "活动 ID"},
				{Name: "item_id", Type: ToolParamString, Description: "关联商品 ID"},
			},
			Returns: "ActivityEvent{ event_type,activity_id,item_id,start_at,end_at,data }",
			Enabled: true,
		},
	}
}

// DefaultXianYuTools returns the tool catalog that mirrors the original
// XianYuApis (goofish_apis.py / goofish_live.py) surface. These cover every
// callable "inactive field" the reference implementation exposes so a LangBot
// adapter can drive the full 闲鱼 workflow: 登录 / 发布 / 改价 / 上传媒体 /
// 分类推荐 / 默认地址 / 会话历史 / 建会话 / 发消息.
//
// Bridges register this catalog in addition to DefaultESPL2Tools(); tools that
// a particular bridge cannot serve should be left disabled so the gateway
// reports them as ToolErrDisabled instead of failing at call time.
func DefaultXianYuTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "publish_item",
			SemanticID:  SemanticPublishItem,
			Category:    ToolCategoryQueryAction,
			Description: "发布闲鱼商品（含标题、简介、图片、价格、运费、分类、地址）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "发布闲鱼商品（含标题、简介、图片、价格、运费、分类、地址）。",
				"en_US":   "Publish a XianYu item (title, description, images, price, post fee, category, address).",
			},
			Parameters: []ToolParameter{
				{Name: "title", Type: ToolParamString, Description: "商品标题（itemTextDTO.title）", Required: true},
				{Name: "description", Type: ToolParamString, Description: "商品简介（itemTextDTO.desc）", Required: true},
				{Name: "images", Type: ToolParamArray, Description: "图片信息列表 imageInfoDOList[]（url/width/height）", Required: true},
				{Name: "price_in_cent", Type: ToolParamNumber, Description: "价格（分，itemPriceDTO.priceInCent）", Required: true},
				{Name: "orig_price_in_cent", Type: ToolParamNumber, Description: "原价（分，itemPriceDTO.origPriceInCent）"},
				{Name: "quantity", Type: ToolParamNumber, Description: "库存数量", Default: 1},
				{Name: "freebies", Type: ToolParamBool, Description: "是否包含赠品", Default: false},
				{Name: "item_type_str", Type: ToolParamString, Description: "商品类型，默认 b", Default: "b"},
				{Name: "simple_item", Type: ToolParamBool, Description: "是否为简单商品", Default: true},
				{Name: "unique_code", Type: ToolParamString, Description: "幂等唯一码"},
				{Name: "source_id", Type: ToolParamString, Description: "来源 ID（sourceId）"},
				{Name: "bizcode", Type: ToolParamString, Description: "业务码"},
				{Name: "publish_scene", Type: ToolParamString, Description: "发布场景 publishScene"},
				{Name: "item_cat", Type: ToolParamObject, Description: "分类 itemCatDTO{catId,catName,channelCatId,tbCatId}"},
				{Name: "item_addr", Type: ToolParamObject, Description: "地址 itemAddrDTO{area,city,divisionId,gps,poiId,poiName,prov}"},
				{Name: "item_post_fee", Type: ToolParamObject, Description: "运费 itemPostFeeDTO{canFreeShipping,supportFreight,onlyTakeSelf,templateId,postPriceInCent}"},
				{Name: "delivery", Type: ToolParamObject, Description: "配送设置 DeliverySettings{choice,post_price,can_self_pickup}"},
				{Name: "item_label_ext_list", Type: ToolParamArray, Description: "扩展标签 itemLabelExtList[]"},
				{Name: "user_rights_protocols", Type: ToolParamArray, Description: "用户权益协议 userRightsProtocols[]{enable,serviceCode}"},
				{Name: "default_price", Type: ToolParamNumber, Description: "默认价格 defaultPrice"},
			},
			Returns:        "ItemPublishResult{ item_id,status,publish_scene }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "update_item",
			SemanticID:  SemanticUpdateItem,
			Category:    ToolCategoryQueryAction,
			Description: "更新闲鱼商品信息（标题/简介/价格/库存/上下架）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "更新闲鱼商品信息（标题/简介/价格/库存/上下架）。",
				"en_US":   "Update a XianYu item (title, description, price, stock, on/off shelf).",
			},
			Parameters: []ToolParameter{
				{Name: "item_id", Type: ToolParamString, Description: "商品 ID", Required: true},
				{Name: "title", Type: ToolParamString, Description: "新标题"},
				{Name: "description", Type: ToolParamString, Description: "新简介"},
				{Name: "price_in_cent", Type: ToolParamNumber, Description: "新价格（分）"},
				{Name: "quantity", Type: ToolParamNumber, Description: "新库存"},
				{Name: "status", Type: ToolParamString, Description: "商品状态", Enum: []string{ItemStatusOnSale, ItemStatusOffShelf, ItemStatusSoldOut}},
			},
			Returns:        "ItemUpdateResult{ item_id,status,updated_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "update_item_price",
			SemanticID:  SemanticUpdateItemPrice,
			Category:    ToolCategoryQueryAction,
			Description: "调整闲鱼商品价格（仅改价，不修改其它字段）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "调整闲鱼商品价格（仅改价，不修改其它字段）。",
				"en_US":   "Adjust a XianYu item price only.",
			},
			Parameters: []ToolParameter{
				{Name: "item_id", Type: ToolParamString, Description: "商品 ID", Required: true},
				{Name: "price_in_cent", Type: ToolParamNumber, Description: "新价格（分）", Required: true},
				{Name: "orig_price_in_cent", Type: ToolParamNumber, Description: "新原价（分）"},
			},
			Returns:        "Price{ current_price,original_price,currency }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "get_category_recommend",
			SemanticID:  SemanticGetCategoryRecommend,
			Category:    ToolCategoryQueryOnly,
			Description: "根据商品标题获取推荐分类与属性（categoryPredictResult）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "根据商品标题获取推荐分类与属性（categoryPredictResult）。",
				"en_US":   "Get recommended category/attributes from an item title (categoryPredictResult).",
			},
			Parameters: []ToolParameter{
				{Name: "title", Type: ToolParamString, Description: "商品标题", Required: true},
				{Name: "description", Type: ToolParamString, Description: "商品简介"},
			},
			Returns: "CategoryRecommend{ cat_id,cat_name,channel_cat_id,tb_cat_id,card_list[],category_predict_result }",
			Enabled: true,
		},
		{
			Name:        "get_default_address",
			SemanticID:  SemanticGetDefaultAddress,
			Category:    ToolCategoryQueryOnly,
			Description: "查询默认发货地址（commonAddresses）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "查询默认发货地址（commonAddresses）。",
				"en_US":   "Get the default shipping address (commonAddresses).",
			},
			Parameters: []ToolParameter{
				{Name: "gps", Type: ToolParamString, Description: "定位坐标（可选）"},
				{Name: "division_id", Type: ToolParamString, Description: "行政区划 ID（可选）"},
			},
			Returns: "DefaultAddress{ poi_id,poi_name,area,city,prov,division_id,gps }",
			Enabled: true,
		},
		{
			Name:        "upload_media",
			SemanticID:  SemanticUploadMedia,
			Category:    ToolCategoryQueryAction,
			Description: "上传图片/媒体到闲鱼 CDN，返回可用的图片 URL 与 pix。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "上传图片/媒体到闲鱼 CDN，返回可用的图片 URL 与 pix。",
				"en_US":   "Upload media to the XianYu CDN, returning object.url and object.pix.",
			},
			Parameters: []ToolParameter{
				{Name: "file_path", Type: ToolParamString, Description: "本地文件路径"},
				{Name: "url", Type: ToolParamString, Description: "远程 URL（与 file_path 二选一）"},
				{Name: "content_type", Type: ToolParamString, Description: "MIME 类型，默认 image/jpeg"},
			},
			Returns:        "MediaUpload{ url,pix,width,height,size }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "refresh_token",
			SemanticID:  SemanticRefreshToken,
			Category:    ToolCategoryQueryAction,
			Description: "刷新闲鱼登录态（loginuser.get），返回新的 cookies / token。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "刷新闲鱼登录态（loginuser.get），返回新的 cookies / token。",
				"en_US":   "Refresh the XianYu login session (loginuser.get).",
			},
			Parameters: []ToolParameter{
				{Name: "cookies", Type: ToolParamString, Description: "当前 cookies 字符串"},
			},
			Returns:        "TokenResult{ cookies,user_id,display_name,refreshed_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "get_token",
			SemanticID:  SemanticGetToken,
			Category:    ToolCategoryQueryOnly,
			Description: "使用登录态换取闲鱼 IM token（login.token）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "使用登录态换取闲鱼 IM token（login.token）。",
				"en_US":   "Exchange the login session for a XianYu IM token (login.token).",
			},
			Parameters: []ToolParameter{
				{Name: "cookies", Type: ToolParamString, Description: "当前 cookies 字符串", Required: true},
			},
			Returns: "TokenResult{ cookies,user_id,display_name,access_token }",
			Enabled: true,
		},
		{
			Name:        "login_qrcode",
			SemanticID:  SemanticLoginQRCode,
			Category:    ToolCategoryQueryAction,
			Description: "获取闲鱼扫码登录二维码，并轮询登录状态。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "获取闲鱼扫码登录二维码，并轮询登录状态。",
				"en_US":   "Fetch the XianYu QR login code and poll login status.",
			},
			Parameters: []ToolParameter{
				{Name: "action", Type: ToolParamString, Description: "操作类型", Default: "get",
					Enum: []string{"get", "poll"}},
				{Name: "session_id", Type: ToolParamString, Description: "轮询时的会话 ID"},
			},
			Returns:        "QRCodeLogin{ qr_url,session_id,status,cookies }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "get_conversation_history",
			SemanticID:  SemanticGetConversationHistory,
			Category:    ToolCategoryQueryOnly,
			Description: "拉取会话历史消息（listUserMessages，支持游标分页）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "拉取会话历史消息（listUserMessages，支持游标分页）。",
				"en_US":   "Fetch conversation history (listUserMessages, cursor paginated).",
			},
			Parameters: []ToolParameter{
				{Name: "cid", Type: ToolParamString, Description: "会话 ID（cid）", Required: true},
				{Name: "next_cursor", Type: ToolParamString, Description: "分页游标 nextCursor"},
				{Name: "count", Type: ToolParamNumber, Description: "拉取条数", Default: 20},
			},
			Returns: "ConversationHistory{ has_more,next_cursor,messages[] }",
			Enabled: true,
		},
		{
			Name:        "create_conversation",
			SemanticID:  SemanticCreateConversation,
			Category:    ToolCategoryQueryAction,
			Description: "创建/打开与指定用户的单聊会话（SingleChatConversation.create）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "创建/打开与指定用户的单聊会话（SingleChatConversation.create）。",
				"en_US":   "Create/open a single chat conversation (SingleChatConversation.create).",
			},
			Parameters: []ToolParameter{
				{Name: "pair_first", Type: ToolParamString, Description: "发起方用户 ID（pairFirst）", Required: true},
				{Name: "pair_second", Type: ToolParamString, Description: "接收方用户 ID（pairSecond）", Required: true},
				{Name: "biz_type", Type: ToolParamString, Description: "业务类型 bizType"},
				{Name: "item_id", Type: ToolParamString, Description: "关联商品 ID（extension.itemId）"},
				{Name: "ctx", Type: ToolParamString, Description: "上下文 ctx"},
			},
			Returns:        "ConversationCreate{ cid,conversation_type,created_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "send_message",
			SemanticID:  SemanticSendMessage,
			Category:    ToolCategoryQueryAction,
			Description: "发送闲鱼消息（sendByReceiverScope，支持文本/图片/音频/交易卡片）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "发送闲鱼消息（sendByReceiverScope，支持文本/图片/音频/交易卡片）。",
				"en_US":   "Send a XianYu message (sendByReceiverScope; text/image/audio/trade card).",
			},
			Parameters: []ToolParameter{
				{Name: "cid", Type: ToolParamString, Description: "会话 ID（cid）", Required: true},
				{Name: "conversation_type", Type: ToolParamNumber, Description: "会话类型（conversationType）", Default: 1},
				{Name: "content_type", Type: ToolParamNumber, Description: "内容类型 contentType", Default: 1,
					Enum: []string{ContentTypeText, ContentTypeImage, ContentTypeAudio, ContentTypeTradeCard}},
				{Name: "text", Type: ToolParamString, Description: "文本内容（contentType=1 时）"},
				{Name: "image_url", Type: ToolParamString, Description: "图片 URL（contentType=2 时）"},
				{Name: "audio_url", Type: ToolParamString, Description: "音频 URL（contentType=3 时）"},
				{Name: "duration_ms", Type: ToolParamNumber, Description: "音频时长（毫秒）"},
				{Name: "custom", Type: ToolParamObject, Description: "自定义内容 custom{type,data(base64)}"},
				{Name: "uuid", Type: ToolParamString, Description: "消息幂等 UUID"},
				{Name: "red_point_policy", Type: ToolParamNumber, Description: "红点策略 redPointPolicy", Default: 0},
				{Name: "ext_json", Type: ToolParamString, Description: "扩展 JSON（extJson）"},
				{Name: "mtags", Type: ToolParamString, Description: "消息标签 mtags"},
				{Name: "msg_read_status_setting", Type: ToolParamNumber, Description: "已读状态设置", Default: 1},
				{Name: "actual_receivers", Type: ToolParamArray, Description: "实际接收人 actualReceivers[]"},
			},
			Returns:        "SendMessageResult{ uuid,msg_id,sent_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
	}
}

// DefaultTaoBaoTools returns the tool catalog that mirrors the original
// TaoBaoApis (taobao_apis.py / taobao_live.py) surface. These cover every
// callable "inactive field" the reference implementation exposes so a LangBot
// adapter can drive the full 淘宝 workflow: 获取 IM Token / 解析商品用户 /
// 上传媒体 / 会话历史 / 建会话 / 发消息.
//
// Tool names carry a "taobao_" prefix so they never collide with the
// XianYuApis-compatible tools of the same intent. The gateway keeps a single
// ToolRegistry keyed by name, and a bridge is selected by instance_id, so the
// names must be globally unique even though both catalogs intentionally share
// the same SemanticID. The Platform field is set to "taobao" so callers can
// filter by platform.
//
// Bridges register this catalog in addition to DefaultESPL2Tools(); tools that
// a particular bridge cannot serve should be left disabled so the gateway
// reports them as ToolErrDisabled instead of failing at call time.
func DefaultTaoBaoTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "taobao_get_token",
			SemanticID:  SemanticGetToken,
			Category:    ToolCategoryQueryOnly,
			Platform:    "taobao",
			Description: "使用淘宝登录态换取 IM token（mtop.taobao.login.token.get.h5）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "使用淘宝登录态换取 IM token（mtop.taobao.login.token.get.h5）。",
				"en_US":   "Exchange the Taobao login session for an IM token (mtop.taobao.login.token.get.h5).",
			},
			Parameters: []ToolParameter{
				{Name: "cookies", Type: ToolParamString, Description: "当前 cookies 字符串", Required: true},
			},
			Returns: "TokenResult{ cookies, user_id, access_token }",
			Enabled: true,
		},
		{
			Name:        "taobao_get_goods_info",
			SemanticID:  SemanticGetGoodsInfo,
			Category:    ToolCategoryQueryOnly,
			Platform:    "taobao",
			Description: "解析淘宝商品页面，提取卖家 userId 与 data-encryptuid（用于建会话）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "解析淘宝商品页面，提取卖家 userId 与 data-encryptuid（用于建会话）。",
				"en_US":   "Parse a Taobao item page to extract seller userId and data-encryptuid (used to create conversations).",
			},
			Parameters: []ToolParameter{
				{Name: "goods_url", Type: ToolParamString, Description: "淘宝/天猫商品详情页 URL", Required: true},
			},
			Returns: "GoodsInfo{ uid, encrypt_uid }",
			Enabled: true,
		},
		{
			Name:        "taobao_upload_media",
			SemanticID:  SemanticUploadMedia,
			Category:    ToolCategoryQueryAction,
			Platform:    "taobao",
			Description: "上传图片到淘宝 CDN（stream-upload.taobao.com），返回 fileId / url / pix。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "上传图片到淘宝 CDN（stream-upload.taobao.com），返回 fileId / url / pix。",
				"en_US":   "Upload an image to the Taobao CDN (stream-upload.taobao.com), returning fileId / url / pix.",
			},
			Parameters: []ToolParameter{
				{Name: "file_path", Type: ToolParamString, Description: "本地图片文件路径", Required: true},
			},
			Returns:        "MediaUpload{ object{url,fileId,pix,size,width,height} }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "taobao_get_conversation_history",
			SemanticID:  SemanticGetConversationHistory,
			Category:    ToolCategoryQueryOnly,
			Platform:    "taobao",
			Description: "拉取淘宝会话历史消息（/r/MessageManager/listUserMessages，支持游标分页）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "拉取淘宝会话历史消息（/r/MessageManager/listUserMessages，支持游标分页）。",
				"en_US":   "Fetch Taobao conversation history (/r/MessageManager/listUserMessages, cursor paginated).",
			},
			Parameters: []ToolParameter{
				{Name: "cid", Type: ToolParamString, Description: "会话 ID（cid）", Required: true},
				{Name: "next_cursor", Type: ToolParamString, Description: "分页游标 nextCursor"},
				{Name: "count", Type: ToolParamNumber, Description: "拉取条数", Default: 20},
			},
			Returns: "ConversationHistory{ has_more, next_cursor, messages[] }",
			Enabled: true,
		},
		{
			Name:        "taobao_create_conversation",
			SemanticID:  SemanticCreateConversation,
			Category:    ToolCategoryQueryAction,
			Platform:    "taobao",
			Description: "创建/打开与指定淘宝用户的单聊会话（/r/SingleChatConversation/create，需要 encryptUid）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "创建/打开与指定淘宝用户的单聊会话（/r/SingleChatConversation/create，需要 encryptUid）。",
				"en_US":   "Create/open a single chat conversation with a Taobao user (/r/SingleChatConversation/create, requires encryptUid).",
			},
			Parameters: []ToolParameter{
				{Name: "encrypt_uid", Type: ToolParamString, Description: "接收方 encryptUid（从 taobao_get_goods_info 获得）", Required: true},
			},
			Returns:        "ConversationCreate{ cid, conversation_type, created_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
		{
			Name:        "taobao_send_message",
			SemanticID:  SemanticSendMessage,
			Category:    ToolCategoryQueryAction,
			Platform:    "taobao",
			Description: "发送淘宝消息（/r/MessageSend/sendByReceiverScope，支持文本/图片）。",
			DescriptionI18n: map[string]string{
				"zh_Hans": "发送淘宝消息（/r/MessageSend/sendByReceiverScope，支持文本/图片）。",
				"en_US":   "Send a Taobao message (/r/MessageSend/sendByReceiverScope; text/image).",
			},
			Parameters: []ToolParameter{
				{Name: "cid", Type: ToolParamString, Description: "会话 ID（cid）", Required: true},
				{Name: "toid", Type: ToolParamString, Description: "接收方用户 ID", Required: true},
				{Name: "sender_nick", Type: ToolParamString, Description: "发送方昵称（如 cntaobao{_nk_}）"},
				{Name: "content_type", Type: ToolParamNumber, Description: "内容类型 contentType", Default: 1,
					Enum: []string{ContentTypeText, "101"}},
				{Name: "text", Type: ToolParamString, Description: "文本内容（contentType=1 时）"},
				{Name: "image_url", Type: ToolParamString, Description: "图片 URL（contentType=101 时）"},
				{Name: "file_id", Type: ToolParamString, Description: "上传媒体后得到的 fileId"},
				{Name: "width", Type: ToolParamNumber, Description: "图片宽度"},
				{Name: "height", Type: ToolParamNumber, Description: "图片高度"},
			},
			Returns:        "SendMessageResult{ uuid, msg_id, sent_at }",
			Enabled:        true,
			RequireConfirm: true,
		},
	}
}
