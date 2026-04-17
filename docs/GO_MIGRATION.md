# Dify Go Migration

`dify-go` now contains:

- The original Dify frontend workspace copied from `../dify` without app-level changes.
- A new Go backend compatibility layer under `cmd/` and `internal/`.
- A generated Python route inventory at [route-manifest.json](/Users/tt/goworkspace/src/dify-go/docs/route-manifest.json).

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
- `POST /console/api/apps/{id}/convert-to-workflow`
- `POST /console/api/apps/{id}/site-enable`
- `POST /console/api/apps/{id}/api-enable`
- `POST /console/api/apps/{id}/site`
- `POST /console/api/apps/{id}/site/access-token-reset`
- `GET /console/api/apps/{id}/trace`
- `POST /console/api/apps/{id}/trace`
- `GET /console/api/apps/{id}/trace-config`
- `POST|PATCH|DELETE /console/api/apps/{id}/trace-config`
- `POST /console/api/apps/{id}/model-config`
- `GET /console/api/apps/{id}/conversation-variables`
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
- `GET /console/api/files/upload`
- `GET /console/api/files/support-type`
- `GET /console/api/files/{id}/preview`
- `GET /console/api/spec/schema-definitions`
- `GET /api/system-features`
- `GET /api/login/status`
- `POST /api/logout`

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
- Most business routes are still pending migration and should either be proxied to the Python backend or implemented next by priority.
