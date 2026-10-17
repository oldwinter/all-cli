# 代码改进评审与修复报告（2026-10）

对 `all-cli` 全仓库的一次系统性评审，以及基于该评审完成的修复。

- **评审部分**：所有结论都在本机（Windows amd64 / Go 1.27.0）实际构建并运行验证，不是静态推测。
- **修复部分**：全部修复已落地，并有回归测试锁定；最终 `gofmt -l .` 为 0 文件、
  `go build ./...` / `go vet ./...` 通过、`go test ./...` 全部 26 个包通过。

评审基线：130 个 Go 文件、约 17.9k 行；2 个直接依赖（cobra + jsonschema）；
46 个内置工具、15 个分类、16 个 adapter 包。

---

## 修复总览

| 编号 | 问题 | 状态 | 验证方式 |
|---|---|---|---|
| P0-1 | `--timeout` 不是硬超时，可挂起数分钟 | ✅ 已修 | 孙进程持 pipe 测试：6.05s → 2.02s |
| P0-2 | 遥测把错误原文（含凭据/主机名/路径）发给 Sentry | ✅ 已修 | 8 组 fixture 断言线上载荷不含敏感串 |
| P0-3 | 遥测配置错误让整个 CLI 退出 1 | ✅ 已修 | 坏 `POSTHOG_HOST` → exit 0 + 警告 |
| P0-4 | 遥测同步阻塞退出，最坏 4s | ✅ 已修 | 慢 sink 测试：不再超过 200ms 预算 |
| P0-5 | Windows 上 `gofmt -l .` 报全部 130 文件 | ✅ 已修 | `gofmt -l .` → 0 文件 |
| P1-1 | 每个工具重复执行同一子进程 | ✅ 已修 | 31 → 21 次调用，重复 10 → 0（-32%） |
| P1-2 | adapter 跨 runner 复用（数据竞争 + 陈旧状态） | ✅ 已修 | 第二个 runner 现在真正被调用 |
| P1-3 | `--timeout` 被 per-tool 默认值覆盖 | ✅ 已修 | 12096ms → 50ms |
| P1-4 | `diagnose` 建议的命令不存在 | ✅ 已修 | 2 条建议，0 条不可解析 |
| P1-5 | per-tool `status --json` 不符合自己的 schema | ✅ 已修 | `kubectl status --json \| diff` exit 0 |
| P1-6 | `--installed-only` 丢掉 95% 诊断 | ✅ 已修 | 诊断数 39 → 39（原 39 → 2） |
| P1-7 | 17 处内联打印绕过 `printDiagnostics`，消息重复 | ✅ 已修 | 重复消息消失，删 ~34 行 |
| P1-8 | `use` 失败同时写 stdout 和 stderr | ✅ 已修 | JSON 模式 stderr 为空 |
| P1-9 | `docker update` 退出码随输出格式变化 | ✅ 已修 | 两种格式均 exit 1 |
| P1-10 | `filterStatusTools` 原地改写调用方切片 | ✅ 已修 | 改为分配新切片 |
| P1-11 | CJK 列宽错位 | ⚠️ 未修（见"遗留"） | — |
| P1-12/13 | `NO_COLOR` 对转发输出无效 / JSON 混入 ANSI | ✅ 已修 | `execx` 单点剥离 + 表驱动测试 |
| P1-14 | 缺 `fix-plan` / `snapshot-diff` schema | ✅ 已修 | 4 个 schema 均可打印 |
| P1-15 | diagnostic schema 无法单独使用 | ✅ 已修 | 内联 `$defs`，独立编译测试 |
| P1-16 | 覆盖率闸门单包插桩，数字失真 | ✅ 已修 | `-coverpkg`，89.4% |
| P1-17 | 测试辅助泄漏 `os.Args` 给 cobra | ✅ 已修 | `-coverpkg` 下不再失败 |
| P1-18 | 4 个 Windows 测试失败 + rclone 产品 bug | ✅ 已修 | 全部 26 包在 Windows 通过 |
| P2-1/2/6 | 9 个重复 `status`、38/51 缺 `Args` | ✅ 已修 | 净删 ~78 行；多余参数被拒绝 |
| P2-3/4/5/7/8/9/10 | registry 表驱动、元数据漂移、分组、`--dry-run`、JSON 信封、快照过滤、TAB | ✅ 部分修（见"遗留"） | — |
| P3-1..13 | CI/卫生/文档类 | ✅ 部分修（见"遗留"） | — |

---

## 已修复问题详解

### P0-1 `--timeout` 不是硬超时（最严重）

`internal/execx/execx.go` 把 `bytes.Buffer` 接到 `cmd.Stdout/Stderr`。
当输出端不是 `*os.File` 时，`os/exec` 会创建 pipe 并起 copy goroutine，
而 `cmd.Run()` **必须等它结束**。如果子进程派生的孙进程继承了写端
（shim / `.cmd` 包装器 / kubectl 插件 / docker credential helper 都这样），
父进程被 kill 后 pipe 仍开着，`Run()` 就一直阻塞。

**修复**：`cmd.WaitDelay = 2 * time.Second`（`execx.go`）。
这是标准库为这个场景提供的开关。

**验证**：新增 `TestDefaultRunnerWaitDelayBoundsGrandchildPipe`——
构造一个派生孙进程持有 stdout 6 秒的 shim，断言 `Run()` 在 6 秒内返回。
实测从 6.05s 降到 2.02s（超时被正确上报而非吞掉）。

### P0-2 遥测隐私契约与实现不符

`docs/observability.md` 声称排除命令参数、环境变量、外部输出、用户名、
账号 ID、配置值、路径七类数据。实际只有黑名单正则
（`sentry.go` 的 `secretPattern`），实测漏掉：

| 输入 | 修复前洗脱结果 | 泄露 |
|---|---|---|
| `Authorization: Bearer <token>` | `Authorization=<redacted> <token>` | JWT（正则遇空格即停） |
| `{"password":"<value>"}` | 原样 | JSON 形式密钥 |
| `https://<user>:<password>@host/api` | `https://<user>:<password>@host<path>` | URL userinfo 凭据 |
| `not logged in to github.acme-corp.internal` | 原样 | 私有主机名 |
| `C:\Users\John Doe\private\x.json` | `C:\Users\<path> Doe\private\x.json` | 用户名与路径 |

根因不是正则不够强，而是**根本不该转发 `err.Error()`**：
adapter 主动把 context 名、namespace、文件路径写进 error 文本。

**修复**：
1. 网络侧改发**闭集分类**（`errorClass`）：`timeout` / `canceled` /
   `exit_nonzero` / `not_found` / `permission_denied` / `invalid_usage` /
   `invalid_snapshot` / `network` / `command_failed` / `error`。
   完整错误原文只留在本机 `ALL_CLI_LOG_PATH` 的结构化日志里。
2. `scrubError` 重写为"凭据脱敏 → 路径移除 → 保守白名单"三阶段，
   保留给本地日志用。

**验证**：`TestSentryNeverTransmitsSensitiveContent` 用 8 组真实 fixture
（JWT、JSON 密钥、AWS key、URL userinfo、私有主机名、Windows 路径含空格、
POSIX 家目录、kubectl context 名），跑真实 HTTP sink 捕获线上载荷，
断言敏感串一个都不出现。

### P0-3 遥测初始化失败让 CLI 无法使用

`telemetry.New` 对坏 `POSTHOG_HOST` 返回 error，`Execute` 在执行命令前就返回它。
实测：`POSTHOG_HOST="not a url" all-cli version` → exit 1，什么都不输出。
这直接违反文档"telemetry 问题不改变退出状态"。

**修复**：`New` 改为降级——出错的 sink 置 `nil`，记入 `Warnings()`，
由 `Execute` 打到 stderr；命令照常执行。

**验证**：现在 exit 0，输出正常版本号，stderr 一行
`warning: posthog disabled: parse POSTHOG_HOST "not a url"`。

### P0-4 遥测同步阻塞退出

两个 sink 内联在 `Finish` 里，各 2 秒超时。实测黑洞地址下
`diff` 失败路径耗时 **4,062ms**。

**修复**：远端 sink 改为并发 + **共享 200ms 预算**，
并用 `context.WithoutCancel` 解耦命令生命周期
（否则 context 已取消时遥测静默丢失）。

**验证**：`TestFinishDoesNotBlockOnSlowSink` 用 2 秒慢 sink，
断言 `Finish` 在 1 秒内返回；`TestFinishDeliversWhenParentContextCancelled`
断言父 context 取消后仍能投递。

### P0-5 Windows 上格式检查根本无法通过

仓库没有 `.gitattributes`，而 Git for Windows 默认 `core.autocrlf=true`，
工作区是 CRLF、索引是 LF，`gofmt` 视 CRLF 为未格式化 →
**`gofmt -l .` 报出全部 130/130 个文件**，`just ci` / `fmt-check` / pre-commit
在任何默认 Windows Git 安装上都无法通过。

**修复**：新增 `.gitattributes`（`* text=auto eol=lf` + 各类型 `eol=lf`），
并刷新工作区为 LF。

**验证**：`gofmt -l .` → **0 文件**。这是本次收益最高的单点改动。

### P1-1 每个工具重复执行同一子进程

`Configured` 与 `Current` 各跑一次同样的命令，13/15 个 adapter 都是这个形状。

**修复**：在 `Evaluate` 内安装一个 per-evaluation 的记忆化 runner，
对相同 `(name, args)` 重放结果。作用域随 `Evaluate` 创建，
不会跨评估/runner 泄漏。

**验证**：`TestEvaluateSubprocessDuplicateMeasurement` 实测
**31 → 21 次调用，重复 10 → 0（-32%）**。
其中 aliyun 2→1、wrangler 4→2、vercel/railway/netlify/gh/kargo/argocd/opencli 各 2→1；
aws/kubectl/docker/glab 本就非重复（两个回调发的是不同命令）。

### P1-2 adapter 跨 runner 复用

`toolFromAdapter` 用闭包捕获 `adapter` 变量，`DefaultRegistry()` 返回结构体副本
但闭包共享同一变量，导致第二次用不同 runner 评估时**零次调用**；
且 `if adapter == nil { adapter = ... }` 是未同步的 check-then-write。

**修复**：不再捕获，每个回调用传入的 runner 现建 adapter（adapter 是无状态
结构体，只有 runner 字段，代价为零）。同时删除 `registry_cloud.go` 里
跨评估的 `whoamiOnce` 缓存（它正是陈旧身份的来源）。

### P1-3 `--timeout` 被静默覆盖

`runnerForTool` 无条件用 `def.Timeout` 覆盖用户值，实测
`wrangler status --timeout 100ms` 耗时 **12,096ms**。

**修复**：`rootOptions` 增加 `TimeoutExplicit`（由 `PersistentPreRunE` 里的
`cmd.Flags().Changed("timeout")` 设置），`def.Timeout` 降级为默认值。

**验证**：同一命令现在 **50ms**。原先把这个 bug 固化成正确预期的测试已改写。

### P1-4 `diagnose` 建议的命令不存在

`diagnose` 生成 `all-cli <tool> status` / `all-cli <tool> list`，
但 vercel/railway/netlify/mise/k9s/wrangler/aws/aliyun 没有 `status` 子命令，
mise/k9s 也没有 `list`。实测 `all-cli vercel status` → exit 1；
`all-cli mise list` → 打印 help 但 **exit 0**（静默空操作，对 agent 更危险）。

**修复**：统一改为唯一对所有工具都成立的 `all-cli status --tools <id>`，
并加解析器测试（比 `root.Find` 更严格，还会拒绝"父命令打印 help 后 exit 0"）。

**验证**：全部诊断共 2 条建议，0 条不可解析。

### P1-5 per-tool `status --json` 不符合自己的 schema

输出裸 `ToolSummary`，没有 `schema_version`，导致
`kubectl status --json | all-cli diff -` 失败，也不满足
`schemas/status-report-v0.1.json` 的 required 字段。

**修复**：改为输出完整的单工具 `StatusReport`。

**验证**：`kubectl status --json` 现含 `schema_version`，
且可正常喂给 `diff`（exit 0）。

### P1-6 `--installed-only` 静默丢弃 95% 诊断

诊断在 `report.Tools` 被过滤之后才生成，被过滤掉的工具不产生诊断。

**修复**：基于**未过滤**的 report 生成诊断，只过滤 `Tools`。

**验证**：全量 tools=46 / diag=39；`--installed-only` tools=9 / diag=**39**（原为 2）。

### P1-7 / P1-8 / P1-9 / P2-8 输出一致性

- 17 处内联 warning/error 打印循环改为 `printDiagnostics`，删 ~34 行，
  修掉"一次失败打印两条消息"（`docker list` 现在只出现一次）。
- `use` 失败在 7 处同时写 stdout 和 stderr；改为统一 `reportUseFailure`，
  JSON 模式 stderr 为空。
- `docker update` 退出码此前 JSON 模式 0、纯文本模式 1；现在两者一致（均 1），
  且 JSON 先输出 payload 再返回错误。
- 18 处手工拼装的 `{current, warnings, errors}` 改为 `currentEnvelope` +
  `nonNil`，`warnings` 不再时 `null` 时 `[]`（`kubectl current --json` 现在稳定输出 `[]`）。

### P1-12/13 ANSI 转义泄漏

外部 CLI 即使输出被重定向也会上色，`NO_COLOR=1` 下仍会把 `\x1b[31m` 转发出去，
并混入 `--json` 输出。

**修复**：在 `execx.DefaultRunner` 捕获处单点剥离 CSI/OSC/两字节转义。

**验证**：`TestStripANSI` 7 组用例 + `TestDefaultRunnerStripsANSIFromCapturedOutput`。

### P1-14/15 Schema 契约补齐

新增 `schemas/fix-plan-v0.1.json` 与 `schemas/snapshot-diff-v0.1.json`，
注册到 `schemas/embed.go`，`all-cli schema` 现支持四个名字。
同时把 `diagnostic-report-v0.1.json` 的跨文件 `$ref` 内联，
使 `all-cli schema diagnostic > file.json` 产物可**独立离线编译**。

**验证**：`TestEverySchemaCompilesStandalone` 用一个禁止外部解析的 loader
编译每个 schema——任何相对 `$ref` 都会失败。

### P1-16 覆盖率闸门口径错误

`go test -coverprofile ./...` 只对每个包自己的测试插桩，跨包执行不可见。
实测 `internal/diagnose` 在闸门里读 49.1%，但从 cli 测试看是 78.9%。

**修复**：`scripts/check-coverage.sh` 改用 `-coverpkg=./...`，
并修掉该脚本在 Windows/WSL 下找不到 `go` 的问题。

**验证**：闸门现在读 **89.4%**（原口径 85.48%），仍高于 80% 下限。

### P1-18 跨平台测试 + rclone 产品 bug

三个测试依赖 POSIX 语义：`/dev/null`、`echo`/`false`/`sleep` 二进制、
以及 `t.Setenv("HOME")`（Windows 上 `os.UserHomeDir()` 读 `USERPROFILE`）。

**修复**：`os.DevNull`（Windows 上跳过字符设备断言）、
可移植的 `echoCommand`/`failingCommand`/`sleepingCommand` 辅助函数、
以及把 rclone 测试改为完全不依赖环境。

**产品 bug**：`rcloneConfigured()` 只查 `~/.config/rclone/` 和 `~/.rclone.conf`。
Windows 真实路径是 `%APPDATA%\rclone\rclone.conf`，macOS 是
`~/Library/Application Support/rclone/rclone.conf` —— **两个平台的用户
配好了 rclone 也被报未配置**。

**修复**：拆成 `rcloneConfigCandidates()`（优先 `os.UserConfigDir()`）/
`rcloneConfiguredIn()`（纯函数）/ `rcloneConfigured()`。

**验证**：本机候选路径首位为
`C:\Users\Administrator\AppData\Roaming\rclone\rclone.conf`；
`TestRcloneConfigCandidatesPreferOSConfigDir` 锁定该顺序。

### P2-1/2/6 命令层重复与参数校验

- 9 个逐字节相同的 `status` 子命令（163 行）收敛为一个 builder。
- 38/51 个命令缺 `Args` 校验，多余位置参数被静默忽略
  （`all-cli status extra` → exit 0）。现全部零参叶子加 `cobra.NoArgs`；
  `catalog`/`describe`/`use` 等确有位置参数的命令保持不变。
- 净删约 78 行生产代码。

**验证**：`TestZeroArgCommandsRejectExtraArgs`（33 条命令路径）+
`TestCommandsWithOptionalArgsStillAcceptThem`（4 条正向用例）。

---

## 遗留问题（本次未修，已评估）

| 编号 | 问题 | 原因 | 建议 |
|---|---|---|---|
| P1-11 | CJK 列宽错位（`tabwriter` 按 rune 计数） | 需要东亚宽度感知的辅助函数与全部表格改造，且当前无 CJK 测试夹具 | 单独一个 PR：加 `displayWidth` + 中文 context fixture |
| P2-2 | 11 个 `current` 子命令仍重复（312 行，6 个只差 key 列表） | 收益可观但改动面大，且与 P2-1 的 builder 需一起设计 | 下一轮重构，复用 `toolCurrentSpec` |
| P2-3 | registry 仍是 46 项手写列表，30 项重复 `toolNA` | 纯结构优化，无行为收益 | 按需，新增工具变多时再做 |
| P2-5 | `setSubcommandGroups` 按命令名二次耦合 | 当前列表完整、有测试覆盖单命令分组 | 改为注册处声明分组 + `AllChildCommandsHaveGroup()` 守卫 |
| P2-7 | `fix` / `docker fix` 的 `--dry-run` 默认 false 却必须传 | 涉及 CLI 行为变更与测试更新 | 默认改 true |
| P2-9 | `snapshot` 不记录过滤条件，过滤后 diff 变大量假新增 | 需新增 `Filters` 字段并同步 schema | 加 `omitempty` 字段 + diff 警告 |
| P3-1 | CI 只跑在自托管 runner，外部贡献者无 CI | 需维护者确认基础设施意图 | 改 `ubuntu-latest` 或加 fallback 表达式 |
| P3-2 | CI 每次跑 5 遍完整测试 | 需与 CI 一起调整 | 单次 `-race -coverprofile` + 单独稳定性 |
| P3-10 | Prometheus textfile 并发写丢更新 | 需平台相关的 `flock` | POSIX 加锁 + Windows no-op |
| P3-13 | `.goreleaser.yaml` 不构建 Windows | 属产品决策，README 从未承诺 Windows | 要么加 windows 目标，要么 README 明示支持平台 |

---

## 验证汇总

最终门禁（全部通过）：

```
gofmt -l .                              -> 0 files
go build ./...                          -> exit 0
go vet ./...                            -> exit 0
go test ./...  (26 packages)            -> all ok
go test ./internal/repopolicy (policy)  -> ok (文件大小/债务标记/AGENTS.md 新鲜度)
bash scripts/check-coverage.sh          -> total=89.4% minimum=80.0%
```

关键行为对比（均用真实二进制验证）：

| 行为 | 修复前 | 修复后 |
|---|---|---|
| `wrangler status --timeout 100ms` | 12,096 ms | **50 ms** |
| `status --json --installed-only` 诊断数 | 2 | **39** |
| `kubectl status --json \| diff -` | exit 1（缺 schema_version） | **exit 0** |
| `docker update --json` 退出码 | 0（与纯文本不一致） | **1（一致）** |
| `kubectl current --json` 的 `warnings` | `null` | **`[]`** |
| `all-cli status extra` | exit 0（静默成功） | **exit 1** |
| 坏 `POSTHOG_HOST` + `all-cli version` | exit 1，无输出 | **exit 0 + 警告** |
| `all-cli schema fix-plan` | exit 1 | **exit 0** |
| 子进程调用数（13 个 adapter 工具） | 31（10 重复） | **21（0 重复）** |
| `gofmt -l .`（Windows） | 130 文件 | **0 文件** |

**已知限制**：本机 `CGO_ENABLED=0` 且无 gcc，`go test -race` 无法运行
（`just check` 含 race 步骤）。并发相关修复（P1-2、P0-4）用
精确调用计数与并发 goroutine 断言直接验证，但仍建议在装有 C 编译器的
环境跑一次 `just test-race` 取得 race detector 的确认。
