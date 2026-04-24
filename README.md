# dify-go

`dify-go` 是一个面向 [Dify](https://github.com/langgenius/dify) 的增量式 Go 后端迁移项目。

本仓库直接复用上游 Dify 的前端工作区，并在尽量不改动前端的前提下，把后端能力按业务域逐步迁移到 Go。项目基于并致敬 [https://github.com/langgenius/dify](https://github.com/langgenius/dify)，感谢 Dify 团队和社区把原始产品开放出来，也让这条迁移路线成为可能。

## 目标

- 保留上游前端，优先保证控制台和运行时页面继续可用。
- 保留原有 `/console/api`、`/api`、`/trigger` 等 HTTP 入口和主要响应结构。
- 按业务域迁移，而不是一次性重写整套后端。
- 在迁移过程中保留 legacy fallback，确保系统始终可运行。

## 当前状态

目前 Go 侧已经迁入并可自持状态的核心域包括：

- 初始化、登录、刷新令牌、退出登录、账号基础信息、工作区基础接口
- 应用 CRUD、导入导出、API Key、站点/API 开关、Tracing、Model Config
- Workflow Draft、发布、版本历史、运行历史、节点运行辅助接口
- Workspace Model Provider、默认模型、凭证、模型启停、参数规则
- Workspace Tools、MCP、Endpoints、Triggers
- Plugin 平台的基础兼容链路
- 账号周边认证 flow、工作区成员、邀请激活、ownership transfer 与 SSO 兼容入口
- Dataset 主链路的第一批基础路由：列表、创建、详情、更新、删除、文档主列表、命中测试和部分索引状态接口

更细的迁移清单见：

- [docs/GO_MIGRATION.md](./docs/GO_MIGRATION.md)
- [docs/TODO.md](./docs/TODO.md)
- [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)

## 仓库结构

```text
cmd/
  dify-server/           Go 服务入口
  dify-route-manifest/   路由清单生成工具
internal/
  config/                运行配置
  server/                HTTP 路由、兼容层、handler
  state/                 已迁业务域的状态模型和持久化
docs/
  GO_MIGRATION.md        迁移进度
  TODO.md                迁移待办
  ARCHITECTURE.md        架构、设计思路与原理
web/
  上游 Dify 前端代码
```

## 设计原则

1. 前端兼容优先。能不改 `web/` 就不改。
2. 业务域闭环优先。一个页面背后的关键链路尽量一起迁。
3. 已迁能力必须由 Go 自持状态，而不是“接口转发 + 本地拼装”。
4. 未迁能力明确 fallback 到 legacy backend，而不是返回半成品行为。
5. 先追求稳定可用，再做深层优化和更重的基础设施替换。

## 快速启动

启动 Go 后端：

```bash
go run ./cmd/dify-server
```

单独启动前端开发环境：

```bash
pnpm install
pnpm --dir web dev
```

如果还需要把未迁移路由转发回原始 Python 后端，可以设置：

```bash
export DIFY_GO_LEGACY_API_BASE_URL=http://127.0.0.1:5001
```

然后再启动 `dify-go`。这样前端继续请求同一组 API 前缀，而 Go 会只接管已经迁完的部分。

## 推荐阅读顺序

1. [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)
   说明整体架构、模块边界、设计思路、迁移原理。
2. [docs/TODO.md](./docs/TODO.md)
   说明每一阶段还有什么工作没做，以及下一步怎么推进。
3. [docs/GO_MIGRATION.md](./docs/GO_MIGRATION.md)
   说明已经迁入 Go 的接口面和当前兼容边界。

## 开发约定

- 每做完一个有边界的迁移切片，都执行 `go build ./...`。
- 关键功能用 `curl` 或前端页面做冒烟验证。
- 同步更新 `README`、`docs/GO_MIGRATION.md`、`docs/TODO.md`、`docs/ARCHITECTURE.md`。
- 保持主线可运行，并在停下之前提交并 push 当前进度。

## 为什么不直接重写前端

因为这个项目迁的是“后端实现”，不是重做产品。上游前端本身已经沉淀了大量成熟的交互、状态流转和接口契约，它正好也是最严格的兼容性测试基线。只要前端基本不动还能继续跑，迁移方向通常就是对的。

## 致敬上游

`dify-go` 明确建立在 [langgenius/dify](https://github.com/langgenius/dify) 的产品和开源工作之上。这里做的是一次工程化迁移和兼容重建，不是对原项目贡献的替代叙事。所有关于产品形态、交互经验和生态语义的 credit，都首先属于 Dify 原始项目。
