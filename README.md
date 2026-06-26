# Survey AI Workspace Assistant

## Personal MVP Scope

Current priority: single-user normal use.

Must work now:

- Deploy PostgreSQL + API on a VPS.
- Load `extension/dist` in Chrome or an anti-detect browser.
- Set the API base URL in the extension settings.
- Bind one browser environment to one Profile, for example `profile-001`.
- Extract current page content.
- Save page snapshots and notes.
- Translate page or selected text.
- Ask AI based on the current page and saved history.
- Search the isolated knowledge base by `profile_id`, `site_key`, and optional `survey_id`.
- Use the admin app only as a data console.

Deferred for later:

- Multi-user login and role permissions.
- Feishu, Notion, Obsidian API sync.
- Team public workflow and scheduled reports.
- Account credential vault.

Useful self-check endpoints:

```text
GET /health
GET /api/system/status
```

`/api/system/status` reports database status, AI chat configuration, embedding configuration, and model names without exposing secrets.

多指纹浏览器问卷 AI 工作台。系统以 `profile_id`、`site_key`、`survey_id` 做数据隔离，提供页面提取、快照、笔记、翻译、AI 当前页面问答、知识库检索、管理员后台和导出能力。

本项目只做辅助理解、翻译、总结、笔记整理和历史检索，不做自动答题、自动提交、验证码处理、风控绕过，也不会把 password、cookie、refresh_token、API key 等凭证送入 AI 或知识库。

## 当前能力

- Go + Gin 后端 API
- PostgreSQL + pgvector 数据库
- Profile / 网站 / 问卷 / 快照 / 笔记 / 翻译 / AI 对话 / 知识库数据模型
- AI 翻译与当前页面问答
- 知识库关键词、全文、可选向量检索
- Chrome Extension MV3 + React 右侧栏
- Gemini 风格输入框与 `/` 命令工作台
- React 管理员后台
- Markdown、CSV、日报导出
- Docker Compose 本地运行
- PowerShell migration 与 smoke test 脚本

## 项目结构

```text
cmd/api/                    Go API 入口
internal/ai/                AI chat 与 embedding client
internal/config/            环境变量配置
internal/database/          PostgreSQL 连接
internal/models/            API JSON model
internal/server/            Gin router 与 handlers
migrations/                 SQL migration
extension/                  Chrome Extension MV3
admin/                      React 管理后台
scripts/apply-migrations.ps1 数据库迁移脚本
scripts/smoke-test.ps1       API 冒烟测试脚本
docker-compose.yml          PostgreSQL(pgvector) + API
```

## 环境变量

复制 `.env.example` 为 `.env`，按需填写：

```text
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_URL=postgres://survey_ai:survey_ai@localhost:5432/survey_ai_workspace?sslmode=disable
AI_API_KEY=
AI_BASE_URL=https://api.openai.com/v1
AI_MODEL=gpt-4.1-mini
AI_EMBEDDING_MODEL=text-embedding-3-small
```

没有配置 `AI_API_KEY` 时，基础 CRUD、快照、笔记、导出仍可用；AI 翻译、AI 对话和向量 embedding 不可用。知识库搜索会保留关键词/全文检索路径。

## 本地运行

启动数据库和 API：

```powershell
docker compose up -d --build
```

执行 migration：

```powershell
.\scripts\apply-migrations.ps1
```

检查 API：

```powershell
Invoke-RestMethod http://localhost:8080/health
```

跑基础链路冒烟测试：

```powershell
.\scripts\smoke-test.ps1
```

## 手动运行 Go API

```powershell
docker compose up -d postgres
.\scripts\apply-migrations.ps1
$env:DATABASE_URL="postgres://survey_ai:survey_ai@localhost:5432/survey_ai_workspace?sslmode=disable"
$env:AI_API_KEY="your-api-key"
go run ./cmd/api
```

## 插件

构建：

```powershell
Set-Location extension
npm install
npm run build
```

加载：

```text
1. 打开 chrome://extensions
2. 开启 Developer mode
3. 点击 Load unpacked
4. 选择 extension/dist
```

常用命令：

```text
/总结
/解释
/翻译
/翻译选中
/保存笔记 note text
/搜索知识库 keyword
/保存快照
/历史笔记
/历史对话
/绑定 profile-001
/问卷 123
```

插件边界：

- 不自动选择选项
- 不提交表单
- 不读取 cookie
- 不读取 password/token 类输入框
- 不把凭证写入知识库

## 管理后台

构建：

```powershell
Set-Location admin
npm install
npm run build
```

本地预览：

```powershell
npm run preview -- --host 127.0.0.1 --port 4174
```

打开：

```text
http://127.0.0.1:4174
```

后台页面覆盖：

- Profile 管理
- 问卷网站管理
- 问卷项目记录
- 页面快照记录
- 笔记记录
- 翻译记录
- AI 对话记录
- 知识库管理
- Markdown / CSV / 日报导出

## 核心 API

```text
POST /api/profiles/bind
GET  /api/profiles
GET  /api/profiles/:id
PUT  /api/profiles/:id

POST /api/sites/detect
GET  /api/sites
POST /api/sites
GET  /api/sites/:site_key
PUT  /api/sites/:site_key

POST /api/surveys
GET  /api/surveys?profile_id=1&site_key=example
GET  /api/surveys/:id
PUT  /api/surveys/:id

POST /api/page-snapshots
GET  /api/page-snapshots?profile_id=1&site_key=example&survey_id=1
GET  /api/page-snapshots/:id

POST   /api/notes
GET    /api/notes?profile_id=1&site_key=example&survey_id=1
GET    /api/notes/:id
PUT    /api/notes/:id
DELETE /api/notes/:id

POST /api/ai/translate
GET  /api/translations?profile_id=1&site_key=example
GET  /api/translations/:id

POST /api/ai/chat
GET  /api/ai/conversations?profile_id=1&site_key=example
GET  /api/ai/conversations/:id

POST   /api/knowledge
GET    /api/knowledge?profile_id=1&site_key=example
GET    /api/knowledge/search?profile_id=1&site_key=example&q=keyword
DELETE /api/knowledge/:id

GET /api/export/markdown?profile_id=1&site_key=example
GET /api/export/notes.csv?profile_id=1&site_key=example
GET /api/export/daily-report?profile_id=1&site_key=example
```

## 知识库检索

`knowledge_chunks` 现在包含：

- `chunk_text`
- `search_vector`，用于 PostgreSQL 全文检索
- `embedding vector(1536)`，用于 pgvector 语义检索

写入页面快照、笔记、AI 对话时会自动生成知识 chunk。配置 `AI_EMBEDDING_MODEL` 且 provider 支持 `/embeddings` 时，会自动写入向量；没有向量时仍可通过全文和 `ILIKE` 关键词检索。

## 数据安全边界

- AI 接口会拒绝明显包含 password、cookie、authorization、bearer、refresh_token、client_secret、api_key 等内容的请求。
- 账号凭证必须另建加密凭证库，不进入本项目知识库。
- 导出只导出业务记录，不导出账号凭证。
- 外部工具如飞书、Notion、Obsidian 只作为导出或展示目标，PostgreSQL 仍是主库。
