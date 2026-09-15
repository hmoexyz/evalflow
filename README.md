# 评估流程表系统 (EvalFlow)

一个轻量的评估流程表系统：**支持多账号注册**，每个账号拥有自己独立的评估项与流程表。账号创建评估项与流程表后，通过分享链接让访客无需登录即可逐项评分（0~10 分，6 分及以下为不合格）、上传证据文件，并自动生成统计结果（总得分/满分、合格项数/总项数），支持一键导出为图片。

## 技术栈

- 前端：Vite + React + TailwindCSS（TypeScript）
- 后端：Go 1.22+（标准库 net/http）
- 数据库：SQLite（纯 Go 驱动 `modernc.org/sqlite`，无需 CGO）
- 部署：宿主机构建 + Docker Compose（Nginx 托管前端、反向代理到 Go 后端）

## 功能

0. **多账号**：任何人都可注册账号，每个账号的评估项、流程表、评测结果相互独立、互不可见
1. 账号创建**评估项**（名称、说明），存储在数据库中
2. 账号创建**流程表**，一个流程表可包含任意数量的评估项
3. 账号可**公开分享**流程表，访客无需登录即可查看和使用
4. 网站首页（`/`）展示**所有已发布的流程表列表**，访客未登录也可直接浏览并选择填写
4. 访客填写**所在餐厅与测评人**，对每个评估项**打分（0~10，6 分及以下不合格）**，并可**拍照/上传多张图片或文件**作为证据
5. 评分完成后生成**最终统计**：总得分/满分，合格项数/总项数，并支持**导出为图片**
6. 每次提交自动生成**结果查看链接与二维码**，任何人可打开链接查看本次结果与证据附件；导出的结果图片包含该链接的二维码

## 目录结构

```
├── backend/            # Go 后端
│   ├── main.go         # 入口：路由、静态文件托管、CORS
│   ├── handlers.go     # HTTP 处理器（登录/注册、账号数据管理）
│   ├── store.go        # SQLite 数据访问（账号隔离）
│   ├── models.go       # 数据结构与统计逻辑
│   ├── auth.go         # 会话 token 与密码哈希
│   ├── evalflow        # build.sh 生成的二进制（不提交）
│   ├── Dockerfile      # 后端镜像：Ubuntu + 拷贝二进制运行
│   └── .dockerignore
├── frontend/           # Vite + React + Tailwind 前端
│   ├── nginx.conf      # Nginx 静态托管与 /api、/uploads 反向代理
│   ├── dist/           # 前端构建产物（build.sh 生成，挂载给 nginx）
│   └── src/
│       ├── pages/      # LoginPage / AdminPage / PublicFormsPage / SharePage / ResultPage
│       └── components/ # ItemsTab / FormsTab / ResultsTab / ResultCard
├── build.sh            # 一键构建前端与后端
├── docker-compose.yml  # 容器编排（backend + Nginx 前端）
└── README.md
```

## 快速开始

### 方式一：Docker 部署（推荐）

前后端都在**宿主机**构建，容器只负责运行：`frontend/dist` 挂载给 Nginx，`backend/evalflow` 打进 Ubuntu 镜像。先构建，再启动：

```bash
./build.sh                    # 1. 构建后端二进制 + 前端产物
docker compose up -d --build  # 2. 构建镜像并启动容器
```

`build.sh` 依次执行：

```bash
cd backend  && go build                       # 生成 backend/evalflow
cd frontend && npm install && npm run build   # 生成 frontend/dist
```

启动后访问 **http://localhost:1001** （映射自容器内 Nginx 的 80 端口）。

常用命令：

```bash
docker compose ps          # 查看状态
docker compose logs -f     # 查看日志
docker compose down        # 停止并删除容器
```

容器说明：

- `backend`：基于 `ubuntu:latest`，`CMD /opt/backend/evalflow`，仅监听容器内 `:8080`，通过 compose 网络由 Nginx 访问
- 前端容器（`nginx:latest`）：将 `./frontend/dist` 挂载到 `/usr/share/nginx/html`，`./frontend/nginx.conf` 挂载到 `/etc/nginx/conf.d/default.conf`，并把 `/api/`、`/uploads/` 反向代理到 `backend:8080`

> 注意：必须先运行 `./build.sh`，否则 `backend/evalflow` 或 `frontend/dist` 不存在，镜像构建 / Nginx 托管会失败。

数据持久化与迁移：SQLite 数据库绑定挂载到宿主机 `./data/data.db`（容器内 `/data`，由 `DB_PATH` 指定），`docker compose down` 删除容器不会丢失。迁移时只需拷贝整个 `data/` 目录到目标机器相同位置再启动即可。

### 方式二：生产模式（单进程，不用 Docker）

```bash
cd frontend && npm install && npm run build
cd ../backend && go build -o evalflow .
./evalflow
```

启动后访问 **http://localhost:8080** ，后端会自动托管前端构建产物（`../frontend/dist`）。首次使用在登录页点击「立即注册」创建账号。

### 方式三：开发模式（热更新）

```bash
# 终端 1：后端
cd backend && go run .

# 终端 2：前端（Vite 代理 /api、/uploads 到 :8080）
cd frontend && npm install && npm run dev
```

前端开发地址 http://localhost:5173 ，后端 API http://localhost:8080 。

## 配置（环境变量）

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `ADDR` | `:8080` | 监听地址 |
| `DB_PATH` | `data.db` | SQLite 数据库文件路径 |
| `FRONTEND_DIST` | `../frontend/dist` | 前端构建产物目录（存在时后端直接托管前端） |

> 注：升级自旧版（单管理员密码）的数据库，原有数据不属于任何账号，会在首次启动时自动清除；请先注册新账号后重新创建评估项与流程表。

## 主要 API

| 方法 | 路径 | 说明 | 认证 |
| --- | --- | --- | --- |
| POST | `/api/register` | 注册账号，返回 token | - |
| POST | `/api/login` | 账号登录（用户名+密码），返回 token | - |
| POST | `/api/password` | 修改密码（需提供当前密码） | 登录 |
| GET/POST | `/api/items` | 当前账号评估项列表 / 新建 | 登录 |
| PUT/DELETE | `/api/items/{id}` | 更新 / 删除评估项（仅限本账号） | 登录 |
| GET/POST | `/api/forms` | 当前账号流程表列表 / 新建 | 登录 |
| GET/PUT/DELETE | `/api/forms/{id}` | 流程表详情 / 更新 / 删除（仅限本账号） | 登录 |
| POST | `/api/forms/{id}/publish` | 发布，返回分享 token | 登录 |
| POST | `/api/forms/{id}/unpublish` | 取消发布 | 登录 |
| GET | `/api/forms/{id}/submissions` | 提交记录列表（仅限本账号） | 登录 |
| GET | `/api/submissions` | 本账号全部评测结果 | 登录 |
| GET | `/api/share` | 所有已发布的流程表列表（公开） | - |
| GET | `/api/share/{token}` | 访客获取流程表（公开） | - |
| POST | `/api/share/{token}/submissions` | 访客提交评分（公开） | - |
| GET | `/api/results/{token}` | 查看评估结果与附件（公开） | - |
| POST | `/api/upload` | 上传证据文件（以 BLOB 存入数据库） | - |
| GET | `/uploads/{name}` | 读取上传的文件内容（BLOB，公开） | - |

## 合格判定

每项得分 **≥ 7 分** 记为合格，**≤ 6 分** 记为不合格。最终统计中「合格项数」按此标准计算。
