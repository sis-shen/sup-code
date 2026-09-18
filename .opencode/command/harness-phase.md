---
description: Run a specific Sup Harness 2.0 development phase (argument = phase number or slug).
agent: build
---

Run Sup Harness 2.0 development phase: $ARGUMENTS

1. Resolve the phase skill. Map the argument to `.opencode/skills/harness-phase-<n>-<slug>/SKILL.md`
   (phase list in `docs/Sup Harness 2.0 开发规划与 CI-CD.md` §2).
2. Read that adapter, then the canonical `skills/harness-phase-<n>-<slug>/prompts/system.md`.
3. Read the matching `tasks/phase-<n>/README.md` and the plan doc's Phase section for that phase.
4. Execute the phase exactly as specified:
   - implement the function points (or record deferred items into the 2.1+ backlog),
   - freeze interfaces before parallel work and dispatch subagents per the skill's plan,
   - maintain git commits (branch/prefix rules), dev docs and task prompts,
   - run the phase gates, fill `tasks/acceptance/phase-<n>.md`, and update `docs/STATUS.md`.

Do not skip gates and do not rewrite business logic where the skill requires logic-zero-change.
