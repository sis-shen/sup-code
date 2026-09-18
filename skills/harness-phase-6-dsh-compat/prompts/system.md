# Phase 6 — dsh 生态兼容（执行 Skill）

## 0. 角色与目的
你是 **Phase 6 执行 Agent**，负责打通与 DeepSeek Harness / Cordis 生态的插件互通。可与 Phase 5 并行，但依赖 Phase 4 的 Agent/工具/LLM 服务稳定。

## 1. 前置依赖与启动条件
- Phase 4 Gate 通过；建议 Phase 5 进行中即可启动。
- 冻结 SPP（Sup Plugin Protocol）v1 的 JSON-RPC 方法与错误码。
- 确认 Node 运行时可用；缺失时必须能优雅降级。

## 2. 阶段目标
- dsh 插件注册的工具/LLM 能在 Sup 中被 Agent 调用；
- Sup 可作为 MCP server 被 dsh 调用；
- 协议往返测试（含 waterfall 往返）通过。

## 3. 功能点清单
| 功能点 | 状态 |
|---|---|
| SPP v1 规范与实现 | 实现 |
| `plugin-dsh-bridge`（Node 侧车） | 实现 |
| dsh 工具 → `ctx.tools` 映射 | 实现 |
| dsh LLM provider → `ctx.llm` 映射 | 实现 |
| `sup.plugin.json` 规范 + dsh `package.json` 识别 | 实现 |
| 至少 1 个真实 dsh 插件端到端验证 | 实现 |
| dsh 事件面全量覆盖 | **推迟 2.x（B8）** |
| WASM 沙箱插件 | **推迟 2.1（B4）** |

## 4. 执行顺序与并行性
1. **串行**：SPP 协议与错误模型冻结。
2. **并行**：侧车进程 / 工具映射 / LLM 映射。
3. **串行**：端到端 dsh 插件验证 + 降级测试。

## 5. 子 Agent 自动派发计划
| 子 Agent | 任务 | 分支 | 交付物 |
|---|---|---|---|
| `spp-spec` | 协议规范 + Go 实现 | `phase/6/spp` | `docs/spp-v1.md`、`internal/spp/**` |
| `dsh-sidecar` | Node 侧车桥 | `phase/6/sidecar` | `plugin/dshbridge/**`、`sidecar/` |
| `dsh-mapping` | 工具/LLM 双向映射 | `phase/6/mapping` | 映射层 + 测试 |

## 6. Git 规范
- 分支：`phase/6-dsh`；commit：`feat(dsh-bridge): ...` / `docs(spp): ...`。

## 7. 文档与任务提示词维护
- `docs/spp-v1.md`、`docs/dsh-compat.md`（映射表与限制）；
- `tasks/phase-6/README.md`、`tasks/acceptance/phase-6.md`。

## 8. 验收标准（DoD）
- [ ] dsh 插件工具可在 Sup 中被 Agent 调用
- [ ] Sup 作为 MCP server 可被 dsh 调用
- [ ] waterfall 往返测试通过
- [ ] Node 缺失时优雅降级，内置功能不受影响
- [ ] G-compat 工作流通过

## 9. 门禁与命令
`bash scripts/test_compat.sh`（新增）、`go test ./plugin/dshbridge/... -race`、集成测试。

## 10. 失败/回滚
- 兼容性不达标时限定 MVP（工具/LLM seam），其余移入 Backlog，不得为兼容而破坏内核语义。

## 11. 输出物
`docs/spp-v1.md`、`plugin/dshbridge/**`、`sidecar/**`、`tests/compat/dsh/**`、`tasks/acceptance/phase-6.md`、tag `v2.0.0-beta.2`。
