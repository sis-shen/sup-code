---
name: harness-phase-6-dsh-compat
description: Use when executing Sup Harness 2.0 Phase 6 (dsh 生态兼容 / SPP protocol, dsh-bridge, Cordis interop). Canonical skill at skills/harness-phase-6-dsh-compat/prompts/system.md. Triggers: phase 6, dsh, deepseek harness, cordis, spp.
---

# Sup Harness Phase 6 (opencode adapter)

Canonical instructions: `skills/harness-phase-6-dsh-compat/prompts/system.md`

## Steps
1. Read the canonical file with the Read tool.
2. Read `docs/Sup Harness 2.0 开发规划与 CI-CD.md` Phase 6 and `tasks/phase-6/README.md`.
3. Freeze SPP first, then implement sidecar/tool/LLM mapping in parallel; keep graceful degradation without Node.
4. Fill `tasks/acceptance/phase-6.md`, update `docs/STATUS.md`, run G-compat.

Follow the canonical file exactly.
