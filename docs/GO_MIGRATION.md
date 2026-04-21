# Dify Go Migration

`dify-go` now contains:

- The original Dify frontend workspace copied from `../dify` without app-level changes.
- A new Go backend compatibility layer under `cmd/` and `internal/`.
- A generated Python route inventory at [route-manifest.json](/Users/tt/goworkspace/src/dify-go/docs/route-manifest.json).
- A step-by-step migration backlog at [TODO.md](/Users/tt/goworkspace/src/dify-go/docs/TODO.md).
- A detailed architecture and design document at [ARCHITECTURE.md](/Users/tt/goworkspace/src/dify-go/docs/ARCHITECTURE.md).

This repository is a Go migration project built on top of, and in tribute to, [langgenius/dify](https://github.com/langgenius/dify).

## Planning Docs

- Migration status: [GO_MIGRATION.md](/Users/tt/goworkspace/src/dify-go/docs/GO_MIGRATION.md)
- Implementation backlog: [TODO.md](/Users/tt/goworkspace/src/dify-go/docs/TODO.md)
- Architecture and design: [ARCHITECTURE.md](/Users/tt/goworkspace/src/dify-go/docs/ARCHITECTURE.md)

## What Works Now

The Go server keeps Dify's existing API prefixes so the frontend can continue calling the same paths:

- `GET /console/api/system-features`
- `GET /console/api/setup`
- `POST /console/api/setup`
- `GET /console/api/init`
- `POST /console/api/init`
- `POST /console/api/login`
- `POST /console/api/logout`
- `POST /console/api/refresh-token`
- `GET /console/api/account/profile`
- `GET /console/api/account/avatar`
- `POST /console/api/workspaces/current`
- `GET /console/api/workspaces`
- `GET /console/api/version`
- `GET /console/api/workspaces/current/model-providers`
- `GET /console/api/workspaces/current/models/model-types/{modelType}`
- `GET|POST /console/api/workspaces/current/default-model`
- `GET /console/api/workspaces/current/model-providers/{provider}/models`
- `POST|DELETE /console/api/workspaces/current/model-providers/{provider}/models`
- `PATCH|POST /console/api/workspaces/current/model-providers/{provider}/models/enable`
- `PATCH|POST /console/api/workspaces/current/model-providers/{provider}/models/disable`
- `GET /console/api/workspaces/current/model-providers/{provider}/models/parameter-rules`
- `GET|POST|PUT|DELETE /console/api/workspaces/current/model-providers/{provider}/credentials`
- `POST /console/api/workspaces/current/model-providers/{provider}/credentials/switch`
- `POST /console/api/workspaces/current/model-providers/{provider}/credentials/validate`
- `GET /console/api/workspaces/current/model-providers/{provider}/models/credentials`
- `POST|PUT|DELETE /console/api/workspaces/current/model-providers/{provider}/models/credentials`
- `POST /console/api/workspaces/current/model-providers/{provider}/models/credentials/switch`
- `POST /console/api/workspaces/current/model-providers/{provider}/models/credentials/validate`
- `POST /console/api/workspaces/current/model-providers/{provider}/models/load-balancing-configs/credentials-validate`
- `POST /console/api/workspaces/current/model-providers/{provider}/models/load-balancing-configs/{configId}/credentials-validate`
- `GET /console/api/workspaces/current/model-providers/{provider}/checkout-url`
- `GET /console/api/workspaces/current/tool-providers`
- `GET /console/api/workspaces/current/tools/builtin`
- `GET /console/api/workspaces/current/tools/api`
- `GET /console/api/workspaces/current/tools/workflow`
- `GET /console/api/workspaces/current/tools/mcp`
- `GET /console/api/workspaces/current/tool-provider/builtin/{provider}/tools`
- `GET /console/api/workspaces/current/tool-provider/builtin/{provider}/credentials_schema`
- `GET /console/api/workspaces/current/tool-provider/builtin/{provider}/credentials`
- `POST /console/api/workspaces/current/tool-provider/builtin/{provider}/update`
- `POST /console/api/workspaces/current/tool-provider/builtin/{provider}/delete`
- `POST /console/api/workspaces/current/tool-provider/api/add`
- `GET /console/api/workspaces/current/tool-provider/api/remote`
- `GET /console/api/workspaces/current/tool-provider/api/tools`
- `POST /console/api/workspaces/current/tool-provider/api/update`
- `POST /console/api/workspaces/current/tool-provider/api/delete`
- `GET /console/api/workspaces/current/tool-provider/api/get`
- `POST /console/api/workspaces/current/tool-provider/api/schema`
- `POST /console/api/workspaces/current/tool-provider/api/test/pre`
- `POST /console/api/workspaces/current/tool-provider/workflow/create`
- `POST /console/api/workspaces/current/tool-provider/workflow/update`
- `POST /console/api/workspaces/current/tool-provider/workflow/delete`
- `GET /console/api/workspaces/current/tool-provider/workflow/get`
- `GET /console/api/workspaces/current/tool-provider/workflow/tools`
- `POST|PUT|DELETE /console/api/workspaces/current/tool-provider/mcp`
- `POST /console/api/workspaces/current/tool-provider/mcp/auth`
- `GET /console/api/workspaces/current/tool-provider/mcp/tools/{providerId}`
- `GET /console/api/workspaces/current/tool-provider/mcp/update/{providerId}`
- `GET /console/api/workspaces/current/agent-providers`
- `GET /console/api/workspaces/current/agent-provider/{agentProvider}`
- `POST /console/api/workspaces/current/endpoints/create`
- `GET /console/api/workspaces/current/endpoints/list`
- `GET /console/api/workspaces/current/endpoints/list/plugin`
- `POST /console/api/workspaces/current/endpoints/delete`
- `POST /console/api/workspaces/current/endpoints/update`
- `POST /console/api/workspaces/current/endpoints/enable`
- `POST /console/api/workspaces/current/endpoints/disable`
- `GET /console/api/workspaces/current/triggers`
- `GET /console/api/workspaces/current/trigger-provider/{provider}/icon`
- `GET /console/api/workspaces/current/trigger-provider/{provider}/info`
- `GET /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/list`
- `POST /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/builder/create`
- `GET /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/builder/{subscriptionBuilderId}`
- `POST /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/builder/update/{subscriptionBuilderId}`
- `POST /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/builder/verify-and-update/{subscriptionBuilderId}`
- `GET /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/builder/logs/{subscriptionBuilderId}`
- `POST /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/builder/build/{subscriptionBuilderId}`
- `POST /console/api/workspaces/current/trigger-provider/{subscriptionId}/subscriptions/update`
- `POST /console/api/workspaces/current/trigger-provider/{subscriptionId}/subscriptions/delete`
- `GET|POST|DELETE /console/api/workspaces/current/trigger-provider/{provider}/oauth/client`
- `GET /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/oauth/authorize`
- `POST /console/api/workspaces/current/trigger-provider/{provider}/subscriptions/verify/{subscriptionId}`
- `GET /console/api/workspaces/current/plugin/debugging-key`
- `GET /console/api/workspaces/current/plugin/list`
- `POST /console/api/workspaces/current/plugin/list/latest-versions`
- `POST /console/api/workspaces/current/plugin/list/installations/ids`
- `GET /console/api/workspaces/current/plugin/icon`
- `GET /console/api/workspaces/current/plugin/asset`
- `POST /console/api/workspaces/current/plugin/upload/pkg`
- `POST /console/api/workspaces/current/plugin/upload/github`
- `POST /console/api/workspaces/current/plugin/upload/bundle`
- `POST /console/api/workspaces/current/plugin/install/pkg`
- `POST /console/api/workspaces/current/plugin/install/github`
- `POST /console/api/workspaces/current/plugin/install/marketplace`
- `GET /console/api/workspaces/current/plugin/marketplace/pkg`
- `GET /console/api/workspaces/current/plugin/fetch-manifest`

说明：`upload/bundle` 当前已经支持从 bundle 压缩包内的 JSON / YAML 依赖声明恢复 `marketplace/github/package` 三类兼容依赖。
- `GET /console/api/workspaces/current/plugin/tasks`
- `GET /console/api/workspaces/current/plugin/tasks/{taskId}`
- `POST /console/api/workspaces/current/plugin/tasks/{taskId}/delete`
- `POST /console/api/workspaces/current/plugin/tasks/delete_all`
- `POST /console/api/workspaces/current/plugin/tasks/{taskId}/delete/{identifier}`
- `POST /console/api/workspaces/current/plugin/upgrade/marketplace`
- `POST /console/api/workspaces/current/plugin/upgrade/github`
- `POST /console/api/workspaces/current/plugin/uninstall`
- `POST /console/api/workspaces/current/plugin/permission/change`
- `GET /console/api/workspaces/current/plugin/permission/fetch`
- `GET /console/api/workspaces/current/plugin/parameters/dynamic-options`
- `POST /console/api/workspaces/current/plugin/parameters/dynamic-options-with-credentials`
- `POST /console/api/workspaces/current/plugin/preferences/change`
- `GET /console/api/workspaces/current/plugin/preferences/fetch`
- `POST /console/api/workspaces/current/plugin/preferences/autoupgrade/exclude`
- `GET /console/api/workspaces/current/plugin/readme`
- `GET /console/api/apps`
- `POST /console/api/apps`
- `GET /console/api/apps/{id}`
- `PUT /console/api/apps/{id}`
- `DELETE /console/api/apps/{id}`
- `POST /console/api/apps/{id}/copy`
- `GET /console/api/apps/{id}/api-keys`
- `POST /console/api/apps/{id}/api-keys`
- `DELETE /console/api/apps/{id}/api-keys/{keyId}`
- `GET /console/api/apps/{id}/export`
- `POST /console/api/apps/imports`
- `GET /console/api/apps/imports/{id}/check-dependencies`
- `POST /console/api/apps/{id}/convert-to-workflow`
- `POST /console/api/apps/{id}/site-enable`
- `POST /console/api/apps/{id}/api-enable`
- `POST /console/api/apps/{id}/site`
- `POST /console/api/apps/{id}/site/access-token-reset`

说明：app / pipeline 的 `check-dependencies` 已经切到 Go，并会基于现有 app model config 与 workflow graph 提取插件依赖。
- `GET /console/api/apps/{id}/trace`
- `POST /console/api/apps/{id}/trace`
- `GET /console/api/apps/{id}/trace-config`
- `POST|PATCH|DELETE /console/api/apps/{id}/trace-config`
- `POST /console/api/apps/{id}/model-config`
- `GET /console/api/apps/{id}/conversation-variables`
- `GET /console/api/apps/{id}/annotations/count`
- `POST /console/api/apps/{id}/annotations`
- `DELETE /console/api/apps/{id}/annotations/{annotationId}`
- `POST /console/api/apps/{id}/feedbacks`
- `GET /console/api/apps/{id}/chat-conversations`
- `GET /console/api/apps/{id}/chat-conversations/{conversationId}`
- `GET /console/api/apps/{id}/completion-conversations`
- `GET /console/api/apps/{id}/completion-conversations/{conversationId}`
- `GET /console/api/apps/{id}/workflow-app-logs`
- `GET /console/api/workflow/{workflowRunId}/pause-details`
- `GET /console/api/apps/{id}/chat-messages`
- `GET|POST|PUT /console/api/apps/{id}/server`
- `GET /console/api/apps/{id}/server/refresh`
- `GET /console/api/apps/{id}/workflows/draft`
- `POST /console/api/apps/{id}/workflows/draft`
- `GET /console/api/apps/{id}/workflows/default-workflow-block-configs`
- `GET /console/api/apps/{id}/workflows/default-workflow-block-configs/{blockType}`
- `GET /console/api/apps/{id}/workflows/publish`
- `POST /console/api/apps/{id}/workflows/publish`
- `GET /console/api/apps/{id}/workflows`
- `PATCH /console/api/apps/{id}/workflows/{versionId}`
- `DELETE /console/api/apps/{id}/workflows/{versionId}`
- `POST /console/api/apps/{id}/workflows/{versionId}/restore`
- `GET /console/api/apps/{id}/workflows/draft/environment-variables`
- `POST /console/api/apps/{id}/workflows/draft/environment-variables`
- `GET /console/api/apps/{id}/workflows/draft/conversation-variables`
- `POST /console/api/apps/{id}/workflows/draft/conversation-variables`
- `GET /console/api/apps/{id}/workflows/draft/system-variables`
- `GET /console/api/apps/{id}/workflows/draft/variables`
- `DELETE /console/api/apps/{id}/workflows/draft/variables`
- `DELETE /console/api/apps/{id}/workflows/draft/variables/{varId}`
- `PUT /console/api/apps/{id}/workflows/draft/variables/{varId}/reset`
- `GET /console/api/apps/{id}/workflows/draft/nodes/{nodeId}/variables`
- `DELETE /console/api/apps/{id}/workflows/draft/nodes/{nodeId}/variables`
- `GET /console/api/apps/{id}/workflows/draft/nodes/{nodeId}/last-run`
- `POST /console/api/apps/{id}/workflows/draft/nodes/{nodeId}/run`
- `POST /console/api/apps/{id}/workflows/draft/nodes/{nodeId}/trigger/run`
- `POST /console/api/apps/{id}/workflows/draft/run`
- `POST /console/api/apps/{id}/advanced-chat/workflows/draft/run`
- `POST /console/api/apps/{id}/workflows/draft/trigger/run`
- `POST /console/api/apps/{id}/workflows/draft/trigger/run-all`
- `POST /console/api/apps/{id}/workflows/draft/iteration/nodes/{nodeId}/run`
- `POST /console/api/apps/{id}/advanced-chat/workflows/draft/iteration/nodes/{nodeId}/run`
- `POST /console/api/apps/{id}/workflows/draft/loop/nodes/{nodeId}/run`
- `POST /console/api/apps/{id}/advanced-chat/workflows/draft/loop/nodes/{nodeId}/run`
- `GET /console/api/apps/{id}/workflow-runs`
- `GET /console/api/apps/{id}/advanced-chat/workflow-runs`
- `GET /console/api/apps/{id}/workflow-runs/{runId}`
- `GET /console/api/apps/{id}/workflow-runs/{runId}/node-executions`
- `POST /console/api/apps/{id}/workflow-runs/tasks/{taskId}/stop`
- `POST /console/api/rag/pipeline/empty-dataset`
- `POST /console/api/rag/pipeline/dataset`
- `GET /console/api/rag/pipeline/templates`
- `GET /console/api/rag/pipeline/templates/{templateId}`
- `PATCH|DELETE|POST /console/api/rag/pipeline/customized/templates/{templateId}`
- `GET /console/api/rag/pipelines/datasource-plugins`
- `GET /console/api/auth/plugin/datasource/list`
- `GET /console/api/auth/plugin/datasource/default-list`
- `GET|POST /console/api/auth/plugin/datasource/{pluginId}/{provider}`
- `POST /console/api/auth/plugin/datasource/{pluginId}/{provider}/update`
- `POST /console/api/auth/plugin/datasource/{pluginId}/{provider}/delete`
- `POST /console/api/auth/plugin/datasource/{pluginId}/{provider}/default`
- `POST|DELETE /console/api/auth/plugin/datasource/{pluginId}/{provider}/custom-client`
- `GET /console/api/oauth/plugin/{pluginId}/{provider}/datasource/get-authorization-url`
- `GET /console/api/oauth/plugin/{pluginId}/{provider}/datasource/callback`
- `POST /console/api/rag/pipelines/imports`
- `POST /console/api/rag/pipelines/imports/{importId}/confirm`
- `GET /console/api/rag/pipelines/{pipelineId}/exports`
- `POST /console/api/rag/pipelines/{pipelineId}/customized/publish`
- `GET|POST /console/api/rag/pipelines/{pipelineId}/workflows/draft`
- `GET /console/api/rag/pipelines/{pipelineId}/workflows/default-workflow-block-configs`
- `GET /console/api/rag/pipelines/{pipelineId}/workflows/default-workflow-block-configs/{blockType}`
- `GET|POST /console/api/rag/pipelines/{pipelineId}/workflows/publish`
- `GET /console/api/rag/pipelines/{pipelineId}/workflows`
- `PATCH /console/api/rag/pipelines/{pipelineId}/workflows/{versionId}`
- `DELETE /console/api/rag/pipelines/{pipelineId}/workflows/{versionId}`
- `POST /console/api/rag/pipelines/{pipelineId}/workflows/{versionId}/restore`
- `GET /console/api/rag/pipelines/{pipelineId}/workflows/draft/pre-processing/parameters`
- `GET /console/api/rag/pipelines/{pipelineId}/workflows/published/pre-processing/parameters`
- `GET /console/api/rag/pipelines/{pipelineId}/workflows/draft/processing/parameters`
- `GET /console/api/rag/pipelines/{pipelineId}/workflows/published/processing/parameters`
- `POST /console/api/rag/pipelines/{pipelineId}/workflows/draft/datasource/nodes/{nodeId}/run`
- `POST /console/api/rag/pipelines/{pipelineId}/workflows/published/datasource/nodes/{nodeId}/run`
- `POST /console/api/rag/pipelines/{pipelineId}/workflows/published/run`
- `GET /console/api/rag/pipelines/{pipelineId}/workflow-runs`
- `GET /console/api/rag/pipelines/{pipelineId}/workflow-runs/{runId}`
- `GET /console/api/rag/pipelines/{pipelineId}/workflow-runs/{runId}/node-executions`
- `POST /console/api/rag/pipelines/{pipelineId}/workflow-runs/tasks/{taskId}/stop`

说明：这一批 route 会先把 `pipelineId` 解析到 Go 侧的 workflow app，再复用既有 workflow draft/publish/version/run 处理器；同时新增 `rag_pipeline_variables` 的持久化与参数过滤，空白 pipeline dataset 删除时也会同步回收绑定的 workflow app。

补充：RAG pipeline DSL 现在已经可以在 Go 侧完成导出、导入和“从 DSL 创建 dataset”。导入时会把 `workflow.graph/features/environment_variables/conversation_variables/rag_pipeline_variables` 回写到 Go workflow draft；如果 DSL 中的 `knowledge-index` 节点带了知识库配置，也会同步更新 dataset 的 `doc_form/indexing_technique/retrieval_model/embedding_model/summary_index_setting`。

补充：pipeline template 目录也已经迁到 Go。内置 built-in 模板现在由 Go 直接提供稳定目录，customized template 支持从当前 pipeline 发布、列表/详情查询、元信息更新、DSL 导出和删除；这些能力通过新的 `pipeline_templates` 持久化切片保存在本地状态文件里。

补充：RAG pipeline 的 `published/run` 现已接到 Go，支持 published preview、首次创建文档、以及基于 `original_document_id` 的文档重处理；运行请求会同时把 datasource 和 processing inputs 写回 dataset document 的 pipeline execution log，前端 create-from-pipeline 与 document settings 可以直接复用这条链路。

补充：`GET /console/api/rag/pipelines/datasource-plugins` 现在不再返回空兼容数组，而是由 Go 直接提供 RAG pipeline datasource catalog。当前 `local_file` 仍然作为内建数据源始终可用，而 `online_document / website_crawl / online_drive` 这三类 provider 会优先根据 workspace plugin 安装态暴露；如果工作区里已经存在旧的 datasource credential / OAuth client 状态，也会继续以兼容回退方式保留在 catalog 中，避免迁移中把既有授权状态直接隐藏掉。

补充：datasource auth 相关的 `list / default-list / credential CRUD / default / custom-client / oauth authorization-url / oauth callback` 也已经切到 Go。当前实现除了把 datasource 凭证和自定义 OAuth client 配置保存在 workspace 本地状态里，还会和 workspace plugin 安装态一起决定 provider 是否对前端可见；provider 卸载后，如果没有遗留 credential 状态就会直接从 auth list 消失，如果仍有已有 credential，则会继续以 `is_installed=false` 的兼容形态保留出来。

补充：RAG pipeline datasource node run 也已经迁到 Go。当前新增的 draft/published `/datasource/nodes/{nodeId}/run` SSE 兼容层已覆盖 `online_document / website_crawl / online_drive` 三类在线数据源，会结合 workspace plugin 安装态和 datasource credential 状态做基础校验，并返回前端 create-from-pipeline 已可直接消费的 notion workspace/page、website crawl result、online drive bucket/file 结构。
- `GET /console/api/datasets/retrieval-setting`
- `GET /console/api/datasets/process-rule`
- `POST /console/api/datasets/indexing-estimate`
- `GET /console/api/datasets/api-base-info`
- `GET|POST /console/api/datasets/api-keys`
- `DELETE /console/api/datasets/api-keys/{keyId}`
- `POST /console/api/datasets/external`
- `GET|POST /console/api/datasets/external-knowledge-api`
- `GET|PATCH|DELETE /console/api/datasets/external-knowledge-api/{apiId}`
- `GET /console/api/datasets/external-knowledge-api/{apiId}/use-check`
- `GET /console/api/datasets/metadata/built-in`
- `GET /console/api/datasets/batch_import_status/{jobId}`
- `GET|POST /console/api/datasets`
- `POST /console/api/datasets/init`
- `GET|PATCH|DELETE /console/api/datasets/{datasetId}`
- `GET /console/api/datasets/{datasetId}/use-check`
- `GET /console/api/datasets/{datasetId}/related-apps`
- `POST /console/api/datasets/{datasetId}/api-keys/enable`
- `POST /console/api/datasets/{datasetId}/api-keys/disable`
- `GET|POST /console/api/datasets/{datasetId}/metadata`
- `PATCH|DELETE /console/api/datasets/{datasetId}/metadata/{metadataId}`
- `POST /console/api/datasets/{datasetId}/metadata/built-in/enable`
- `POST /console/api/datasets/{datasetId}/metadata/built-in/disable`
- `GET|POST|DELETE /console/api/datasets/{datasetId}/documents`
- `POST /console/api/datasets/{datasetId}/documents/metadata`
- `PATCH /console/api/datasets/{datasetId}/documents/status/{action}/batch`
- `POST /console/api/datasets/{datasetId}/documents/generate-summary`
- `GET /console/api/datasets/{datasetId}/documents/{documentId}`
- `GET /console/api/datasets/{datasetId}/documents/{documentId}/download`
- `GET /console/api/datasets/{datasetId}/documents/{documentId}/pipeline-execution-log`
- `GET /console/api/datasets/{datasetId}/documents/{documentId}/notion/sync`
- `GET /console/api/datasets/{datasetId}/documents/{documentId}/website-sync`
- `PUT /console/api/datasets/{datasetId}/documents/{documentId}/metadata`
- `GET /console/api/datasets/{datasetId}/documents/{documentId}/indexing-status`
- `POST /console/api/datasets/{datasetId}/documents/{documentId}/rename`
- `PATCH /console/api/datasets/{datasetId}/documents/{documentId}/processing/pause`
- `PATCH /console/api/datasets/{datasetId}/documents/{documentId}/processing/resume`
- `POST /console/api/datasets/{datasetId}/documents/download-zip`
- `GET|DELETE /console/api/datasets/{datasetId}/documents/{documentId}/segments`
- `POST /console/api/datasets/{datasetId}/documents/{documentId}/segment`
- `PATCH /console/api/datasets/{datasetId}/documents/{documentId}/segment/enable`
- `PATCH /console/api/datasets/{datasetId}/documents/{documentId}/segment/disable`
- `PATCH /console/api/datasets/{datasetId}/documents/{documentId}/segments/{segmentId}`
- `GET|POST /console/api/datasets/{datasetId}/documents/{documentId}/segments/{segmentId}/child_chunks`
- `PATCH|DELETE /console/api/datasets/{datasetId}/documents/{documentId}/segments/{segmentId}/child_chunks/{childChunkId}`
- `POST /console/api/datasets/{datasetId}/documents/{documentId}/segments/batch_import`
- `GET /console/api/datasets/{datasetId}/batch/{batchId}/indexing-status`
- `GET /console/api/datasets/{datasetId}/auto-disable-logs`
- `GET /console/api/datasets/{datasetId}/queries`
- `GET /console/api/datasets/{datasetId}/error-docs`
- `POST /console/api/datasets/{datasetId}/hit-testing`
- `POST /console/api/datasets/{datasetId}/external-hit-testing`
- `POST /console/api/datasets/{datasetId}/retry`

说明：dataset 第二批迁移已经把 metadata、segment、child chunk 和 batch import 状态轮询接入 Go 状态文件，并通过真实 HTTP 冒烟验证了创建知识库、metadata CRUD、文档 metadata 更新、segment CRUD、child chunk CRUD 与 batch import 状态查询。

补充：本轮继续把 external knowledge API CRUD、external dataset 创建、单文档下载和批量 zip 下载迁到 Go，并验证了 external API 绑定关系会随着更新/删除同步到 dataset 状态。

补充：dataset 文档也已经具备 pipeline execution log 兼容状态，并通过真实 HTTP 冒烟验证了 local file / website / notion 三类 datasource 的 execution log 返回，以及 notion / website 的 sync 动作。

补充：文件上传现在已经走 Go 侧持久化存储，支持 console/public 本地上传、`remote-files/upload` 远程拉取入库，以及知识库 hit-testing 里 `attachment_ids` 到 `image_query.file_info` 的查询记录回写。

补充：`/datasets/{datasetId}/external-hit-testing` 现在会按上游 external knowledge API 契约调用已绑定的 `endpoint/retrieval`，透传 `knowledge_id`、`top_k` 和 `score_threshold`，并用 Go HTTP 集成测试校验请求头、请求体、query 校验和命中记录写回。
- `POST /console/api/remote-files/upload`
- `GET|POST /console/api/files/upload`
- `GET /console/api/files/support-type`
- `GET /console/api/files/{id}/preview`
- `GET /files/{id}/file-preview`
- `GET /files/{id}/image-preview`
- `GET|POST /api/files/upload`
- `GET /api/files/support-type`
- `POST /api/remote-files/upload`
- `GET /console/api/spec/schema-definitions`
- `GET /console/api/rag/pipelines/imports/{pipelineId}/check-dependencies`
- `GET /api/system-features`
- `GET /api/login/status`
- `POST /api/logout`
- `ANY /trigger/builders/{builderId}`
- `ANY /trigger/subscriptions/{subscriptionId}`
- `ANY /trigger/endpoints/{hookId}`

## Compatibility Mode

Unported API routes can be forwarded to the original Python backend by setting:

```bash
export DIFY_GO_LEGACY_API_BASE_URL=http://127.0.0.1:5001
```

That lets us migrate endpoint groups incrementally while the frontend stays untouched.

## Run

Start the Go backend:

```bash
go run ./cmd/dify-server
```

Start the unchanged frontend in another terminal:

```bash
pnpm install
pnpm --dir web dev
```

The frontend already defaults to:

- Console API: `http://localhost:5001/console/api`
- Public API: `http://localhost:5001/api`

## Current Limits

- The Go backend currently uses a lightweight file-backed bootstrap store at `var/state.json`.
- Session storage is in-memory for now.
- Dataset external retrieval、bulk import semantics，以及 RAG pipeline 的 datasource plugin 发现、真实 transform/batch 执行语义仍在继续迁移。
- Most remaining business routes are still pending migration and should either be proxied to the Python backend or implemented next by priority.
