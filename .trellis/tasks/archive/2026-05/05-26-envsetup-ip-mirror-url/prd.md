# envsetup: 国内 IP 优先使用 mirror_url 下载源

## Goal

当 GeoIP 检测到中国 IP（country_code = CN）时，组件下载候选 URL 的优先顺序应改为：
**`mirror_url` 优先，`url`/`download_url` 作为回退**。  
目前代码逻辑是反过来的：先用 `url`，失败后才用 `mirror_url`。

## What I already know

- 核心逻辑位于 `internal/envsetup/release_fallback.go`：
  - `orderedAssetURLsByCountry(asset, countryCode)` 决定候选 URL 顺序
  - `preferDirectAndMirror(countryCode)` 判断是否走"url/mirror 链路"（countryCode == "" 或 "CN"）
  - 当前 CN 顺序：`{urlKindDirect, asset.URL}` → `{urlKindMirror, asset.MirrorURL}`
- `release.Asset` 结构定义在 `internal/release/resolver.go`，字段：`URL`, `DownloadURL`, `GitHubURL`, `MirrorURL`, `BrowserDownloadURL`
- 非 CN（如 US、JP）走 `github_url`，不受影响
- 空国家码（countryCode == ""）当前与 CN 走同一分支（preferDirectAndMirror = true）
- GeoIP 失败时 countryCode = ""，也走同一分支
- 下载循环在 `downloadAndInstallWithFallback` 中，按候选顺序依次尝试，首个成功即返回

## Decision (ADR-lite)

**Context**: `preferDirectAndMirror` 对 CN 和空国家码返回相同结果，两者共用下载策略。  
**Decision**: 选项 B — CN 和空国家码都改为 mirror_url 优先。  
**Consequences**: GeoIP 失败的国内用户也受益；极少数海外用户 GeoIP 失败时会先尝试 mirror，失败后自动回退 url，影响可接受。

## Requirements

- [x] CN country_code 时，下载候选顺序改为 mirror_url 先于 url/download_url
- [x] 空 country_code（GeoIP 失败）同步改为 mirror_url 优先

## Acceptance Criteria

- [ ] `orderedAssetURLsByCountry` 对 CN 和空 countryCode 返回 mirror_url 在前的候选列表
- [ ] 当 mirror_url 为空时，能正确降级到 url/download_url（原有去重/过滤逻辑不变）
- [ ] 非 CN 国家行为不变（走 github_url）
- [ ] 相关单元测试覆盖新顺序（`CNUsesMirrorThenURL`、`UnknownCountryUsesMirrorThenURL`）
- [ ] `go test ./internal/envsetup ./internal/release` pass

## Definition of Done

- 单元测试 pass（`go test ./internal/envsetup ./internal/release`）
- 类型检查 + 前端构建 pass

## Out of Scope

- FFmpeg 固定 URL 逻辑
- GeoIP 检测逻辑本身
- 非 CN 国家下载策略

## Technical Notes

- 改动点：`internal/envsetup/release_fallback.go` 第 86–94 行 `orderedAssetURLsByCountry` 函数
- 相关测试：`internal/envsetup/release_fallback_test.go`
- `source_test.go` 中有 countryCode 相关测试，需同步验证
