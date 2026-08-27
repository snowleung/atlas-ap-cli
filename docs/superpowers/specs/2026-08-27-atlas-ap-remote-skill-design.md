# Atlas AP Remote CLI Skill 设计

## 目标

在仓库内创建一个可被 Codex 使用的 `atlas-ap-remote` skill，指导 Codex 通过该 CLI 完成上传、状态查询、结果下载和任务取消。涉及“生成安评/安全评估/安评报告”的请求必须先确认用户提供了配方文件；缺少文件时不得执行上传。

## 设计

skill 采用任务执行型入口，使用清晰的触发描述覆盖 Atlas AP Remote CLI 和安评生成场景。主体包含：意图到子命令的映射、配置优先级、四个命令的调用模板、JSON 输出和退出码处理、单请求不轮询约束、token 保密、下载目录及 ZIP 安全规则。

安评请求的前置流程是：检查配方文件是否存在且可作为本地文件传给 `submit --file`；若没有，提示用户必须上传配方文件并停止；若有，则确认必要业务字段，使用 `--json` 提交并将 `job_id` 交给用户。后续 `status` 和 `download` 只在用户明确要求时执行，不自动轮询。

## 文件边界

- `skill/atlas-ap-remote/SKILL.md`：唯一运行时指令文件，包含完整工作流和 CLI 约束。
- `docs/superpowers/specs/2026-08-27-atlas-ap-remote-skill-design.md`：本设计记录。
- `docs/superpowers/plans/2026-08-27-atlas-ap-remote-skill.md`：落地计划。

不新增脚本、引用资料或资产，因为 README 已提供该 skill 所需的稳定 CLI 合约，额外资源不会提升当前任务的可靠性。

## 验证

使用 `skill-creator/scripts/quick_validate.py` 检查 skill frontmatter、目录命名和脚手架残留；同时人工检查触发条件、配方文件硬性门槛、token 安全和“不自动轮询”规则均已覆盖。
