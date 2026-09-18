---
description: Start or advance the Sup Harness 2.0 development lifecycle via the orchestrator skill.
agent: build
---

Load and follow the Sup Harness orchestrator skill.

1. Read `.opencode/skills/harness-lifecycle-orchestrator/SKILL.md`, then the canonical
   `skills/harness-lifecycle-orchestrator/prompts/system.md`.
2. Read the authoritative inputs: `docs/STATUS.md`,
   `docs/Sup Harness 2.0 开发规划与 CI-CD.md`, `docs/Sup Harness 2.0 架构重整方案.md`,
   and `tasks/acceptance/phase-*.md`.
3. Determine the next phase, run its precheck (previous gate must be green), then load the
   matching `harness-phase-<n>-<slug>` skill and execute it:
   implement function points, dispatch parallel subagents where the skill allows,
   maintain git commits / dev docs / task prompts, and enforce gates.
4. Update `tasks/acceptance/phase-<n>.md` and `docs/STATUS.md`; tag the phase version when the gate passes.

Optional focus: $ARGUMENTS

Do not skip gates. If the previous phase is not finished, enter the fix loop instead.
