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
- `GET /console/api/files/upload`
- `GET /console/api/files/support-type`
- `GET /console/api/files/{id}/preview`
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
- Dataset metadata, segments, external knowledge base, and RAG pipeline deep-link semantics are still only partially migrated.
- Most remaining business routes are still pending migration and should either be proxied to the Python backend or implemented next by priority.
