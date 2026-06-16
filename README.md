# Fund Dashboard

朋友基金净值看板 —— 用 **NAV 单位法**核算几位朋友在我 Binance 合约账户里的份额、收益与回撤，并向他们透明展示交易。

## 功能

**朋友视图（移动优先）**
- 我的估值 / PnL、净值曲线（「我的 / 基金」切换）
- 当前持仓双环：资金配置（保证金占用 vs 闲置现金 + 全仓杠杆）、名义分布
- 折叠的复盘明细（平仓记录、按标的盈亏）与成员对比

**复盘视图（仅 admin）**
- 净值曲线 + 水下回撤（最大 / 当前回撤）
- 每日盈亏日历热力图
- 胜率、赢赔比、平均持仓时长、当前持仓、平仓记录、按标的盈亏

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
