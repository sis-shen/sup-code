# Pull Request

## 阶段 / Phase
- [ ] Phase 0 工程地基与基线冻结
- [ ] Phase 1 Cordis 内核 core
- [ ] Phase 2 叶子插件化
- [ ] Phase 3 状态与交互
- [ ] Phase 4 心脏迁移（Agent）
- [ ] Phase 5 双子进程与远程
- [ ] Phase 6 dsh 生态兼容
- [ ] Phase 7 收尾与发布
- [ ] 非阶段（chore / docs / hotfix）

## 变更说明
<!-- 做了什么，为什么 -->

## 影响范围
- 影响的目录/包：
- 是否改动 `core/`：
- 是否涉及接口/事件契约变更：

## 门禁自检（本地）
- [ ] `go build ./... && go vet ./...`
- [ ] `bash scripts/lint.sh`
- [ ] `go test ./...`（对照 `docs/baseline-v1.md`，无新增失败）
- [ ] `bash scripts/check_arch.sh`
- [ ] `bash scripts/check_skills.sh`
- [ ] `bash scripts/check_task_checkboxes.sh`

## 文档 / 任务
- [ ] 相关 `docs/*` 已更新
- [ ] `tasks/phase-N/README.md` 已勾选
- [ ] `tasks/acceptance/phase-N.md` 已填写
- [ ] 关键决策已写入 `docs/adr/`

## 备注 / 推迟项
<!-- 移入 2.1+ Backlog 的内容请注明 -->
