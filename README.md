# Fund Dashboard

朋友基金净值看板 —— 用 **NAV 单位法**核算几位朋友在我 Binance 合约账户里的份额、收益与回撤，并向他们透明展示交易。

## 功能

**基金总览 / 成员资产**
- 管理员优先查看基金权益、全员净收益、NAV 与个人资产；成员查看自己的估值、收益与份额
- 基金与纳指 100 / 标普 500 的区间收益对比，支持鼠标、触摸与键盘检查历史读数
- 保证金、闲置资金、全仓杠杆与名义持仓分布；展示数据更新时间
- 管理员持仓明细、可搜索与分页的平仓记录、成员资产对比

**复盘视图（仅 admin）**
- 水下回撤（最大 / 当前回撤）
- 可切换月份与选定日期的每日盈亏日历
- 胜率、赢赔比、平均持仓时长、当前持仓、平仓记录、按标的盈亏

**基金管理（仅 admin）**
- 资金流水与入金 / 赎回记录，保留手动 NAV 与首笔偏离校验选项
- 创建成员与停用 / 恢复访问
- 最近成交与手动净值快照

## 技术栈

- **后端** Go（`net/http`，手写 JWT cookie 鉴权）；纯 Go SQLite（`modernc`，WAL）
- **前端** SvelteKit SPA（静态导出）+ Tailwind，深色主题
- **数据源** Binance USD-M 合约**只读** API + 自有 `fund.db`
- **部署** 单容器（Go 二进制内嵌 SPA），作为资源受限的共租容器跑在 VPS 上

## 核算方式

NAV 单位法：每笔入金按当时 NAV 铸份额、赎回按当时 NAV 销份额；每 30 分钟拍一次净值快照。
某位朋友的收益 = `份额 × 当前 NAV − 累计净投入`。核心数学见 [`backend/nav`](backend/nav)，改动前务必跑 `TestCanonicalScenario`。

## 本地开发

```sh
# 后端：读 .env（BINANCE_API_KEY/SECRET、JWT_SECRET、FUND_DB_PATH）
go run ./cmd/dashboard            # :8090

# 前端：Vite dev，代理到后端
cd web && npm install && npm run dev   # :3100

# 运维 CLI
go run ./cmd/dashctl status
go run ./cmd/dashctl add-friend <name> <username> <password>
go run ./cmd/dashctl deposit <username> <amount>
```

复制 `.env.example` 为 `.env` 填入只读 Binance key（仅勾选 Enable Reading）。

### 前端演示预览

```sh
cd web
npm run dev:preview              # http://127.0.0.1:3100
```

预览只使用内存中的模拟数据，不连接 Go 服务、Binance 或 `fund.db`。默认进入管理员视图；退出后以用户名 `member` 和任意非空密码登录可查看成员视图，其他用户名进入管理员视图。操作在重启预览后清空。演示中间件仅由显式的 `FUND_PREVIEW=1` 开发模式启用，正常开发和生产构建不启用。

`npm run check` 检查 Svelte / TypeScript；`npm run build` 生成供 Go 内嵌的静态站点。视觉系统约定见 `DESIGN.md`。

## 部署 / 备份

本地构建、VPS 只组装薄镜像（VPS 内存小，**绝不在上面跑构建**）：

```sh
./deploy/deploy.sh               # rsync + 本地构建 + 远端组装重启；默认不动线上 fund.db
./deploy/backup.sh               # 拉一份线上 fund.db 快照到 ./backups
```

线上每 6 小时由 cron 自动备份；账本是真金白银，改动走 PR 心态、先备份。

## 贡献者

- [@xiaxiagaogao](https://github.com/xiaxiagaogao) —— 作者 / 运营
- 🤖 [Claude](https://claude.com/claude-code)（Anthropic）—— 结对编程，见各 commit 的 `Co-authored-by`
