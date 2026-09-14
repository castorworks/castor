# CLAUDE.md

@AGENTS.md

## Claude Code 补充

- 在 `apps/api/` 或 `apps/web/` 下工作时，会自动加载该目录的 CLAUDE.md（引用其 AGENTS.md）。
- 完成改动后运行 `scripts/check.sh <api|web|scripts>` 验证受影响部分，全部通过后再报告完成。
- 跨前后端的探索可以并行派发子代理；实现时仍须按 AGENTS.md 第 5 节清单逐层落地。
