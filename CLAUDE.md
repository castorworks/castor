# CLAUDE.md

@AGENTS.md

## Claude Code notes

- Working under `apps/api/` or `apps/web/` automatically loads that directory's CLAUDE.md (which references its AGENTS.md).
- After a change, run `python3 scripts/check.py <api|web|scripts>` for the affected parts and report completion only once they all pass.
- Cross-stack exploration can fan out to parallel subagents; implementation still follows the section 5 checklist of AGENTS.md layer by layer.
