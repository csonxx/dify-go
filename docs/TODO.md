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

状态：已完成（兼容版）

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
- [x] `/workspaces/current/endpoints/create`
- [x] `/workspaces/current/endpoints/list`
- [x] `/workspaces/current/endpoints/list/plugin`
- [x] `/workspaces/current/endpoints/delete`
- [x] `/workspaces/current/endpoints/update`
- [x] `/workspaces/current/endpoints/enable`
- [x] `/workspaces/current/endpoints/disable`
- [x] `/workspaces/current/triggers`
- [x] `/workspaces/current/trigger-provider/{provider}/icon`
- [x] `/workspaces/current/trigger-provider/{provider}/info`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/list`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/builder/create`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/builder/{subscriptionBuilderId}`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/builder/update/{subscriptionBuilderId}`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/builder/verify-and-update/{subscriptionBuilderId}`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/builder/logs/{subscriptionBuilderId}`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/builder/build/{subscriptionBuilderId}`
- [x] `/workspaces/current/trigger-provider/{subscriptionId}/subscriptions/update`
- [x] `/workspaces/current/trigger-provider/{subscriptionId}/subscriptions/delete`
- [x] `/workspaces/current/trigger-provider/{provider}/oauth/client`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/oauth/authorize`
- [x] `/workspaces/current/trigger-provider/{provider}/subscriptions/verify/{subscriptionId}`

本阶段剩余重点：

- [ ] 更完整的 built-in/provider catalog 与真实 MCP/trigger/plugin 语义对齐
- [ ] 更贴近真实插件运行时的 endpoint / trigger 执行语义、回调安全校验和 provider-specific 行为

## 阶段 3：插件平台

状态：进行中（基础兼容版已落地）

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

本轮已完成的子范围：

- [x] `/workspaces/current/plugin/debugging-key`
- [x] `/workspaces/current/plugin/list`
- [x] `/workspaces/current/plugin/list/latest-versions`
- [x] `/workspaces/current/plugin/list/installations/ids`
- [x] `/workspaces/current/plugin/icon`
- [x] `/workspaces/current/plugin/asset`
- [x] `/workspaces/current/plugin/upload/pkg`
- [x] `/workspaces/current/plugin/upload/github`
- [x] `/workspaces/current/plugin/upload/bundle`
- [x] `/workspaces/current/plugin/install/pkg`
- [x] `/workspaces/current/plugin/install/github`
- [x] `/workspaces/current/plugin/install/marketplace`
- [x] `/workspaces/current/plugin/marketplace/pkg`
- [x] `/workspaces/current/plugin/fetch-manifest`
- [x] `/workspaces/current/plugin/tasks`
- [x] `/workspaces/current/plugin/tasks/{task_id}`
- [x] `/workspaces/current/plugin/tasks/{task_id}/delete`
- [x] `/workspaces/current/plugin/tasks/delete_all`
- [x] `/workspaces/current/plugin/tasks/{task_id}/delete/{identifier}`
- [x] `/workspaces/current/plugin/upgrade/marketplace`
- [x] `/workspaces/current/plugin/upgrade/github`
- [x] `/workspaces/current/plugin/uninstall`
- [x] `/workspaces/current/plugin/permission/change`
- [x] `/workspaces/current/plugin/permission/fetch`
- [x] `/workspaces/current/plugin/parameters/dynamic-options`
- [x] `/workspaces/current/plugin/parameters/dynamic-options-with-credentials`
- [x] `/workspaces/current/plugin/preferences/change`
- [x] `/workspaces/current/plugin/preferences/fetch`
- [x] `/workspaces/current/plugin/preferences/autoupgrade/exclude`
- [x] `/workspaces/current/plugin/readme`
- [x] `/apps/imports/{appId}/check-dependencies`
- [x] `/rag/pipelines/imports/{pipelineId}/check-dependencies`

本阶段剩余重点：

- [ ] 接入真实 plugin daemon / marketplace 元数据，而不是当前兼容版的本地推导 manifest
- [ ] 继续把 upload/install/upgrade 语义推进到真实包解析、bundle 依赖拆解和失败回滚
  当前已经支持从 bundle 压缩包内的 JSON/YAML 依赖声明做兼容解析，并且 app/pipeline dependency check 会从现有 app model config 与 workflow graph 中提取插件依赖
- [ ] 把 dynamic options、权限校验、README/asset/icon 等接口从兼容占位继续收敛到真实插件运行时行为

## 阶段 4：知识库与 Dataset 主链路

状态：进行中（主链路与 metadata/segment 第二批已落地）

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

本轮已完成的子范围：

- [x] `/datasets`
- [x] `/datasets/init`
- [x] `/datasets/retrieval-setting`
- [x] `/datasets/process-rule`
- [x] `/datasets/indexing-estimate`
- [x] `/datasets/api-base-info`
- [x] `/datasets/api-keys`
- [x] `/datasets/{datasetId}`
- [x] `/datasets/{datasetId}/use-check`
- [x] `/datasets/{datasetId}/related-apps`
- [x] `/datasets/{datasetId}/api-keys/enable`
- [x] `/datasets/{datasetId}/api-keys/disable`
- [x] `/datasets/{datasetId}/documents`
- [x] `/datasets/{datasetId}/documents/metadata`
- [x] `/datasets/{datasetId}/documents/status/{action}/batch`
- [x] `/datasets/{datasetId}/documents/{documentId}`
- [x] `/datasets/{datasetId}/documents/{documentId}/metadata`
- [x] `/datasets/{datasetId}/documents/{documentId}/indexing-status`
- [x] `/datasets/{datasetId}/documents/{documentId}/rename`
- [x] `/datasets/{datasetId}/documents/{documentId}/processing/pause`
- [x] `/datasets/{datasetId}/documents/{documentId}/processing/resume`
- [x] `/datasets/{datasetId}/documents/{documentId}/segments`
- [x] `/datasets/{datasetId}/documents/{documentId}/segment`
- [x] `/datasets/{datasetId}/documents/{documentId}/segment/enable`
- [x] `/datasets/{datasetId}/documents/{documentId}/segment/disable`
- [x] `/datasets/{datasetId}/documents/{documentId}/segments/{segmentId}`
- [x] `/datasets/{datasetId}/documents/{documentId}/segments/{segmentId}/child_chunks`
- [x] `/datasets/{datasetId}/documents/{documentId}/segments/{segmentId}/child_chunks/{childChunkId}`
- [x] `/datasets/{datasetId}/documents/{documentId}/segments/batch_import`
- [x] `/datasets/{datasetId}/batch/{batchId}/indexing-status`
- [x] `/datasets/batch_import_status/{jobId}`
- [x] `/datasets/{datasetId}/auto-disable-logs`
- [x] `/datasets/metadata/built-in`
- [x] `/datasets/{datasetId}/metadata`
- [x] `/datasets/{datasetId}/metadata/{metadataId}`
- [x] `/datasets/{datasetId}/metadata/built-in/enable`
- [x] `/datasets/{datasetId}/metadata/built-in/disable`
- [x] `/datasets/{datasetId}/queries`
- [x] `/datasets/{datasetId}/error-docs`
- [x] `/datasets/{datasetId}/hit-testing`
- [x] `/datasets/{datasetId}/external-hit-testing`
- [x] `/datasets/{datasetId}/retry`
- [x] `/datasets/external`
- [x] `/datasets/external-knowledge-api*`
- [x] `/datasets/{datasetId}/documents/{documentId}/download`
- [x] `/datasets/{datasetId}/documents/download-zip`
- [x] `/datasets/{datasetId}/documents/{documentId}/pipeline-execution-log`
- [x] `/datasets/{datasetId}/documents/{documentId}/notion/sync`
- [x] `/datasets/{datasetId}/documents/{documentId}/website-sync`
- [x] `/console/api/files/upload`
- [x] `/console/api/files/support-type`
- [x] `/console/api/remote-files/upload`
- [x] `/console/api/files/{fileID}/preview`
- [x] `/files/{fileID}/file-preview`
- [x] `/files/{fileID}/image-preview`
- [x] `/api/files/upload`
- [x] `/api/files/support-type`
- [x] `/api/remote-files/upload`

本阶段剩余重点：

- [ ] 把批量导入、外部知识库命中链路继续从兼容壳推进到更贴近上游的真实语义
  当前 `external-hit-testing` 已经按上游契约请求外部 `endpoint/retrieval`，会透传 `knowledge_id`、`top_k`、`score_threshold`，并补上 query 校验、HTTP 集成测试，以及 Bedrock 风格 `retrievalResults / content.text / location.*` 响应解析
- [ ] 收敛 dataset service API、索引状态流转、命中测试记录与后续 pipeline 之间的共享模型
  当前已经支持 console/public 本地上传、`remote-files/upload`、hit-testing 记录里的 `attachment_ids -> image_query -> file_info/source_url` 回写链路，以及 dataset service API 的动态 base URL、workspace API key CRUD、dataset enable/disable 状态回写
- [ ] 继续压缩知识库详情页剩余 fallback，优先补 provider-specific external retrieval 和 remote file 行为
  当前 external retrieval response parser 已经支持标准 `records`、`retrievalResults`、`retrieval_results`、`results` 和 `data` 包装，并把 AWS Bedrock location uri/url 合并进 metadata/title fallback
- [ ] 为 dataset metadata / segments / child chunks 增加更系统的集成测试覆盖
  当前已经补上首批 Go HTTP 回归测试，覆盖 uploads、metadata field 更新、segment / child chunk 生命周期、batch import、hit-testing attachment query 记录，以及 external-hit-testing 的外部 API 契约

## 阶段 5：RAG Pipeline

状态：进行中（空白 dataset + workflow alias + DSL import/export + template/customized template + datasource catalog + datasource auth + published run 第五批已落地）

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

本轮已完成的子范围：

- [x] `/rag/pipeline/empty-dataset`
- [x] `/rag/pipeline/dataset`
- [x] `/rag/pipeline/templates`
- [x] `/rag/pipeline/templates/{templateId}`
- [x] `/rag/pipeline/customized/templates/{templateId}`
- [x] `/rag/pipelines/datasource-plugins`
- [x] `/auth/plugin/datasource/list`
- [x] `/auth/plugin/datasource/default-list`
- [x] `/auth/plugin/datasource/{pluginId}/{provider}`
- [x] `/auth/plugin/datasource/{pluginId}/{provider}/update`
- [x] `/auth/plugin/datasource/{pluginId}/{provider}/delete`
- [x] `/auth/plugin/datasource/{pluginId}/{provider}/default`
- [x] `/auth/plugin/datasource/{pluginId}/{provider}/custom-client`
- [x] `/oauth/plugin/{pluginId}/{provider}/datasource/get-authorization-url`
- [x] `/oauth/plugin/{pluginId}/{provider}/datasource/callback`
- [x] `/rag/pipelines/imports`
- [x] `/rag/pipelines/imports/{importId}/confirm`
- [x] `/rag/pipelines/{pipelineId}/exports`
- [x] `/rag/pipelines/{pipelineId}/customized/publish`
- [x] `/rag/pipelines/{pipelineId}/workflows/draft`
- [x] `/rag/pipelines/{pipelineId}/workflows/publish`
- [x] `/rag/pipelines/{pipelineId}/workflows`
- [x] `/rag/pipelines/{pipelineId}/workflow-runs`
- [x] `/rag/pipelines/{pipelineId}/workflow-runs/tasks/{taskId}/stop`
- [x] `/rag/pipelines/{pipelineId}/workflows/draft/pre-processing/parameters`
- [x] `/rag/pipelines/{pipelineId}/workflows/published/pre-processing/parameters`
- [x] `/rag/pipelines/{pipelineId}/workflows/draft/processing/parameters`
- [x] `/rag/pipelines/{pipelineId}/workflows/published/processing/parameters`
- [x] `/rag/pipelines/{pipelineId}/workflows/draft/datasource/variables-inspect`
- [x] `/rag/pipelines/{pipelineId}/workflows/draft/datasource/nodes/{nodeId}/run`
- [x] `/rag/pipelines/{pipelineId}/workflows/published/datasource/nodes/{nodeId}/run`
- [x] `/rag/pipelines/{pipelineId}/workflows/published/run`
- [x] `/datasets/{datasetId}/documents/{documentId}/pipeline-execution-log`

本阶段剩余重点：

- [ ] 继续把 published run 从当前兼容执行面推进到更贴近上游的真实 pipeline transform / batch job 执行语义
  当前已经先把 published preview / dataset indexing estimate 的预览结果收敛到同一套 Go builder：会按 `doc_form` 返回 `text_model / hierarchical_model / qa_model` 三种结构化 payload，并补上 `chunk_structure / parent_mode / qa_preview`
- [x] 把 datasource plugin 列表从当前空兼容响应推进到 Go 侧可直接驱动前端的内置 datasource catalog
- [x] 把 datasource auth 的列表、凭证 CRUD、默认项切换、自定义 OAuth client、授权回跳兼容链路迁到 Go
- [x] 把 datasource plugin / credential 发现从当前内置 catalog + workspace state 推进到 workspace plugin 安装态优先的 provider 发现语义，并保留对既有 credential 状态的兼容回退
- [ ] 继续收敛 pipeline 与 dataset 之间的共享状态，让空白 dataset、publish 状态、execution log、文档处理流程完全共用 Go 模型
  当前 draft workflow 保存、publish，以及 workflow version restore 之后，knowledge-index 里的 `chunk_structure / indexing_technique / retrieval_model / embedding_model / summary_index_setting` 已经会自动回写到 linked dataset，pipeline editor 与 dataset 详情页的知识库配置不再各走各的

补充：

- 现在 `.pipeline` / YAML DSL 已经可以在 Go 侧完成导出、导入、以及“从 DSL 创建 dataset”。
- 导入时会把 `workflow.graph/features/environment_variables/conversation_variables/rag_pipeline_variables` 同步到 Go 的 workflow draft。
- 如果 DSL 的 `knowledge-index` 节点里带了 `chunk_structure`、`indexing_technique`、`retrieval_model`、`embedding_model(_provider)`、`summary_index_setting`，会同步回写到 dataset 状态，确保前端 dataset/pipeline 面板看到的是同一份配置。
- Template 目录现在也已经落到 Go：内置 built-in 模板列表/详情由 Go 直接提供，customized template 支持发布、列表、详情、更新元信息、导出和删除。
- Published run 现在已经支持 preview、首次创建文档、以及基于 `original_document_id` 的重处理，并且会把 `datasource_type / datasource_info / input_data / datasource_node_id` 落到 dataset document 的 pipeline execution log，前端 create-from-pipeline 和 document settings 都可以直接复用这条 Go 链路。
- Published run 的在线 datasource 现在会复用 datasource catalog/auth 的 workspace 校验：`online_document / website_crawl / online_drive` 必须是当前 workspace 可见 provider，并且每个 datasource item 都要携带仍然有效的 `credential_id`，避免数据源选择页和正式创建文档时的授权状态不一致。
- dataset `/datasets/indexing-estimate` 和 RAG pipeline `published/run` 的 preview 输出现在也开始共用一套按 `doc_form` 生成的 Go 预览语义：`text_model` 返回普通 `preview`，`hierarchical_model` 返回带 `child_chunks` 和 `parent_mode` 的 parent-child 结构，`qa_model` 返回前端可直接渲染的 `qa_preview`，这样原样搬过来的 preview 面板不需要额外前端兼容代码。
- Published run 写入的 `/rag/pipelines/{pipelineId}/workflow-runs` 历史、详情和 `/node-executions` tracing 现在也已经带上 pipeline 语义化上下文：会持久化 `pipeline_id / dataset_id / datasource_type / datasource_info_list / start_node_id / processing_inputs / batch / document_ids / original_document_id / preview_result`，前端 run history、run detail、trace panel 不再只看到通用 workflow 占位数据。
- console workflow stop 现在也开始有真实语义了：`/apps/{appId}/workflow-runs/tasks/{taskId}/stop` 和 `/rag/pipelines/{pipelineId}/workflow-runs/tasks/{taskId}/stop` 会把已持久化的 run 与 node execution 改成 `stopped`，并且在 run history/detail 一起暴露 `task_id`；如果这是 RAG pipeline 的 create/reprocess run，还会把 linked dataset document/batch 一并落到 `paused + stopped_at`，避免 processing 页和 run trace 各说各话。
- Published run 创建/重处理出来的 dataset document 现在不再直接落成 `completed`，而是会先进入 `waiting`，再通过 Go 侧统一的 document processing 状态推进器在 `document detail / document list / document indexing-status / batch indexing-status` 这些读路径上推进到 `parsing / cleaning / splitting / indexing / completed`；这样 create-from-pipeline 的 processing 页面、文档详情页和普通知识库文档列表看到的是同一条状态流。
- datasource node run 现在也已经切到 Go，补上了 draft/published 两套 `/datasource/nodes/{nodeId}/run` SSE 兼容接口，当前覆盖 `online_document / website_crawl / online_drive` 三类节点，create-from-pipeline 的在线数据源选择页不再需要走 Python fallback。
- draft datasource `variables-inspect` 现在也已经接入 Go：前端在 create-from-pipeline 里单跑 datasource 节点时，会得到 `NodeRunResult`，Go 同时把 datasource outputs 保存成该节点 last-run，并让 `/workflows/draft/nodes/{nodeId}/variables` 从 last-run outputs 生成变量检查项，数据源选择、节点 last run 和变量面板不再割裂。
- `/rag/pipelines/datasource-plugins` 现在会按 workspace plugin 安装态返回 datasource catalog：`local_file` 始终内建可用，其它 `online_document / website_crawl / online_drive` provider 只有在工作区已安装对应 plugin，或工作区里已经存在旧的 datasource credential / OAuth client 状态时才会继续暴露；这样前端 block selector、plugin install/uninstall、既有授权状态可以共用一套发现语义。
- datasource auth 相关的 `list / default-list / credential CRUD / default / custom-client / oauth authorization-url / oauth callback` 也已经切到 Go，当前除了把 datasource 凭证和自定义 OAuth client 配置保存在 workspace 本地状态里，还会与 workspace plugin 安装态联动；provider 卸载后如果没有遗留 credential 状态会直接从 auth 列表消失，如果仍有既有 credential 则会以 `is_installed=false` 的兼容姿态继续可见，避免把历史工作区状态直接“藏掉”。
- RAG pipeline dataset 与底层 workflow app 的名称、描述、图标元数据现在也已经开始共用一份 Go 状态：`PATCH /datasets/{id}` 会同步回写 `/apps/{pipelineId}`，`PUT /apps/{pipelineId}` 也会反向更新 dataset 的 `name / description / icon_info`，避免 pipeline 编辑器、dataset 详情、app detail 看到不同步的元数据。
- 这条 app -> dataset 元数据同步链现在也扩到了 `/apps/{pipelineId}/site`：如果用户从 app overview / site settings 改了标题、描述、图标，linked dataset 也会同步更新，避免 app site 配置页和 dataset detail 再次漂移。
- RAG pipeline draft graph 与 linked dataset 的知识库配置现在也开始共用一条 Go 同步链：平时在 pipeline editor 里保存 draft、正式 publish、或者从历史 workflow version restore 回当前 draft 时，`knowledge-index` 节点里的 `chunk_structure / indexing_technique / retrieval_model / embedding_model / summary_index_setting` 都会自动回写到 dataset 状态，避免 dataset settings / processing form 继续显示旧配置。
- RAG pipeline 的发布与 app 生命周期状态也继续往共享模型收敛了：`POST /workflows/publish` 现在会把 linked dataset 的 `is_published` 持久化到 Go 状态；通过 `/apps/{appId}/copy` 复制 pipeline app 时，会同时生成一个新的 linked pipeline dataset；而 `/apps/{appId}` 删除 pipeline app 时，也会一起回收对应 dataset，避免留下孤儿 `pipeline_id`。

## 阶段 6：公共运行时 API

状态：已完成

范围：

- WebApp `/webapp/access-mode`、`/site`、`/meta`、`/parameters`
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

本轮已完成的子范围：

- [x] `/api/webapp/access-mode`
- [x] `/api/passport`
- [x] `/api/site`
- [x] `/api/parameters`
- [x] `/api/meta`
- [x] `/api/chat-messages`
- [x] `/api/chat-messages/{taskId}/stop`
- [x] `/api/completion-messages`
- [x] `/api/messages`
- [x] `/api/conversations*`
- [x] `/api/messages/{messageId}/feedbacks`
- [x] `/api/messages/{messageId}/suggested-questions`
- [x] `/api/messages/{messageId}/more-like-this`
- [x] `/api/saved-messages*`
- [x] `/api/audio-to-text`
- [x] `/api/text-to-audio`
- [x] `/api/workflows/run`
- [x] `/api/workflows/tasks/{taskId}/stop`
- [x] `/api/login/status` 的 public app `app_logged_in` 识别
- [x] public webapp 会话态从单纯 bootstrap token 收敛到 `X-App-Passport` / `dify_go_public_session` 的 app session 语义，公开会话、消息历史、收藏与反馈不再跨访客串线

## 阶段 7：账号、工作区与平台集成

状态：进行中（成员管理、邀请激活、基础 account features 已落地）

范围：

- Workspace members 与 ownership transfer
- Account init、integrates、education
- Email register 与 forgot-password
- OAuth provider 管理
- SSO 登录入口
- 通用 account OAuth / SSO 辅助接口
- Compliance 与 change-email 流程

为什么第七批做：

- 它们很重要，但不在当前迁移的最热路径上。
- 更适合等核心产品面可用后再统一补齐。

完成标准：

- 工作区成员与账号管理页面不再依赖 Python。
- 登录周边流程和 provider auth 能从 Go 提供。

本阶段已完成：

- [x] `GET /console/api/features`
- [x] `GET /console/api/account/integrates`
- [x] `GET /console/api/account/education`
- [x] `POST /console/api/account/init`
- [x] `POST /console/api/email-register/send-email`
- [x] `POST /console/api/email-register/validity`
- [x] `POST /console/api/email-register`
- [x] `POST /console/api/forgot-password`
- [x] `POST /console/api/forgot-password/validity`
- [x] `POST /console/api/forgot-password/resets`
- [x] `POST /api/forgot-password`
- [x] `POST /api/forgot-password/validity`
- [x] `POST /api/forgot-password/resets`
- [x] `POST /console/api/account/change-email`
- [x] `POST /console/api/account/change-email/validity`
- [x] `POST /console/api/account/change-email/reset`
- [x] `POST /console/api/account/change-email/check-email-unique`
- [x] `POST /console/api/email-code-login`
- [x] `POST /console/api/email-code-login/validity`
- [x] `POST /api/email-code-login`
- [x] `POST /api/email-code-login/validity`
- [x] `GET /console/api/account/education/verify`
- [x] `POST /console/api/account/education`
- [x] `GET /console/api/account/education/autocomplete`
- [x] `POST /console/api/oauth/provider`
- [x] `POST /console/api/oauth/provider/authorize`
- [x] `GET /console/api/workspaces/current/members`
- [x] `POST /console/api/workspaces/current/members/invite-email`
- [x] `PUT /console/api/workspaces/current/members/{memberId}/update-role`
- [x] `DELETE /console/api/workspaces/current/members/{memberId}`
- [x] `POST /console/api/workspaces/current/members/send-owner-transfer-confirm-email`
- [x] `POST /console/api/workspaces/current/members/owner-transfer-check`
- [x] `POST /console/api/workspaces/current/members/{memberId}/owner-transfer`
- [x] `GET /console/api/activate/check`
- [x] `POST /console/api/activate`
- [x] `GET /console/api/enterprise/sso/{protocol}/login`
- [x] `GET /api/enterprise/sso/{protocol}/login`
- [x] `GET /api/enterprise/sso/members/{protocol}/login`
- [x] Auth flow token 状态持久化，覆盖 register / forgot-password / change-email / email-code-login
- [x] Ownership transfer 验证 token 状态持久化

本阶段剩余重点：

- [ ] 接入真实外部 IdP 的 SSO metadata、callback、token exchange 与 enterprise webapp auth 策略
- [ ] 把 register / forgot-password / change-email / ownership transfer 从兼容验证码语义继续推进到真实邮件投递链路
- [ ] 把账号 flow / session 从单进程文件状态继续推进到共享存储或数据库状态

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

下一轮从公共运行时转回“剩余业务域的真实语义收敛”。

推荐顺序：

1. 继续推进阶段 5，把 RAG pipeline published run / batch execution / execution log 做得更贴近上游真实行为
2. 进入阶段 7，补 workspace members、ownership transfer、email register / forgot-password 与 OAuth / SSO
3. 进入阶段 8，把 session 与高频状态写入从文件态继续往更稳的持久化方案演进
