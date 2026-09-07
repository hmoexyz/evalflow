# 评估流程表系统 (EvalFlow)

一个轻量的评估流程表系统：**支持多账号注册**，每个账号拥有自己独立的评估项与流程表。账号创建评估项与流程表后，通过分享链接让访客无需登录即可逐项评分（0~10 分，6 分及以下为不合格）、上传证据文件，并自动生成统计结果（总得分/满分、合格项数/总项数），支持一键导出为图片。

## 技术栈

- 前端：Vite + React + TailwindCSS（TypeScript）
- 后端：Go（标准库 net/http）
- 数据库：SQLite（纯 Go 驱动 `modernc.org/sqlite`，无需 CGO）

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
│   └── auth.go         # 会话 token 与密码哈希
├── frontend/           # Vite + React + Tailwind 前端
│   └── src/
│       ├── pages/      # LoginPage(登录/注册) / AdminPage / SharePage / ResultPage
│       └── components/ # ItemsTab / FormsTab / ResultCard
└── start.bat           # Windows 一键启动脚本
```

## 快速开始

### 方式一：生产模式（单进程，推荐）

```bat
:: Windows 双击 start.bat，或手动执行：
cd frontend && npm install && npm run build
cd ..\backend && go build -o evalflow.exe .
evalflow.exe
```

启动后访问 **http://localhost:8080** ，后端会自动托管前端构建产物（`frontend/dist`）。首次使用在登录页点击「立即注册」创建账号。

### 方式二：开发模式（热更新）

```bash
# 终端 1：后端
cd backend && go run .

# 终端 2：前端（Vite 代理 /api、/uploads 到 :8080）
cd frontend && npm install && npm run dev
```

前端开发地址 http://localhost:5173 ，后端 API http://localhost:8080 。

### 方式三：Docker 部署

项目根目录提供 `Dockerfile` 与 `docker-compose.yml`，一条命令即可打包并运行（前端构建 → Go 交叉编译 → 精简运行时镜像，单容器）。

```bash
docker compose up -d --build
```

启动后访问 http://localhost:8080 。其他常用命令：

```bash
docker compose down        # 停止（保留数据）
docker compose down -v     # 停止并删除数据卷
docker compose logs -f     # 查看日志
```

数据持久化：SQLite 数据库（评估项/流程表/评测结果/上传文件 BLOB）保存在命名卷 `data`（容器内 `/data`）中，重建/升级容器不会丢失。若想直接备份查看，可把 `docker-compose.yml` 中的卷改为本机目录（如 `./data:/data`，Linux 下需注意容器内以 uid 10001 运行的文件写权限）。

## 配置（环境变量）

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `ADDR` | `:8080` | 监听地址 |
| `DB_PATH` | `data.db` | SQLite 数据库文件路径 |
| `FRONTEND_DIST` | `../frontend/dist` | 前端构建产物目录 |

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
