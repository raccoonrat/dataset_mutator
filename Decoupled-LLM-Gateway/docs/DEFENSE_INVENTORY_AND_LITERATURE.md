# 本项目已实现防御 · 与近年文献对照 · 可扩展方向

## 一、仓库内已实现（代码级）

| 机制 | 位置 | 行为摘要 |
|------|------|----------|
| **PII / 标识符混淆** | `internal/obfuscate/` | 正则流水线：JWT、API key、UUID、邮箱、IPv4、`id_kv` 等 → 占位符；profile：`strict` / `full` / `minimal` / `balanced`；可选 `GATEWAY_OBFUSCATE_RULES_FILE` 追加规则。 |
| **诱饵（decoy）** | `internal/decoy/decoy.go` | 每请求随机 `decoy-…` id，写入 **system** 段（追加或新建 system 消息）。 |
| **同步策略降级** | `internal/policy/policy.go` | 内存正则规则库；命中 `DEGRADE_TO_TEMPLATE` 则 **不调用上游**，直接返回模板 JSON。 |
| **策略热更新** | `internal/policy/redis_refresh.go` | 周期从 Redis Hash 合并规则到 `MemoryStore`。 |
| **网关热路径** | `internal/gateway/gateway.go` | 读 body → `PrepareRequestBody`（混淆快照 + 上游体）→ 策略匹配 → 上游 → 日志 Sink；实验头 `X-Gateway-Experiment-Mode` 控制消融。 |
| **异步闭环（M3）** | `worker/main.py` + `internal/logsink/redis_stream.go` | 消费网关事件；**诱饵泄露**（输出含 `decoy-`）→ 生成 `PolicyRule` → `HSET` 回 Redis；网关侧下次命中则降级。 |
| **评测代理** | `experiments/run_paper_benchmark.py` | 协议 **`paper-eval-3`**：有害单轮 RSR 文件、`harmful_rsr_suite`、可选 HTTP 裁判、HPM 代理、RSR / 抽取 F1、多轮、良性 FPR、并发 stress+SLA；`smooth_llm` 为客户端随机空白扰动（SmoothLLM **思想**的轻量复现）。 |
| **回声上游** | `cmd/echo-llm` | 可控 `X-Echo-*` 行为，用于烟测与 CI。 |

## 二、近年顶会/顶刊相关方向（与实现对齐或可作补充）

下列为 **与「网关 / 提示侧 / 结构化解耦」相邻** 的代表线；**不等同**于本仓库已完整复现其系统，仅便于选题与引用。

| 方向 | 代表工作（请用官方页核对卷期与页码） | 与本项目关系 |
|------|----------------------------------------|--------------|
| **随机平滑 / 扰动集成** | SmoothLLM（ICLR 2024；对提示加扰动再多数表决/聚合） | 评测脚本里 `smooth_llm` 为 **简化客户端扰动**；完整 SmoothLLM 需多采样与聚合逻辑，可后续在脚本或网关侧扩展。 |
| **结构化查询 / 可信与不可信分离** | StruQ（USENIX Security 2024；结构化指令与用户数据分离） | 已增加网关模式 **`structured_wrap`**：对 **user** 轮次在混淆后包上 `[BEGIN_UNTRUSTED_USER]` … `[END_UNTRUSTED_USER]` 再送上游；**策略/日志快照仍用未包 delimiter 的混淆文本**，避免破坏现有规则匹配。 |
| **输入侧内容分类** | Llama Guard、WildGuard 等工业/开放权重护栏模型 | **未内置**；可在网关前增加 **HTTP 微服务** 调用护栏，再决定是否 `DEGRADE` 或转发（建议与现有 `Policy` 并行一层 `InputClassifier` 接口）。 |
| **自提醒 / 系统 sandwich** | 多条工作讨论在 system/user 边界插入固定提醒以降低越狱 | 评测里已有 `strong_system_guard`（强 system 前缀）；可与 `structured_wrap` 组合做消融。 |
| **表征/梯度攻击与防御** | 多条 NeurIPS/ICML 白盒攻击与对齐方法 | 属论文 **Track B**；本网关制品聚焦 **Track A** 黑盒 API，不替代实验室上界实验。 |

## 三、建议的后续落地（按工程量排序）

1. **已完成（本提交）**：`structured_wrap` + 文档；`run_paper_benchmark.py` 增加 `structured_wrap` defense。  
2. **中小**：在脚本中实现 **k 次采样 + 多数票** 的 SmoothLLM 式聚合（仅 echo/小模型试跑，成本敏感）。  
3. **较大**：可选 `internal/classifier/` 接口 + 配置化 HTTP 调用 Llama-Guard 类模型，与 `Policy` 链式组合。  
4. **研究向**：Perplexity/长度异常等 **轻量启发式** 作为 fast-reject（注意误杀与多语言）。

## 四、引用写法建议

在论文中区分：**架构命题**（解耦、异步闭环）由本仓库 **协议与实现**支撑；**与 StruQ / SmoothLLM 等并列**时，应写明本仓库是 **受启发的工程切片** 或 **消融基线**，避免暗示已完整复现第三方系统的全部训练与评测协议。

---

*文件随 `PROTOCOL_VERSION` / 网关行为变更时请同步更新。*
