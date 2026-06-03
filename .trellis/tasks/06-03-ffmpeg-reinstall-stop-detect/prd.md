# PRD: 修复 FFmpeg 重装前后台探测占用

## Goal

FFmpeg 组件点击“重新安装”前，应先停止或等待当前后台 FFmpeg 编码能力探测结束，再删除 `<dataDir>/ffmpeg` 目录并重新安装，避免 Windows 因 `ffmpeg.exe` 仍在运行而报“删除 ffmpeg 目录失败”。

## What I Already Know

- 用户复现路径：为了测试取消下载，对 FFmpeg 点击“重新安装”，提示删除 ffmpeg 目录失败。
- HLAE 和插件可正常重装，因为它们没有启动 FFmpeg 能力探测进程。
- FFmpeg 安装/检测成功后会调用 `scheduleFFmpegCapabilityDetection` 异步执行 `<dataDir>/ffmpeg/bin/ffmpeg.exe`。
- `reinstallFFmpeg` 当前直接 `os.RemoveAll(<dataDir>/ffmpeg)`，没有等待或取消后台探测。
- 管理员权限无法解决正在运行的 exe 文件占用问题。

## Requirements

- FFmpeg 重装前必须停止或等待当前后台能力探测退出。
- 重装等待不能永久卡死；探测进程已有 per-probe timeout，重装流程应可确定结束。
- 新的探测调度仍保持“同一时间只跑一个探测任务”的语义。
- HLAE / 插件重装行为不变。

## Acceptance Criteria

- [ ] 当 FFmpeg 后台探测正在运行时，调用 `reinstallFFmpeg` 会先结束探测，再尝试删除 ffmpeg 目录。
- [ ] 不再因本软件自己的 FFmpeg 探测进程占用导致立即删除失败。
- [ ] 有回归测试覆盖“重装前会等待/停止探测”。
- [ ] `go test ./...` 通过。

## Definition of Done

- Tests added/updated.
- Backend checks pass.
- 如新增约定，更新 `.trellis/spec`。
- 提交独立 work commit。

## Out of Scope

- 不处理用户外部录制/合成任务或第三方杀毒、资源管理器预览占用 ffmpeg 目录。
- 不改前端 UI 文案。
- 不改变 FFmpeg 能力探测策略本身。

## Technical Notes

- 相关文件：
  - `internal/envsetup/ffmpeg.go`
  - `internal/envsetup/service_actions.go`
  - `internal/envsetup/ffmpeg_detect_test.go`
- 可能实现方向：为 FFmpeg 能力探测保存 cancel func，在重装前 cancel 并 `Wait()`，然后再 `os.RemoveAll`。
