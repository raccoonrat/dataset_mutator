# Track A 正式论文跑分清单（paper-eval-5 / 主表）

按顺序执行；与前序小规模烟测共用同一 `Decoupled-LLM-Gateway` 根目录与 `./env`。

## 0. 环境与仓库

- [ ] `cd Decoupled-LLM-Gateway && . ./env`（网关上游密钥、`OPENROUTER_API_KEY` 等）
- [ ] 安装 benchmark 依赖：`bash experiments/scripts/install_benchmark_requirements.sh`（**推荐**；避免在已导出 `HTTPS_PROXY`/`ALL_PROXY` 为 SOCKS 时直接 `pip install`，否则会报 *Missing dependencies for SOCKS support*）。含 **PySocks** + **requests** + **datasets** 等。
- [ ] 若本机访问 Hub 需代理：在 `env` 中设置 **`TRACKA_HF_SOCKS5=127.0.0.1:1080`**（或 **`HF_SOCKS5_PROXY`**；可与 **`JUDGE_SOCKS5_PROXY`** 同值）。`run_trackA_full_paper.sh` / `fetch_all_paper_datasets.sh` 会导出 `HTTPS_PROXY`/`ALL_PROXY` 并保留 **`NO_PROXY` 含 127.0.0.1**，避免网关与本地 judge 误走 SOCKS。已自行设置 **`HTTPS_PROXY`** / **`ALL_PROXY`** 时不会覆盖。
- [ ] 网关进程：`GATEWAY_UPSTREAM` 指向**真实** API，且非 echo（`paper_preflight_non_echo` 会通过）

## 1. LG4 HTTP 裁判（主表默认）

- [ ] 终端 A：`bash experiments/scripts/run_openrouter_llama_guard4_judge.sh`（默认 `JUDGE_SOCKS5_PROXY=127.0.0.1:1080`，按需覆盖）。脚本会在需要时 **无代理地** `pip install -r experiments/judge_service/requirements.txt`（PySocks+requests）。若你**不用**该脚本、手动 `python3 experiments/judge_service/server.py`，请在**同一 Python** 下执行：`env -u HTTPS_PROXY -u ALL_PROXY pip install -r experiments/judge_service/requirements.txt`
- [ ] 跑分终端：`export PAPER_EVAL_JUDGE_URL=http://127.0.0.1:8765/judge`（端口与 `JUDGE_PORT` 一致）
- [ ] 可选：`export JUDGE_MODEL_REVISION=…` 写入 `manifest.eval_judge_chat`

## 2. 数据集（与 full 脚本一致）

- [ ] `bash experiments/scripts/run_trackA_full_paper.sh` 会在未设 `SKIP_FETCH_DATASETS=1` 时拉取 JBB / StrongREJECT / wild；或事先手动 fetch，保证 `experiments/data/` 下文件齐全

## 3. 全量主表

- [ ] `bash experiments/scripts/run_trackA_full_paper.sh`  
  - 默认：`TRACKA_JUDGE_MODE=http`、`TRACKA_PROMPT_WORKERS=4`、按 defense 粒度 checkpoint  
- [ ] 中断续跑：`RESUME=1 OUT=results/同一路径.json`（checkpoint 与 fingerprint 需一致）

## 4. 收尾

- [ ] `python3 experiments/scripts/validate_paper_json.py results/….json`
- [ ] 脚本成功时会更新 `paper/generated/trackA_main_table*.tex`（若未设 `RUN_VALIDATE_AND_EXPORT=0`）
- [ ] 归档：`manifest`、`GIT_SHA`/`EVAL_HOSTNAME`（可选 export）、输出 JSON 与 checkpoint 删除策略（跑完全量后 checkpoint 会自动删）

## 5. 小规模烟测（正式跑前）

- [ ] `bash experiments/scripts/run_trackA_smoke_main_table.sh`（默认本地 heuristic judge；LG4 见该脚本头部说明）
