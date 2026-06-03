# PRD: 环境准备下载组件增加取消按钮

## 背景
用户在环境准备（startup）页面下载 HLAE / Plugin / FFmpeg 组件时，如果下载过程耗时较长或遇到网络问题，无法中断当前下载操作。用户希望能在下载进行时主动取消，随后通过已有的"导入文件"功能手动选择本地档案继续。

## 需求

### 功能描述
1. 当组件状态为 `downloading` 时，在任务项右侧显示一个「取消」按钮。
2. 用户点击取消后：
   - 正在进行的 HTTP 下载请求立即中止
   - 组件状态变为 `failed`，显示错误信息"下载已取消"
   - 进度条消失
   - 原有失败状态下的「重试」和「导入文件」按钮自动出现
3. 取消后用户可选择：
   - **重试**：重新触发自动下载流程
   - **导入文件**：选择本地已下载的档案进行手动安装
4. 自更新（Self-Update）不加入取消按钮，仅针对 HLAE、Plugin、FFmpeg 三个组件。

### 非功能要求
- 取消操作必须安全并发（`Service` 的 `runTasksDefault` 中组件检查是并行 goroutine 的）
- 取消一个组件不影响其他组件的下载
- `downloadAndInstallWithFallback` 回退链中，取消时应不再尝试剩余候选 URL
- 取消后需清理临时文件（已下载的部分）

## 影响范围

### 后端
| 文件 | 改动 |
|---|---|
| `internal/download/file.go` | 增加 `context.Context` 参数支持取消；接收 `ctx` 并在请求和读循环中响应取消 |
| `internal/envsetup/service.go` | `Service` 结构体新增 `cancelMap map[string]context.CancelFunc` + `cancelMu sync.Mutex` |
| `internal/envsetup/service_state.go` | `downloadFile` 创建可取消的 context 并存储到 `cancelMap`；下载完成后清理 |
| `internal/envsetup/service_actions.go` | 新增 `CancelStartupDownload(componentID string) StartupState` 方法 |
| `internal/envsetup/release_fallback.go` | `downloadAndInstallWithFallback` 支持 context 取消时停止回退链 |
| `internal/app/app_startup.go` | 新增 Wails 绑定 `CancelStartupDownload(componentID string)` |

### 前端
| 文件 | 改动 |
|---|---|
| `frontend/src/features/startup/components/StartupWizard.vue` | 在 `task.status === 'downloading'` + `task.kind === 'component'` 时显示取消按钮 |
| `frontend/src/features/startup/composables/useStartupWizard.ts` | 新增 `cancelDownload(componentID)` 方法 |

### 稳定契约变更
- 新增 Wails 暴露方法：`CancelStartupDownload(componentID string) StartupState`

## 验证
- `go test ./...` 通过
- `cd frontend && npm run build` 通过
- 手动验证：启动 app，观察组件正在下载时出现取消按钮，点击后状态变为 failed，可重试或导入
