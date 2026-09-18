#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""gen_phase_tasks.py — 生成 Sup Harness 2.0 阶段任务看板与验收报告骨架.

Usage:
    python scripts/gen_phase_tasks.py [--force]

Reads scripts/phase_manifest.json and writes:
    tasks/phase-<n>/README.md
    tasks/acceptance/phase-<n>.md
Existing files are preserved unless --force is given.
"""
import argparse
import json
import os
from datetime import date

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
MANIFEST = os.path.join(ROOT, "scripts", "phase_manifest.json")

BOARD_TMPL = """# Phase {n} — {name} 任务看板

- 执行 Skill：`{skill}`
- 版本：{ver}
- 前置阶段：{prev}
- 集成分支：`{branch}`
- 验收报告：`tasks/acceptance/phase-{n}.md`
- 规划文档：`docs/Sup Harness 2.0 开发规划与 CI-CD.md`

## 功能点
{fps}

## 子 Agent 派发

| 子 Agent | 分支 | 状态 | 交付物 |
|---|---|---|---|
| （见阶段 Skill） | | ⬜ | |

## 门禁
- [ ] G0 编译 `go build ./... && go vet ./...`
- [ ] G1 静态检查 `bash scripts/lint.sh`
- [ ] G2 单元测试 `go test ./...`
- [ ] G3 竞态 `bash scripts/test.sh`
- [ ] G4 覆盖率（新增代码 ≥80%）
- [ ] G5 架构 `bash scripts/check_arch.sh`
- [ ] G6 Skill `bash scripts/check_skills.sh`
- [ ] G7 文档/任务 `bash scripts/check_task_checkboxes.sh`

## 文档维护
- [ ] 相关 `docs/*` 已更新
- [ ] 关键决策写入 `docs/adr/`
- [ ] 验收报告 `tasks/acceptance/phase-{n}.md` 已填写并勾选 DoD
"""

REPORT_TMPL = """# Phase {n} — {name} 验收报告（{ver}）

> 状态：**NOT STARTED**
> 执行 Skill：`{skill}`
> 分支：`{branch}`
> 生成时间：{today}

## 1. 交付物清单

| 交付物 | 路径 | 是否存在 |
|---|---|---|
| （填写） | | |

## 2. 门禁结果

| 门禁 | 命令 | 结果 |
|---|---|---|
| G0 编译 | `go build ./... && go vet ./...` | ⬜ |
| G1 静态检查 | `bash scripts/lint.sh` | ⬜ |
| G2 单元测试 | `go test ./...` | ⬜ |
| G3 竞态 | `bash scripts/test.sh` | ⬜ |
| G4 覆盖率 | `diff-cover … --fail-under=80` | ⬜ |
| G5 架构 | `bash scripts/check_arch.sh` | ⬜ |
| G6 Skill | `bash scripts/check_skills.sh` | ⬜ |
| G7 文档/任务 | `bash scripts/check_task_checkboxes.sh` | ⬜ |

## 3. 验收标准（DoD）

- [ ] （从阶段 Skill 的 DoD 复制）

## 4. 证据

<!-- 命令输出、CI 链接、git diff --stat、覆盖率报告 -->

## 5. 推迟 / Backlog

| 项 | 原因 | 目标版本 |
|---|---|---|
| | | |

## 6. 结论

- 状态：**PENDING**
- 签署：
"""


def write(path: str, content: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w", encoding="utf-8", newline="\n") as fh:
        fh.write(content)


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument("--force", action="store_true", help="overwrite existing files")
    args = ap.parse_args()

    with open(MANIFEST, encoding="utf-8") as fh:
        data = json.load(fh)

    today = date.today().isoformat()
    created = 0
    for p in data["phases"]:
        board = os.path.join(ROOT, "tasks", f"phase-{p['n']}", "README.md")
        report = os.path.join(ROOT, "tasks", "acceptance", f"phase-{p['n']}.md")
        if not args.force and os.path.exists(board) and os.path.exists(report):
            print(f"skip   phase-{p['n']} (exists)")
            continue

        fps = "\n".join(f"- [ ] {x}" for x in p["fps"])
        write(board, BOARD_TMPL.format(n=p["n"], name=p["name"], skill=p["skill"],
                                       ver=p["ver"], prev=p["prev"], branch=p["branch"], fps=fps))
        write(report, REPORT_TMPL.format(n=p["n"], name=p["name"], skill=p["skill"],
                                         ver=p["ver"], branch=p["branch"], today=today))
        created += 1
        print(f"write  tasks/phase-{p['n']}/README.md, tasks/acceptance/phase-{p['n']}.md")

    print(f"done. {created} phase(s) generated.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
