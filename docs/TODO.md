# Dify Go TODO

这个文档是 `dify-go` 的执行型待办清单。

目标不是从零重做一套 Dify，而是在尽量不动上游前端的前提下，把后端能力按业务域逐步迁移到 Go，并且在每一轮迁移后都能保持系统可运行、可验证、可继续推进。

本仓库基于并致敬 [langgenius/dify](https://github.com/langgenius/dify)。

## 总原则

1. 前端优先保持不动，除非兼容性问题无法绕开。
2. 优先兼容 Dify 现有的 API 前缀、字段名和返回结构。
3. 以业务域为单位迁移，而不是零散地补单个接口。
4. 在某个业务域迁完之前，保留 `DIFY_GO_LEGACY_API_BASE_URL` 作为 fallback。
5. 每一阶段结束时都要完成 `go build ./...`、冒烟验证、commit、push。
6. 先追求“前端可用 + 行为稳定”，再做内部结构优化。

## 已完成

- [x] 初始化、登录、刷新令牌、退出登录、账号基础信息、工作区基础接口
- [x] 应用 CRUD、应用导入导出、API Key、站点/API 开关、Tracing、Model Config
- [x] Workflow Draft 编辑、发布、版本历史、运行历史、SSE 运行模拟、节点运行辅助接口
- [x] Workspace 级 Model Provider、默认模型、Provider/Model 凭证、模型启停、参数规则
- [x] 路由清单和迁移状态文档

## 执行顺序

剩余工作按下面顺序推进。

## 阶段 1：应用运营与日志

状态：已完成

范围：

- `/apps/{appId}/annotations/count`
- `/apps/{appId}/chat-conversations`
- `/apps/{appId}/chat-conversations/{conversationId}`
- `/apps/{appId}/completion-conversations`
- `/apps/{appId}/completion-conversations/{conversationId}`
- `/apps/{appId}/workflow-app-logs`
- `/workflow/{workflowRunId}/pause-details`
- `/apps/{appId}/server`
- `/apps/{appId}/server/refresh`

为什么先做：

- 这批接口和已经迁完的 app/workflow 逻辑最接近。
- 可以直接解锁控制台里的日志、会话查看、暂停排查、MCP Server 管理。
- 复用现有 app/workflow 状态模型最多，新增复杂度最低。

完成标准：

- 应用日志和会话页面不再依赖 Python fallback。
- Workflow pause details 能从前端正常查看。
- MCP server 的创建、更新、刷新在已迁 app 上可用。

本阶段额外补齐了日志详情页实际依赖的兼容接口：

- `/apps/{appId}/chat-messages`
- `/apps/{appId}/feedbacks`
- `/apps/{appId}/annotations/{annotationId}`

## 阶段 2：工作区扩展能力

状态：进行中

范围：

- `/workspaces/current/tool-providers`
- `/workspaces/current/tools/builtin`
- `/workspaces/current/tools/api`
- `/workspaces/current/tools/workflow`
- `/workspaces/current/tools/mcp`
- `/workspaces/current/tool-provider/builtin/*`
- `/workspaces/current/tool-provider/api/*`
- `/workspaces/current/tool-provider/workflow/*`
- `/workspaces/current/tool-provider/mcp*`
- `/workspaces/current/agent-providers`
- `/workspaces/current/agent-provider/{agentProvider}`
- `/workspaces/current/endpoints/*`
- `/workspaces/current/trigger-provider/*`

为什么第二批做：

- Tools、Endpoints、Triggers 本质上都属于工作区级扩展系统。
- 可以共用一套状态设计和兼容策略。
- 一起迁能避免重复造轮子。

完成标准：

- 工具选择和 provider 管理面板不再依赖 Python。
- 插件 endpoint 管理可用。
- 初始支持的 trigger provider 可以完成配置和订阅流程。

本轮已完成的子范围：

- [x] `/workspaces/current/tool-providers`
- [x] `/workspaces/current/tools/builtin`
- [x] `/workspaces/current/tools/api`
- [x] `/workspaces/current/tools/workflow`
- [x] `/workspaces/current/tools/mcp`
- [x] `/workspaces/current/tool-provider/builtin/{provider}/tools`
- [x] `/workspaces/current/tool-provider/builtin/{provider}/credentials_schema`
- [x] `/workspaces/current/tool-provider/builtin/{provider}/credentials`
- [x] `/workspaces/current/tool-provider/builtin/{provider}/update`
- [x] `/workspaces/current/tool-provider/builtin/{provider}/delete`
- [x] `/workspaces/current/tool-provider/api/add`
- [x] `/workspaces/current/tool-provider/api/remote`
- [x] `/workspaces/current/tool-provider/api/tools`
- [x] `/workspaces/current/tool-provider/api/update`
- [x] `/workspaces/current/tool-provider/api/delete`
- [x] `/workspaces/current/tool-provider/api/get`
- [x] `/workspaces/current/tool-provider/api/schema`
- [x] `/workspaces/current/tool-provider/api/test/pre`
- [x] `/workspaces/current/tool-provider/workflow/create`
- [x] `/workspaces/current/tool-provider/workflow/update`
- [x] `/workspaces/current/tool-provider/workflow/delete`
- [x] `/workspaces/current/tool-provider/workflow/get`
- [x] `/workspaces/current/tool-provider/workflow/tools`
- [x] `/workspaces/current/tool-provider/mcp`
- [x] `/workspaces/current/tool-provider/mcp/auth`
- [x] `/workspaces/current/tool-provider/mcp/tools/{providerId}`
- [x] `/workspaces/current/tool-provider/mcp/update/{providerId}`
- [x] `/workspaces/current/agent-providers`
- [x] `/workspaces/current/agent-provider/{agentProvider}`

本阶段剩余重点：

- [ ] `/workspaces/current/endpoints/*`
- [ ] `/workspaces/current/trigger-provider/*`
- [ ] 更完整的 built-in/provider catalog 与真实 MCP/trigger/plugin 语义对齐

## 阶段 3：插件平台

状态：待实现

范围：

- `/workspaces/current/plugin/install/*`
- `/workspaces/current/plugin/upgrade/*`
- `/workspaces/current/plugin/uninstall`
- `/workspaces/current/plugin/tasks*`
- `/workspaces/current/plugin/preferences/*`
- `/workspaces/current/plugin/readme`
- `/workspaces/current/plugin/asset`
- `/workspaces/current/plugin/debugging-key`
- `/workspaces/current/plugin/marketplace/pkg`
- `/apps/imports/{appId}/check-dependencies`
- `/rag/pipelines/imports/{pipelineId}/check-dependencies`

为什么第三批做：

- 插件平台会依赖前一阶段的工作区扩展基础设施。
- 这一批迁完后，工作区层面的 fallback 流量会下降很多。

完成标准：

- 插件安装、升级、卸载不再依赖 Python。
- 插件任务轮询、任务删除、偏好配置可用。
- app/pipeline 导入依赖检查接口可用。

## 阶段 4：知识库与 Dataset 主链路

状态：待实现

范围：

- Dataset CRUD
- Dataset 设置与 retrieval 设置
- 文档上传、导入、索引状态
- 文档重命名、暂停、恢复、下载、重试
- Metadata CRUD 与 built-in metadata 开关
- Segment CRUD 与 child chunk 管理
- Error docs、queries、related apps、hit testing
- Dataset API keys
- External API knowledge 与 external knowledge base

代表路由：

- `/datasets`
- `/datasets/{datasetId}`
- `/datasets/{datasetId}/documents*`
- `/datasets/{datasetId}/metadata*`
- `/datasets/{datasetId}/queries`
- `/datasets/{datasetId}/error-docs`
- `/datasets/{datasetId}/hit-testing`
- `/datasets/{datasetId}/external-hit-testing`
- `/datasets/external`
- `/datasets/external-knowledge-api*`

为什么第四批做：

- 这是剩余体量最大的核心业务域之一。
- 它应该有自己完整的状态模型，不适合和插件或工具系统混做。
- 后续 RAG pipeline 很多能力也依赖 datasets 先落地。

完成标准：

- 知识库页面能在 Go 后端上完成创建、查看、管理。
- 文档导入和索引状态由 Go 持久化。
- Segments、metadata、hit testing 等能力在前端可用。

## 阶段 5：RAG Pipeline

状态：待实现

范围：

- Pipeline template 列表和详情
- Customized template 更新、删除、导出
- Pipeline 导入与确认
- Draft/Published pre-processing 参数
- Draft/Published processing 参数
- Datasource plugin 发现
- Publish 与 published run
- Pipeline execution log
- Publish as customized pipeline

代表路由：

- `/rag/pipeline/templates*`
- `/rag/pipeline/customized/templates*`
- `/rag/pipelines/imports*`
- `/rag/pipelines/{pipelineId}/workflows/*`
- `/rag/pipelines/datasource-plugins`
- `/datasets/{datasetId}/documents/{documentId}/pipeline-execution-log`

为什么第五批做：

- 这块同时依赖 workflow runtime 和 dataset 状态。
- 当前 Go 里已经有 workflow 基础，可以复用很多逻辑。

完成标准：

- RAG pipeline 的编辑、发布、运行主链路不再依赖 Python。
- 模板导入导出、执行日志、参数面板都能在 Go 侧提供。

## 阶段 6：公共运行时 API

状态：待实现

范围：

- WebApp `/site`、`/meta`、`/parameters`
- `/messages`
- `/conversations*`
- `/chat-messages*`
- `/completion-messages*`
- `/workflows/run`
- `/workflows/tasks/{taskId}/stop`
- `/audio-to-text`
- `/text-to-audio`
- `/saved-messages*`
- `/passport`

为什么第六批做：

- 这批接口面向真实终端用户访问路径。
- 适合在控制台侧主要作者能力稳定后集中处理。

完成标准：

- Public share/webapp 体验能通过 Go 后端执行 chat、completion、workflow。
- 会话历史、重命名、pin/unpin、saved messages 可用。

## 阶段 7：账号、工作区与平台集成

状态：待实现

范围：

- Workspace members 与 ownership transfer
- Account init、integrates、education
- Email register 与 forgot-password
- OAuth provider 管理
- SSO 登录入口
- Datasource auth 辅助接口
- Compliance 与 change-email 流程

为什么第七批做：

- 它们很重要，但不在当前迁移的最热路径上。
- 更适合等核心产品面可用后再统一补齐。

完成标准：

- 工作区成员与账号管理页面不再依赖 Python。
- 登录周边流程和 provider auth 能从 Go 提供。

## 阶段 8：工程化加固与移除 Legacy

状态：待实现

范围：

- 把高频写入域从文件状态逐步演进到更稳的持久化方案
- Session 持久化或迁移到共享存储
- 为已迁路由组增加集成测试
- 基于 `docs/route-manifest.json` 做覆盖率追踪
- 缩小并最终移除 `DIFY_GO_LEGACY_API_BASE_URL`
- 对高频状态变更做性能分析

完成标准：

- 已迁能力可测、可追踪、可稳定运行。
- fallback 面足够小且清晰可见。
- 对已支持能力集可以不依赖 legacy backend 运行。

## 每阶段的标准执行模板

每一轮都按这个节奏走：

1. 盘点目标业务域在 `web/service` 里的前端调用面。
2. 设计这一域最小可用的状态模型。
3. 先做兼容优先的 Go 路由实现。
4. 跑 `go build ./...` 和定向 `curl`/前端冒烟验证。
5. 更新 `docs/GO_MIGRATION.md` 和本 TODO 文档。
6. commit 并 push。

## 下一步

下一轮直接做“阶段 1：应用运营与日志”。

推荐顺序：

1. `annotations/count`
2. `chat-conversations` 与 `completion-conversations`
3. `workflow-app-logs`
4. `workflow/{id}/pause-details`
5. `apps/{id}/server` 与 `apps/{id}/server/refresh`
