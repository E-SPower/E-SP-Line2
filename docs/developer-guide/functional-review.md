# 功能实现审查报告

审查对象：`E-SP-Line2`（`/home/typer/ESPL2/E-SP-Line2`，版本 `1.0.0` 加一批未提交改动）。
审查方式：静态通读 + 实际起服务验证（`PORT=18xxx` 临时端口、独立 SQLite 库、真实 HTTP/WebSocket 客户端）。
审查日期：2026-11-07。

## 0. 结论速览

**这个系统目前是一套"能跑的 CRUD 后台"，还不是一个"消息接入器"。**

登录、列表、建实例、启停适配器、审批消息都能对着数据库跑通，但**产品的核心承诺——平台消息进来、按规则路由、投给下游应用——整条链路没有接通**：WebSocket 是回显、`message.Hub` 从未被实例化、没有消息入库的 HTTP 入口、建命令不会派发、路由规则从不变动内存路由表。

叠加一个更严重的鉴权问题：**`AuthRequired` 只检查 `Authorization` 头是否是 `Bearer xxx` 的形状，完全不验签**，任意字符串都能当令牌用；所有实体的 `user_id` 都是写死的 `user-123`，多用户之间没有隔离。

好消息是基础设施健康度不差：`go vet`、`go build`、`CGO_ENABLED=1 go test ./...`、`npx tsc --noEmit`、`python3 scripts/check_generated.py matrix.json`、`npm run lint`、`gofmt -l` 全部通过。问题集中在"接线"，不在"能不能编译"。另外前端构建在只读工作树上会因 Vite 要写临时配置文件而失败（m19）；`CONTRIBUTING.md` 那个与 `engines` 相反的 Node `18+` 也在审查中被顺手改成了 `20.11+`（m20）。

### 实测命令结果

| 命令 | 结果 |
| --- | --- |
| `CGO_ENABLED=1 go test ./...` | **通过**（`adapter` / `messagefields` / `platformspec` 三包有测试） |
| `CGO_ENABLED=0 go test ./...` | **失败**，`adapter` 包两个用例 `requires cgo to work. This is a stub` |
| `go vet ./...` | 通过（仅 go-sqlite3 的 C 警告） |
| `npx tsc --noEmit` | 通过 |
| `python3 scripts/check_generated.py matrix.json` | 通过（生成物未过期） |
| `npm run lint` | 通过（`--max-warnings 23`，23 条全是 `no-explicit-any`） |
| `gofmt -l backend` | 无输出 |
| `npx vite build`（仓库只读时） | **失败** `EROFS ... vite.config.ts.timestamp-*.mjs`；拷到可写目录后通过（m19） |
| `npm run build` | 同上（内含 `vite build`） |

> 审查期间仓库被并发修改（`dispatcher.go`、`server.go`、`package.json`、`.eslintrc.cjs`、`.github/workflows/ci.yml` 都有变动，CI 期间加入了 `golangci-lint`）。**下文每条结论都在最后一次改动之后重新跑过一遍**；行号按最新工作树复核。若后续再改，以代码为准。

---

## 1. Blocker

### B1. ~~`AuthRequired` 完全不校验 JWT~~（基线更新时已在主干修复）

> **已修复（本条保留作背景）**：`internal/middleware/auth.go` 现在有 `ParseToken`，`AuthRequired` 真验签并由 claims 写 context。本树实测：`Bearer total-garbage` → **401**，真 token 打 `/auth/me` → **200** 且返回真实用户。下面记录的是修复前的状态。

**现象**：`middleware.AuthRequired` 只做两件事——要求 `Authorization` 头存在、且形如 `Bearer <something>`；然后无条件写入写死的身份并放行。

实测：

```
$ curl -H 'Authorization: Bearer total-garbage-not-a-jwt' /api/v1/adapters   → 200 + 全量数据
$ curl -H 'Authorization: Bearer aaaa.bbbb.cccc'        /api/v1/routes       → 200
$ curl -H 'Authorization: Bearer garbage'               /api/v1/auth/me      → 404 {"error":"user not found"}
```

连真实 JWT 也是 404 —— 因为 `userID` 被写死成 `user-123`，库里根本没有这个用户。注册流程返回的那个可用令牌，从来没被校验过。

**定位**：
- `internal/middleware/auth.go:34` — `// TODO: Validate token using auth service`
- `internal/middleware/auth.go:36` — `_ = token`（拿到令牌后直接丢弃）
- `internal/middleware/auth.go:39-41` — 写死 `userID`/`username`/`role`
- `internal/service/auth.go:109` — 实现了 `ValidateToken`，**全仓零调用点**（`grep -rn ValidateToken` 只命中定义本身）

**影响**：鉴权等于没有。任何人只要发一个 `Authorization: Bearer x` 就能读写全部资源；`AdminRequired`（`middleware/auth.go:48`）也读不到真实 `role`，且当前没有任何路由挂它。

**建议**：`AuthRequired` 改为接收 `*service.AuthService`，取 `parts[1]` 调 `ValidateToken`，失败回 401，成功则把 `claims.UserID/Username/Role` 写进 context。`server.go` 里 11 处 `middleware.AuthRequired()` 同步改传参。注意 gin 的 group 中间件在 `Use` 之前注册的路由不生效（见 `server.go:75-86` 的现有写法），改签名时要一并核对这些位置。

### B2. 两条 WebSocket 链路都是回显，没有接进 `message.Dispatcher` / `message.Router`

**现象**：`/ws/adapter` 和 `/ws/app` 读完一条消息后，包一层 `{"type":"ack","data":"<原样字符串>"}` 写回**同一个连接**，注释就写着 "Echo back for now"。

实测（真实 WebSocket 握手 + 发一帧 v3 envelope）：

```
[adapter] welcome: {"type":"connected",...}
[adapter] after-send: {"data":"{\"protocol_version\":\"v3\",...}","type":"ack",...}
[app]     after-send: 同上（app 连接也只收到自己那帧的回显）
跨连接：adapter 发帧 → 只有 adapter 自己收到，app 一帧没收到
```

`instance_id` / `app_id` 两个 query 参数只用于打日志，从未参与任何注册。

**根因**：`message.NewHub()`（`internal/message/hub.go:22`）**在全仓没有调用点**。`Dispatcher` / `Router` / `Hub` 三个类型连同 `RegisterConnection`、`ProcessMessage`、`SendCommand` 全是死代码——`grep -rln 'internal/message"' backend` 的结果是**没有任何包导入 `internal/message`**。`service.Services`（`internal/service/services.go:10-21`）里也没有 `Hub` 字段。

**定位**：
- `internal/handler/websocket.go:60` 与 `:112` — 两处 `// Echo back for now (implement actual message processing)`
- `internal/handler/websocket.go:50`、`:102` — 读循环只用局部变量，不回填 dispatcher
- `internal/message/hub.go:22` — `NewHub` 无调用方
- `internal/server/server.go:146-150` — 两条 WS 路由直接传 service，没有 hub 参与

**影响**：适配器连上来发消息，服务端不会解析、不会入库、不会路由、不会转发给应用。WebSocket 在产品意义上是空的。

**建议**：`Services` 增加 `Hub *message.Hub` 并在 `NewServices` 里 `message.NewHub()`；`Close` 里关掉。两条 handler 改为 `hub.RegisterAdapter/RegisterApp(&message.Connection{...})` 后不再自己读循环，交给 `dispatcher` 的 `readPump`/`writePump`（它们已经写好且带 ping/pong 与心跳）。路由 key 用登录/握手身份，不要用 query 明文。

### B3. 没有消息入库的任意入口，`MessageService.Create` 不可达

**现象**：`messages` 组只注册了 `GET ""`、`GET "/:id"`、`POST "/:id/ack"`。`POST /api/v1/messages` 实测 **404 page not found**。

`MessageService.Create`（`internal/service/message.go:35`）写好了幂等、trace、状态，但**没有任何 handler 调它**。

**定位**：
- `internal/server/server.go:118-122` — 消息路由组只有三个只读/审批动作
- `internal/service/message.go:35` — `Create` 无调用方

**影响**：入站消息唯一可能的来源是 B2 里那个回显的 WebSocket，而它也不入库。所以 `GET /messages` 永远是空数组——实测连续发 WS 帧后 `GET /messages` 仍返回 `{"data":[],"total":0}`。

**建议**：补 `POST /api/v1/messages`（或独立的 `/api/v1/ingest`），绑定 `CreateMessageRequest`，落库后调 `hub.ProcessMessage`。给 `platform_id`+`idempotency_key` 做唯一约束配合现有 `FindByIdempotencyKey`。

### B4. `inbound_events.idempotency_key` 唯一索引 + 空串，第二条不带 key 的消息必然失败

**现象**：`IdempotencyKey` 标了 `uniqueIndex`，但 SQLite 的 `UNIQUE` 允许**多行 NULL、只允许一行空串**。Go 的 `string` 零值是 `""` 而非 NULL，所以第二条 `idempotency_key` 为空的插入直接撞唯一约束。

已用最小复现验证：

```
first  insert: ok
second insert with empty idempotency_key:
  UNIQUE constraint failed: inbound_events.idempotency_key
```

一旦 B3 把入口接上，这个问题会立刻变成"**第二条消息开始全部 500**"。

**定位**：
- `internal/models/models.go:75` — `IdempotencyKey string` + `gorm:"uniqueIndex"`
- `internal/service/message.go:37-42` — 只有 key 非空时才查重，空 key 直接走插入
- `internal/service/message.go:44-59` — 空 key 原样 `""` 写库

**影响**：接入接完后即刻触发，且因为 B3 尚未接通，目前是**潜伏缺陷**——上线接消息就炸。

**建议**：改成 `*string` 让空值为 NULL；或在 `Create` 里生成一个兜底 key（如 `models.GenerateIdempotencyKey(platform, instance, GenerateID())`）。配合唯一索引改成 `uniqueIndex` + 非空更稳。

### B5. 建命令只落库不派发，`MarkSent` / `MarkFailed` 无处调用

**现象**：`POST /api/v1/commands` 写一行 `status:"created"` 就返回 201，之后没有任何东西把它推给适配器，也没有东西推进状态。实测建完命令后 status 一直是 `created`。

`CommandService` 里 `MarkSent`、`MarkFailed`、`ListPending` 三个方法全仓无调用方。

**定位**：
- `internal/handler/command.go:45-60` — `CreateCommand` 只调 `commandService.Create`
- `internal/service/command.go:86`、`:99`、`:78` — `MarkSent` / `MarkFailed` / `ListPending` 无调用方
- `internal/message/hub.go:98` — `SendCommand` 同样无调用方

**影响**：下行方向（应用→平台）完全不通。命令表只是个流水账。

**建议**：`CreateCommand` 落库后调 `hub.SendCommand(cmd.InstanceID, ...)` 并 `MarkSent`；失败走 `MarkFailed`。补一个重试扫描（`ListPending` + 定时器）消费 `retrying`。

---

## 2. Major

### M1. ~~所有实体归属写死 `user-123`~~（基线更新时已在主干修复）

**现象**：`CreateInstance` 从 context 取 `userID`，但那是 B1 写死的值。实测 alice 建的实例 `user_id` 是 `"user-123"`；bob 登录后 `GET /instances` **看得到** alice 的实例，`DELETE` 也能删掉（200）。

`AdapterInstance` 有 `UserID` 字段和索引（`models.go:48`），`FindByUserID`（`repository/instance.go:34`）也写好了，但 `List` 走的是无过滤的 `repo.List`（`service/instance.go:102-110`）。

**定位**：
- `internal/handler/instance.go:54-55` — 取的是假 `userID`
- `internal/service/instance.go:102-110` — `List` 不按用户过滤
- `internal/repository/instance.go:34` — `FindByUserID` 无反调用方

**影响**：多租户不存在。任何登录用户可增删改别人的实例、适配器、路由。

**建议**：修完 B1 后，列表与写操作按 `c.Get("userID")` 收窄；`Update`/`Delete` 先查归属再动手，非本人回 403。

### M2. CORS `*` 与 `Allow-Credentials: true` 组合

**现象**：无条件 `Access-Control-Allow-Origin: *` 同时 `Access-Control-Allow-Credentials: true`。实测来自 `https://evil.example` 的预检照样拿到这两个头。

**定位**：`internal/middleware/middleware.go:70` 与 `:74`。

**影响**：`*` 时浏览器带不了凭据；将来若改用 Cookie 承载会话，跨域请求会静默丢 Cookie。当前前端把 JWT 放 `Authorization` 头（不受此限），所以**现在是隐患而非活故障**——属于"配置错但暂时没踩到"。

**建议**：要么回显 `Origin` 并配白名单，要么明确只走 Bearer 不打算用 Cookie，把 `Allow-Credentials` 删掉以免误导。

**现已修复**（P1/ESPL-20）：走了"回显 + 白名单"这条路，因为前端将来未必一直用 Bearer。`internal/middleware/middleware.go` 的 `CORS` 现在只对 `security.allowed_origins` 里的 Origin 回 `Access-Control-Allow-Origin: <origin>` 并保留 `Allow-Credentials: true`，同时加 `Vary: Origin`；不带 `Origin` 的客户端（curl、Python 桥）完全不受影响。实测：允许的 Origin 预检拿到 204 + 回显 + `Vary`，外来 Origin 只剩 `Vary`、无 `Access-Control-Allow-Origin`。

### M3. 默认 JWT secret 两处硬编码，且没有任何"是否已改"的检查

**定位**：`internal/config/config.go:126`（viper 默认值）、`config/config.yaml:32`（随仓库提交的值），内容都是 `change-this-secret-in-production`。

**影响**：部署方照抄配置就等于用公开密钥签发令牌，可离线伪造任意身份（叠加 B1 之后更不需要密钥，但修好 B1 就会立刻变成真问题）。

**建议**：`config.Load` 后若 secret 仍是占位串则打 WARN（或直接拒绝启动）；README/SECURITY 已有的提醒保留。

**现已修复**（P1/ESPL-20）：选了 WARN 而不是拒启，避免把现有部署一下卡死。判定收敛到 `config.UsingPlaceholderJWTSecret()`（占位串或空串都算），`server.New` 启动时打一条 WARN 并带修法提示。只做前缀比较不够——`change-this-secret-in-production-prod` 已是真轮换，不该报警，测试里钉住了这条。实测启动日志：`WARN ... JWT secret is still the shipped placeholder`。

### M4. WebSocket `CheckOrigin` 恒真，且握手不做鉴权

**定位**：`internal/handler/websocket.go:17` — `return true // Allow all origins for development`。

**影响**：任意站点可开 WS 订阅/灌入消息。叠加 B1/B2，WS 侧连"令牌形状"都不要求（两条 WS 路由不挂任何中间件，见 `server.go:145-150`），只要知道 `instance_id` / `app_id` 就能冒充。

**建议**：接 B1 后校验 `Sec-WebSocket-Protocol` 或 `?token=`，`CheckOrigin` 收白名单。

**现已修复**（P1/ESPL-20）：两条 WS 在**升级前**校验 token，缺失或非法都回 401（不再只是"验个形状"）；`CheckOrigin` 改为查与 CORS 同一份 `security.allowed_origins`，跨域来源在升级前被 403 拒掉；适配器网关的 upgrader 也接了同一份列表，`GatewayConfig.AllowedOrigin` 注入。token 三种承载：`?token=`（Python 桥，已同步改 `adapters/*/esp_bridge.py`）、`Authorization: Bearer`、`Sec-WebSocket-Protocol: espl, <jwt>`（浏览器唯一可设的头）。无 `Origin` 的非浏览器客户端放行。注意这是**鉴权到位**，不是**授权**：握手只验 token 有效，"这个 token 能不能碰这个 instance_id"仍属 M1。

### M5. `POST /platforms` 等三个 handler 有实现、无路由

**现象**：`CreatePlatform`、`UpdatePlatform`、`DeletePlatform` 三个 handler 完整存在，但 `server.go` 只注册了 `GET ""` 和 `GET "/:id"`。实测 `POST /api/v1/platforms` → **404**。

同类：`AdapterSession` 的 `GetSession` / `UpdateSessionHeartbeat`（`service/instance.go:113,118`）也无入口。

**定位**：`internal/handler/platform.go:45/64/84`、`internal/server/server.go:74-86`。

**影响**：平台数据只能靠启动时 `SyncBuiltins` 灌，运维侧无法增改；且 `/platforms` 与 `/platforms/spec` 两套模型并存，前端用后者、API 文档写前者（见 m4）。

**建议**：补齐三条路由（平台管理属管理动作，可挂 `AuthRequired` + `AdminRequired`）。

### M6. 前端所有"新建/编辑/删除"按钮都没接线

**现象**：`apiClient` 里有 `createAdapter` / `createInstance` / `createRoute` / `createCommand`，**四个函数零调用**（`grep -rn 'createAdapter\|createInstance\|createRoute\|createCommand' web/src/pages/` 无结果）。页面上的 `<Plus />` 按钮连 `onClick` 都没有：

| 页面 | `onClick` 数量 |
| --- | --- |
| `Instances.tsx` | 0 |
| `Routes.tsx` | 0 |
| `Platforms.tsx` | 0 |
| `Adapters.tsx` | 1（只有启停） |

编辑/删除更彻底：`client.ts` 里**根本没有** `delete*` / `update*` 方法。

**定位**：`web/src/pages/Instances.tsx:51`、`web/src/pages/Routes.tsx:54`、`web/src/pages/Platforms.tsx:60`、`web/src/api/client.ts`（缺方法）。

**影响**：十个页面全是"只能看"的只读视图；除适配器启停和消息审批外，UI 完不成任何写操作。

**建议**：补 `delete*/update*` 客户端方法 + 表单弹窗。这是纯增量工作，接口层（除 M5 缺的三条）已就绪。

### M7. ~~硬编码 `http://localhost:8080`~~（基线更新时已在主干修复）

**现象**：API 基址与健康检查地址都写死。实测 `vite build` 产物中 `localhost:8080` 出现 2 次，打进 bundle。

`vite.config.ts:41-53` 配了 `/api`、`/ws` 代理，但只在 `vite dev` 生效；产物不走代理，也不读 `import.meta.env`。

**定位**：`web/src/api/client.ts:1`（`API_BASE_URL`）、`web/src/api/client.ts:178`（`fetch('http://localhost:8080/health')`）。

**影响**：把 `dist` 部署到任何非本机场景，前后端直接失联；`http://` 还会在 https 页面上触发混合内容拦截。

**建议**：改成 `import.meta.env.VITE_API_BASE_URL ?? '/api/v1'`，健康检查同样走相对路径；部署用反向代理托管同源。

### M8. ~~无 401 处理~~（基线更新时已在主干修复）

**现象**：`request()` 在 `!response.ok` 时统一 `return { error }`，不区分状态码；`401` 也只是一个字符串错误。没有拦截器、没有 `clearToken()` 调用、没有跳转。

`clearToken()`（`client.ts:22`）同样零调用——登出用的是 `localStorage.removeItem('auth_token')`（`components/Layout.tsx:106`），绕过了 client 的内存副本，所以**登出后内存里的 token 仍在**，直到整页刷新。

**定位**：`web/src/api/client.ts:45-48`、`web/src/api/client.ts:22`、`web/src/components/Layout.tsx:105-108`。

**影响**：JWT 24h（`config.yaml:33`）过期后，用户看到的是每条列表各自报错，得手动清 localStorage。

**建议**：`request()` 里 `if (response.status === 401) { this.clearToken(); location.assign('/login') }`；`handleLogout` 改调 `apiClient.clearToken()`。

### M9. 前端无路由守卫，未登录可直接进主界面

**现象**：`App.tsx:17-31` 里 `/` 分组没有任何鉴权判断，`/login` 与主界面平级。未登录访问 `/dashboard` 会渲染完整外壳，只是各页各自报错。

**定位**：`web/src/App.tsx:17-30`。

**影响**：观感上像"进去但没数据"，与"需要登录"的预期不符。

**建议**：`/` 外包一层读 token 的 `RequireAuth`，无 token 重定向 `/login`。

### M10. Monitoring 读的字段在实例模型上不存在，指标恒为 0

**现象**：`Monitoring.tsx` 映射 `instance.last_heartbeat`、`instance.message_count`、`instance.error_count`；而 `models.AdapterInstance`（`models.go:41-51`）只有 `id/adapter_id/platform_id/name/config/status/user_id/created_at/updated_at`，这三个字段一个都没有。

`running` 状态同样永远拿不到：`AdapterInstance.Status` 只在建实例时写死 `"stopped"`（`service/instance.go:48`），`StartAdapter` 改的是 **`AdapterPackage.Status`**（`service/adapter.go:120-134`，`AdapterService` 压根没持有 `InstanceRepository`）；`running` 只存在于 `adapter.Manager` 那个未接线的内存模型里（`adapter/manager.go:19`）。

**定位**：`web/src/pages/Monitoring.tsx:86-88`、`internal/models/models.go:41-51`、`internal/service/instance.go:48`、`internal/service/adapter.go:120`。

**影响**：监控页除"总数"外的所有数字恒为 0，状态恒为 "stopped"，错误率恒 `0.00%`。页面看起来在工作，数据是假的。

**建议**：给 `AdapterInstance` 补上三个统计字段（或改读 `adapter_sessions` + 聚合查询）；实例级启停走 `InstanceService` 而不是 `AdapterService`。

### M11. 空操作也回 200，"删了不存在的资源"是假成功

**现象**：

```
POST /messages/does-not-exist/ack   → 200 {"message":"message acknowledged"}
DELETE /routes/nope                 → 200 {"message":"route deleted"}
```

原因是 `UpdateStatus` / `Delete` 用 `db.Model(...).Update()` / `db.Delete()`，**不检查 RowsAffected**（`repository/message.go:49-52`、`:165-167`）。对比 `PUT /instances/nope` 回 400 `{"error":"record not found"}` —— 同一套 API 里两种语义。

**影响**：前端无法区分"成功"与"打了个空枪"；配合 M6 接线后会出现"点删除没报错但也没生效"。

**建议**：`UpdateStatus`/`Delete` 判 `RowsAffected == 0` 回 `gorm.ErrRecordNotFound`，handler 统一映射 404。

### M12. 错误码语义混乱：找不到是 400/404，内部错误也是 400

**现象**：`gorm.ErrRecordNotFound` 全仓无人识别（`grep -rn ErrRecordNotFound` 无命中）。结果是 handler 把 service 返回的一切错误都当同一类。

- `PUT /instances/nope` → **400**（应是 404）
- `UpdateRoute` 找不到 → **400**（`handler/route.go:75`）
- `DeleteRoute` 失败 → **500**（`handler/route.go:88`）
- `StartAdapter` 找不到适配器 → **400**（`handler/adapter.go:101`）
- 全仓**没有一处 409**，但 `api-reference.md` 的"Error codes"表把 409 列为 `Resource conflict`
- 注册重名回 400 且把 `"username already exists"` 原文回吐（`handler/auth.go:21`），409 更贴切

**定位**：`internal/handler/*.go` 的错误分支、`internal/repository/message.go:165-167`、`docs/developer-guide/api-reference.md:799-808`（Error codes 表）。

**影响**：客户端无法据状态码做分支；文档与实现对不上。

**建议**：引入 `errors.Is(err, gorm.ErrRecordNotFound)` 辅助函数统一映射 404；重名/重复走 409；service 层包一层哨兵错误，别把 `record not found` 直接当 400 文案返回。

### M13. 真实功能几乎零测试：`handler`、`service`、`server`、`protocol/v3`、`middleware` 全无测试文件

**现象**：这五个包都没有 `_test.go`（`ls` 计数均为 0）。有测试的只有 `adapter`、`messagefields`、`platformspec`，加上审查期间刚补的 `message`（仅覆盖 shutdown，见 M14）——前者是注册表/字段表类纯数据校验，后者只有连接关闭。

**影响**：B1（鉴权）和 B2（WS 不路由）任何一条都能被一个 httptest 用例抓住，但没有。`CGO_ENABLED=0` 下 `adapter` 包直接失败，所以 CI 得整体置 `CGO_ENABLED=1` 并 `apt-get install gcc`（`.github/workflows/ci.yml:40`、`:50-51`）——测试对 cgo 有硬依赖，这层脆弱性本身也值得收。

**建议**：先用 `httptest` + `gin.New()` 给鉴权补表驱动用例（空头/坏格式/过期/有效四态），再给 WS 补一个"发帧后库里多一行 + 目标端收到"的集成用例。`sqlite` 依赖 cgo，可在 `main_test.go` 里先探测再 `t.Skip`，让 `CGO_ENABLED=0` 也绿。

### M14. ~~`message.Dispatcher` 关闭路径会 `close of closed channel` panic~~（审查中已被修复）

**原现象**：`Close()` 遍历 `close(conn.Send)`；`readPump` 的 defer 在连接断开时又走 `UnregisterConnection` 再 `close` 一次。两个关闭点各自裸调 `close(chan)`，谁后到谁 panic。用临时用例复现过：

```
TestReviewClose        → PASS
TestReviewDoubleClose  → PANIC: close of closed channel
```

**现已修复**：`Connection` 加了 `closeOnce sync.Once` 与 `closeSend()`（`dispatcher.go:34-45`），两个关闭点都改走它；`Close` 末尾清空 map，后续心跳 tick 不会再写到已关闭的连接（`dispatcher.go:335` 附近）。新增 `dispatcher_test.go` 五个用例把两种顺序、重复 unregister、Close 后无残留连接全部钉住，实测 5/5 通过。

**残留建议**：`message` 包此前完全没有 `TestMain`，为让 `logger.Init` 生效才补了一个（`dispatcher_test.go:16-21`）——`logger` 未初始化时是 nil zap，这个坑值得在 `pkg/logger` 里直接兜掉，而不是每个包的测试各写一遍。

### M15. `http.Server` 未设任何超时，无请求体大小限制

**定位**：`internal/server/server.go:50-53` 只设了 `Addr` 和 `Handler`；全仓无 `MaxBytesReader`（`grep` 无命中）。

**影响**：Slowloris 可一直挂着连接；超大 body 全量读进内存。实测 2MB 的 JSON 能过（回 400 是业务校验，不是大小拒绝）。

**建议**：补 `ReadHeaderTimeout` / `ReadTimeout` / `WriteTimeout` / `IdleTimeout`，加 `MaxBytesReader`。

**现已修复**（P1/ESPL-20）：四类超时全部落到 `http.Server`，值来自 `server.*` 配置（默认 10s/60s/120s/120s，0 = 不设）。请求体上限 `server.max_body_bytes`（默认 4 MiB）由全局中间件 `MaxBodyBytes` 执行：声明超限直接 413，不声明（chunked/撒谎的 Content-Length）由 `http.MaxBytesReader` 兜住。实测 8KB body → 413、正常 body → 200；只发一半头后挂住的连接 **2.0s**（配置值）被 `ReadHeaderTimeout` 断开。

### M16. `make migrate` 与 `make dev` 传的参数 `main.go` 根本不读

**现象**：`main.go` 没有任何 `flag.` / `os.Args` 解析（`grep` 无命中）。实测带 `migrate` 参数启动，进程**照常起 HTTP 服务**，不迁移、不退出。

- `Makefile:39` → `go run main.go migrate`
- `Makefile:43` → `go run main.go --dev`

**定位**：`Makefile:38-43`、`main.go:18`。

**影响**：`make migrate` 看起来成功（进程在跑），实际什么也没做，得手动 Ctrl-C。`migrations/README.md:27` 其实已经自首"未实现"。

**建议**：要么加 `flag` 分发子命令，要么把 Makefile 改成先 `build` 再跑二传，并把 `--dev` 去掉或实现。

---

## 3. Minor

### m1. 路由规则的修改进不了内存路由表（接通后也不生效）
`RouteService` 只写库，`message.Router.AddRule`（`router.go:42`）**零调用**，`ListEnabled`（`service/route.go:121`）也无调用方。B2 修好之后，规则仍然只在库里——除非再加一层"启动加载 + 变更同步"。优先级排序是手写冒泡（`router.go:89-95`），规则多时会慢。

### m2. 路由 `priority: 0` 改不动
`UpdateRouteRequest.Priority` 是 `int`，`Update` 用 `if req.Priority != 0` 判空（`service/route.go:80`）。实测建成 `priority:7` 后 PUT `{"priority":0}`，值仍是 7。改成 `*int` 即可。`CreateRouteRequest.Enabled` 同理：传 `false` 建出来的却是 `true`（`service/route.go:53` + 模型默认值），要显式 `enabled` 才能关。

### m3. `Count(&total)` 的 error 被丢弃
六个 `List` 都写 `r.db.Model(...).Count(&total)` 不接返回值（`repository/adapter.go:55`、`instance.go:55`、`message.go:59`、`message.go:121`、`message.go:174`、`platform.go:58`、`user.go:58`）。计数失败会静默返回 `total: 0`。

### m4. API 文档与实现四处不符
- `api-reference.md` 的"Error codes"列了 409，后端一处没有
- 无 `POST /platforms` 文档，实际也 404（M5）
- `api-reference.md:753-798` 的 WS 示例用 `"type":"message.received"`，而 `envelope.go:22` 的 JSON tag 是 `event_type`；文档的"Message format"与代码里的 `Validate`（`protocol.go:59-79`）要求也对不上
- 文档说"Rate limiting 100/min、认证接口 10/min"，**没有任何限流中间件**（`grep rate internal/middleware/` 无命中）

### m5. `v3.MessageEnvelope.Sign` 不可重入
`Sign`（`envelope.go:88`）把整个结构体（含 `Signature` 字段）一起 marshal，二次签名会把旧签名也签进去。实测同一 envelope 连签两次，第二次 `Verify` 返回 `false`。`Verify` 那边 copy 后清了 `Signature`（`envelope.go:106-107`），两边不对称。

### m6. Layout 顶部"已连接"徽章是写死的
`components/Layout.tsx:321-323` 无条件渲染 `app.connected`，不查后端；`Dashboard` 里那个真实的 `healthCheck` 状态只影响本页。

### m7. Dashboard 用 `data.length` 当总数
四个计数取 `platforms.data?.length`（`Dashboard.tsx:64-67`），而接口默认只回 20 条 —— 超过 20 就永远显示 20，尽管响应里带着正确的 `total`。

### m8. `Docs` 页依赖一步手工部署，没人做就会全 404
`vite.config.ts` 的 docs 中间件只挂了 `configureServer`（`:23`），只有 dev server 生效；实测 `dist/` 里确实没有 `docs/`。`web/DOCS.md:6` 已把"拷 docs/ 或加反代"列为部署步骤，但没有任何构建脚本或 CI 检查去做，漏了就全断。

### m9. `index.html:5` 引用的 `/vite.svg` 不存在
无 `web/public/` 目录，`dist/vite.svg` 也没有 —— favicon 恒 404。

### m10. 前端 `Settings` 页不落后端
四个分区全存 `localStorage['app_settings']`（`Settings.tsx:80,96`），后端也没有 `/settings` 接口。"保存成功"只代表写进了浏览器。

### m11. 无用户管理接口
注册后无法改角色、禁用账号、改密码；`UserRepository` 的 `Update`/`Delete`/`List` 全无路由。`AdminRequired` 因此永远空转。`AuditLog` 模型建了表（`repository.go:47`）但**从不写入**。

### m12. 没有一处数据库事务
`grep '\.Transaction(\|Begin()'` 无命中。多步写（如 B5 的"落库+派发+改状态"）目前靠运气，不是原子。

### m13. Redis 只配置不使用
`config` 有完整的 `RedisConfig`、`config.yaml:20-25` 也写了，`grep -rn redis backend --include=*.go` 除 `config.go` 外零命中。README 却把 Redis 列进技术栈。

### m14. 列表接口无上限
`limit` 直接透传 `Limit(limit)`，`?limit=999999` 回 200（实测）。`limit=abc` 被 `strconv.Atoi` 吞成 0 后落到默认值，行为能接受但没报错。

### m15. 端口/健康检查信息暴露
`GET /health`（`server.go:169-174`）无鉴权回 `{"status":"ok","version":"1.0.0"}`，版本号可用于对号入座已知问题。属可接受的设计取舍，但建议生产环境收敛。

### m16. 前端 23 条 `no-explicit-any` 靠 `--max-warnings 23` 顶着
`package.json` 的 lint 脚本把闸门抬到 23（`web/package.json:12`）。规则是只许降（`web/.eslintrc.cjs:18-20`）——**新增一条 any 就会红**，但删掉一条后要人工把 23 调回 22，否则预算永远虚高。`api/client.ts` 通篇 `request<any[]>`，是最该先补类型的地方。

### m17. 前端没有任何测试，`DOCS.md` 却声称有
`web/package.json` 的 `scripts` 只有 `dev`/`build`/`lint`/`preview`，**没有测试脚本、没有 vitest/jest 依赖**，`web/src` 下也没有一个 `*.test.*`/`*.spec.*`。
而 `web/DOCS.md:11` 写着"`docs.link*` 键在两个语言包里必须同步（**web 构建里有一个测试会比对它们**）"——那个测试不存在。两包一旦漂移不会有人发现。

### m18. `node_modules` 被提交进仓库
`git ls-files web/node_modules | wc -l` = **11077**，占全仓 `11170` 个受版本控制文件的 99%。克隆体积和 diff 噪音都很大。仓库根目录没有 `.gitignore`，`web/.gitignore` 只有 `dist` 和 `*.timestamp-*.mjs` 两行（`web/.gitignore:1-2`），没有任何地方忽略 `node_modules`——即使想 ignore 也得先 `git rm -r --cached`。

### m19. ~~`npm run build` 在只读工作树上直接失败，PR-QA 却让人跑它~~（已修复）

`docs/developer-guide/pr-qa.md:100` 把 `npm run build` 列进 PR 自检命令：
`(cd web && npx tsc --noEmit && npm run lint && npm run build)`。

实测在只读树上必挂：

```
$ cd web && npx vite build
failed to load config from .../vite.config.ts
Error: EROFS: read-only file system, open '.../vite.config.ts.timestamp-1794026724323-....mjs'
```

原因是 Vite 加载 TS 配置时会在配置**旁边**写一个 `vite.config.ts.timestamp-*.mjs`，再 import 它。`web/.gitignore:2` 已经把这类文件忽略了（说明已知会生成），但生成动作需要写权限。**整棵树拷到可写目录后 `vite build` 能正常完成**（`✓ 1594 modules transformed`），所以不是构建配置错，是环境前提没写进自检步骤。

**影响**：在只读/只挂载源码的环境（CI 的 `--read-only` 容器、`docker run -v $PWD:/src:ro`、本机这类被管控的工作树）里，PR-QA 第 97 行是照做即红的命令，容易被误判成"构建挂了"。

**建议**：`pr-qa.md` 该行补一句前提（"产物目录须可写；只读挂载请先 `cp -r web /tmp` 或 `vite build --config` 指向可写副本"），或给 `vite build` 加个把临时配置写到 `os.tmpdir()` 的开关。这属文档修补，不阻塞合并。

### m20. ~~Node 版本下限文档不一致~~（审查中已被修复）

**原现象**：`web/package.json:6-8` 加了 `engines.node: "^20.11.0"`，CI 两处都改成 `node-version-file: web/package.json`（`ci.yml:74`、`:142`），工具链统一到 20.11；但 `.github/CONTRIBUTING.md:16` 的开发环境表还写着 `Node | 18+`，同一行末尾却又补了"(`web/package.json` `engines`; CI reads it)"——值和要求自相矛盾。按 `engines`，Node 18 是装不起来的。

**现已修复**：`CONTRIBUTING.md:16` 已改成 `20.11+`，与 `engines`、CI、`docs/user-guide/getting-started.md:10` 四处一致。实测全文再无 `18+` 残留。

**残留建议**：`engines` 是硬约束（`npm ci` 会按它拦），但只有 `CONTRIBUTING` 一张表在维护它——建议在 `web/package.json` 旁边留一句"改 `engines` 要同步 CONTRIBUTING 与 getting-started"，否则下次调版本还会漂。

---

## 4. 明确判定为「设计取舍」而非缺陷的部分

避免误伤，以下都有正当理由，不作为问题上报：

- **`/platforms/spec`、`/platforms/spec/:id`、`/message-fields` 不需登录**：`server.go:75-81` 把它们注册在 `platforms.Use(AuthRequired)` 之前是有意的，注释写明"前端建实例前先看这里，不需要登录"。实测这三个匿名可达（200），`/platforms` 本身匿名 401 —— 行为与注释一致，**gin 的 `Use` 只影响其后注册的路由这一点在这里被正确利用了**，不要"顺手修掉"。
- **`AutoMigrate` 与应用内迁移并存**：`repository.go:37-50` 启动即建表，`migrations/*.sql` 是给 DBA 的手工路径。`migrations/README.md:27` 已注明子命令未实现，属已知欠账（M16 已收）。
- **SQLite 作默认驱动**：开发形态合理，代价是 cgo 强依赖（CI 已装 gcc）。Postgres 分支同在 `initDB` 里，不是死代码。
- **密码 bcrypt、令牌 HS256 + 24h 过期、`PasswordHash` 有 `json:"-"`**：`service/auth.go:65`、`:147`、`models.go:116` 都做对了。登录"用户不存在"与"密码错"回同一句 `invalid credentials`（`auth.go:94`、`:102`）也是正确的防枚举写法。
- **`jwt.expiration` 是小时数**：`config.yaml:33` 的注释写清了，`auth.go:133` 也是按小时乘的，没有单位歧义。
- **`/docs` 产线需要额外部署动作**：`web/DOCS.md:6` 明确写了"生产构建必须把 `docs/` 拷到 `dist/` 旁边或前置反代"——也就是说这是已知的部署动作，不是缺陷；m8 只是提醒别漏了这一步。
- **`golangci-lint` 已启用，`errcheck` 被有意关掉**：`ci.yml:135-139` 把它作为 `format-lint` 的一步跑（`v2.1.6`，配置在 `.golangci.yml`），配置里写明关掉 `errcheck` 的理由（实测 25 处：非测试 22、测试 3，都是 `Close`/`rand.Read`/`logger.Sync` 这类失败无可恢复路径的调用；开在今天只会得到一条开局即红的门）。
- **`--max-warnings 23` 是刻意的债务预算**：`web/.eslintrc.cjs:18-20` 说明了它只许降不许升，不是把报警压掉。

---

## 5. 按优先级排序的修复清单

### P0 — 不做就等于没有产品（建议同一批提交）

1. **B1 鉴权**：`AuthRequired` 接 `AuthService.ValidateToken`，写真实 claims；`server.go` 11 处调用点改传参。给 `middleware` 补四个表驱动用例（空头 / 坏格式 / 过期 / 有效）。
2. **B2 WebSocket 接线**：`Services` 持有 `Hub`；两条 handler 改为向 hub 注册并把读循环交给 `readPump`/`writePump`；key 不再用 query 明文。
3. **B3 消息入口**：新增 `POST /messages`（或 `/ingest`），落库后 `hub.ProcessMessage`。
4. **B4 幂等键**：`IdempotencyKey` 改 `*string`（或落库前生成兜底值），迁移改成 `uniqueIndex` + 非空。**必须与 B3 同批**，否则接上入口即 500。
5. **B5 命令派发**：`CreateCommand` 落库后 `hub.SendCommand` + `MarkSent`，失败走 `MarkFailed`；补 `ListPending` 重试扫描。
6. **M1 归属隔离**：列表/写操作按 `userID` 收窄，越权回 403。

### P1 — 安全与可部署性

7. **M7 API 基址**：改 `import.meta.env.VITE_API_BASE_URL ?? '/api/v1'`，健康检查走相对路径。
8. **M8/M9 登录态**：401 统一 `clearToken` + 跳 `/login`；`handleLogout` 改用 client；加 `RequireAuth` 路由守卫。
9. **M3 JWT secret**：已完成（P1/ESPL-20，启动 WARN）。
10. **M4 WS 鉴权 + CheckOrigin 白名单**：已完成（P1/ESPL-20，升级前 401 / 跨域 403，白名单与 CORS 同一份）。
11. **M2 CORS**：已完成（P1/ESPL-20，回显白名单 Origin + `Vary: Origin`）。

### P2 — 正确性与可观测性

12. **M10/M6**：实例模型补统计字段、实例级启停改走 `InstanceService`；再补前端增删改表单与 `delete*/update*` 客户端方法。
13. **M11/M12 状态码**：`RowsAffected == 0` 回 not-found；统一 `ErrRecordNotFound → 404`、重名 `→ 409`。
14. **M5**：补齐 `POST/PUT/DELETE /platforms`，挂 `AuthRequired` + `AdminRequired`。
15. **M13 测试**：先鉴权、再 WS 端到端；`CGO_ENABLED=0` 下用探测 + `t.Skip` 兜底。
16. **M14**：已完成（`sync.Once` + 五个用例）。剩余只需把 `logger` 未初始化就 nil 的问题在 `pkg/logger` 兜掉，免得每个包写一遍 `TestMain`。
17. **M15**：已完成（P1/ESPL-20，四类超时 + body 上限，超限 413）。
18. **m1/m2**：`Hub` 建好后加载已启用规则并在增删改时同步内存；`Priority`/`Enabled` 改指针类型。

### P3 — 文档与卫生

19. **M16**：实现 `migrate`/`--dev` 或修正 Makefile 与 `migrations/README.md`。
20. **m4**：统一文档与实现（WS 字段名 `event_type`、409 表、限流说明、平台写接口）。
21. **m3/m5/m6/m7/m8/m9/m10/m11/m12/m13/m14/m15** 逐条处理；其中 **m5（签名不可重入）** 建议优先，因为它会在 B2 接通后被真实流量踩到。
22. **m16/m17/m18**：`client.ts` 补类型把 any 清掉、回收 `--max-warnings`；补一个语言包键对齐的测试（把 `DOCS.md` 的声称补实）；`node_modules` 移出版本控制并补 `.gitignore`。
23. **m20**：已完成（`CONTRIBUTING.md:16` 已是 `20.11+`，全文无 `18+` 残留）。建议再补一句“改 `engines` 要同步这两处文档”。
24. **m19**：已完成（`pr-qa.md:97-99` 已注明产物目录须可写；原引用行号 `:97` 现为 `:100`）。
