# WI-20261019-allcli-qa-manual

- Status: 审查
- Repo / Issue / PR: oldwinter/all-cli / 无 Issue / PR 见本分支 `mayi/qa-manual-dispatch`
- Owner seat: 码一 · Desktop Eng
- Claimed-by: 码一（阿冬 2026-10-19 08:38 +08:00 指派）
- 用户决定: 2026-10-19 01:33 +08:00，用户定下 functional-qa 只在手动触发（`workflow_dispatch`）时运行；缺 key 时不能再在 PR 上显示成跳过的“绿”。合入由用户决定。
- 现象: #135 合入后，`.github/workflows/qa.yml` 仍由 `pull_request`（opened / synchronize / reopened）触发。仓库没有 `FACTORY_API_KEY`，所以每个 PR 上都会出现一个 functional-qa check：precheck 跳过 8 个步骤后结束，显示成绿色。看起来像 QA 过了，其实 QA 根本没跑。
- 想改成啥:
  1. 触发器只保留 `workflow_dispatch`，去掉 `pull_request`（原来就没有 `push`），PR 上不再出现 functional-qa check。
  2. 只在 PR 场景运行的步骤删掉：Resolve PR metadata、Apply skill updates from QA、Commit skill updates、Post QA report as PR comment。没有 PR 触发以后它们永远不会执行，Apply 步骤改了文件也没有后续步骤提交。`qa-results/`（包括 `skill-updates.json`）仍然作为 artifact 上传。
  3. 权限从 `contents/pull-requests/issues: write` 收紧到 `contents: read`；checkout 加 `persist-credentials: false`；concurrency group 改成 `qa-${{ github.ref }}`。
  4. 保留 FACTORY_API_KEY precheck。手动运行时缺 key：**让这次运行明确失败**（`::error`、step summary 和 report.md 都写 “NOT RUN - FACTORY_API_KEY missing”，然后 `exit 1`）。
- 为什么缺 key 选“失败”而不是“跳过”: 手动运行是有人明确要跑 QA。缺 key 时 QA 没跑，结果如果是绿色，又会造成用户 01:33 要消除的那种“跳过当通过”的误读。红色加明确的错误文案，直接告诉触发的人去补 secret 再重跑。PR 上已经不会出现这个 check，所以红色不会挡任何 PR，也不会影响分支保护。
- `continue-on-error` 已去掉（阿冬 2026-10-19 09:38 +08:00，不需要用户再拍板）: `Run QA` 步骤原来带 `continue-on-error: true`，QA 自己失败时整次运行仍然显示绿色。用户的长期规则是：证据缺失必须明确标成跳过，不能显示成绿色；QA 失败却显示绿色是同一类问题，所以删掉，让 QA 失败时运行报红。
- 验收:
  1. `qa.yml` 的 `on:` 只有 `workflow_dispatch`；本 PR 和之后的 PR 上不出现 functional-qa check。
  2. actionlint 1.7.7 对 `qa.yml` 和全部 workflow 都是 0 个问题。
  3. 手动运行缺 key：precheck 报 `::error`，step summary 写明 QA not run，运行结果为失败；后面步骤全部跳过，`qa-results/report.md` 作为 artifact 上传。
  4. 有 key 时 QA 自身失败，整次运行报红（`Run QA` 不再有 `continue-on-error`）。
5. 分支保护的 required checks 不包含 functional-qa（只读核对：main 只要求 `test`，没有 rulesets）。
- In / Out:
  - In: `.github/workflows/qa.yml`、本 WI。
  - Out: 不加 secret，不改分支保护和仓库设置。
- Blocked 等人: 合入要用户批准。
- Evidence: 见 PR 描述（actionlint 输出、分支保护只读结果）。
- Unfinished: 等码审 → 用户批准 → 仓清合并。合入后可以手动触发一次，确认缺 key 时报红。
