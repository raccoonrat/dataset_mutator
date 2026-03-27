# P0：真实上游全量主表（与 `run_trackA_full_paper.sh` 预检一致）

## 目标

得到可信的 `results/trackA_full_paper_seed3.json`：经网关路径**不得**再出现 `[echo]`，且 `validate_paper_json.py` 通过。

## 步骤

1. **编辑本仓库根目录 `env`（勿提交）**  
   参考同目录 **`env.example`**：至少设置  
   `GATEWAY_UPSTREAM=https://api.deepseek.com`（或你的 OpenAI 兼容根 URL）  
   以及 `DEEPSEEK_API_KEY` / `OPENAI_API_KEY` 等。

2. **停掉旧网关**，在**新终端**执行：

   ```bash
   cd Decoupled-LLM-Gateway
   set -a && . ./env && set +a
   go run ./cmd/gateway
   ```

   启动日志中应出现 **`upstream https://api.deepseek.com`**（或你的 URL）且 **`upstream_auth=bearer`**。若出现 `upstream http://127.0.0.1:9090`，说明仍在 echo，须改 `env` 后重启。

3. **另一终端**跑全量（含预检、校验、LaTeX 导出）：

   ```bash
   cd Decoupled-LLM-Gateway
   bash experiments/scripts/run_trackA_full_paper.sh
   ```

   - 预检会向网关发一条短请求；若响应含 **`[echo]`**，脚本立即退出（避免再生成混跑 JSON）。  
   - 若需跳过预检（仅调试）：`SKIP_PREFLIGHT=1 bash experiments/scripts/run_trackA_full_paper.sh`（**不推荐**写论文）。  
   - 若只要 JSON、不更新论文表：`RUN_VALIDATE_AND_EXPORT=0 bash experiments/scripts/run_trackA_full_paper.sh`。

4. **确认**

   ```bash
   python3 experiments/scripts/validate_paper_json.py results/trackA_full_paper_seed3.json
   ```

   应无 `ISSUE`；若有，勿用当前 JSON 更新主文表。

## 常见错误

| 现象 | 原因 |
|------|------|
| `PREFLIGHT FAIL: ... [echo]` | 网关进程的 `GATEWAY_UPSTREAM` 仍为 echo；未用 `. ./env` 启动或改后未重启网关 |
| `REFUSE: GATEWAY_UPSTREAM=...9090` | `env` 里上游填成 echo；改为 HTTPS API 根 URL |
| 脚本很慢 | 正常；全矩阵 × 多种子 × 真实 API |

## P1（续）

- **HTTP 裁判子集**：`bash experiments/scripts/run_trackA_p1_http_judge_subset.sh`（需网关已按上文 P0 就绪）。  
- **SmoothLLM K>1 全矩阵**：`bash experiments/scripts/run_trackA_p1_smooth_k5.sh`（成本高；输出与主表 K=1 对照 `smooth_llm` 行）。

## P2

1. **有害集（AdvBench 80）**：`python3 experiments/scripts/fetch_advbench_subset.py -n 80 -o experiments/data/harmful_prompts_trackA_en.txt`，更新附录表 SHA256 后**重跑主表**：`bash experiments/scripts/run_trackA_full_paper.sh`。  
2. **输出守卫消融**：在 `env` 中设置 `GATEWAY_OUTPUT_GUARD_URL=http://127.0.0.1:8766/judge`（及 `GATEWAY_OUTPUT_GUARD_*`），**重启网关**，再 `bash experiments/scripts/run_trackA_p2_output_guard.sh`。主进程网关与裁判 URL 必须一致。
