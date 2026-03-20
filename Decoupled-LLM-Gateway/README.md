# Decoupled-LLM-Gateway

**同步网关（上半部）** + **异步治理环占位（下半部）** 的最小可行实现。设计原则：热路径只做字符串与路由；重判断在后台；命中策略或熔断时立即模板降级。

## 仓库布局

| 组件 | 路径 | 说明 |
|------|------|------|
| 同步网关 | `cmd/gateway` | OpenAI 形态 `POST /v1/chat/completions` 入口 |
| Mock LLM | `cmd/echo-llm` | Milestone 1 回声后端，无推理 |
| 契约 | `internal/contracts` | 网关 ↔ 异步环 JSON 结构体 |
| 上下文混淆 | `internal/obfuscate` | UUID / `user_id=` 类剥离（可单测扩展） |
| 诱饵 | `internal/decoy` | 系统消息注入可追踪 token |
| 策略缓存 | `internal/policy` | 进程内 `Store`（Milestone 3 可换 Redis） |
| Worker 桩 | `worker/main.py` | 从 stdin 读 NDJSON 日志 |

## 快速开始

终端 1（回声模型）：

```bash
cd Decoupled-LLM-Gateway
make run-echo
```

终端 2（网关，默认转发到 `http://127.0.0.1:9090`）：

```bash
cd Decoupled-LLM-Gateway
make run-gateway
```

或使用一键脚本（先起 echo，再前台起 gateway）：

```bash
bash scripts/dev.sh
```

调用示例（保留任意 OpenAI 客户端额外字段，仅改写 `messages`）：

```bash
curl -sS http://127.0.0.1:8080/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -d '{"model":"mock","temperature":0.7,"messages":[{"role":"user","content":"hello"}]}'
```

网关会向 **stdout** 打印一行 JSON（`GatewayLogEvent`），便于管道接入 Worker：

```bash
make run-gateway 2>/dev/null | python3 worker/main.py
```

## 环境变量

| 变量 | 默认 | 含义 |
|------|------|------|
| `GATEWAY_LISTEN` | `:8080` | 网关监听地址 |
| `GATEWAY_UPSTREAM` | `http://127.0.0.1:9090` | 上游 OpenAI 兼容 API 根 URL |
| `GATEWAY_UPSTREAM_TIMEOUT_MS` | `30000` | 上游 HTTP 超时 |
| `GATEWAY_MAX_BODY_BYTES` | `4194304` | 请求/响应体上限 |
| `GATEWAY_ASYNC_LOG` | `1` | `0` 关闭事件输出（不推荐） |
| `GATEWAY_POLICY_SEED_FILE` | 空 | 启动时加载策略 JSON 数组（见 `examples/`） |
| `ECHO_LISTEN` | `:9090` | 回声服务监听 |

## 策略种子（防御性降级）

`examples/policy_seed.json` 为演示规则。启动时注入：

```bash
GATEWAY_POLICY_SEED_FILE=examples/policy_seed.json go run ./cmd/gateway
```

当 **混淆后的用户侧快照** 匹配规则中的 `trigger_pattern`（Go `regexp` 语法）且 `action` 为 `DEGRADE_TO_TEMPLATE` 时，网关 **不会** 访问上游，直接返回模板 JSON。

## 里程碑对照

- **M1**：本目录当前状态 — 反向代理路径 + Echo + 诱饵注入 + NDJSON 日志 + CI。
- **M2**：在 `internal/obfuscate` 扩充规则与基准测试（论文中的 ID 对齐 / 规范化）。
- **M3**：Python Worker 消费队列、混合裁判、向 Redis 写入规则并由网关订阅或轮询刷新 `Store`。

## 许可

与父仓库一致；若单独拆库请在此补充 `LICENSE`。
