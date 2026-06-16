# 录制配置选项扩展（天空变黑 / 击杀留存 / 屏蔽击杀）

## Goal

把用户已在 `internal/clipsjson/builder.go` 的 `buildBootstrapSequence` 里硬编码的 4 条新命令，按"天空变黑 / 击杀留存时间 / 屏蔽所有击杀信息"三类暴露到设置页面（与现有 `hide_all_ui` / `pov_hud_enabled` 同区块），同时调整默认值：POV HUD 默认开启、天空变黑默认开启、屏蔽所有击杀信息默认关闭。

## What I already know

- git diff 在 `builder.go:501-508` 新增 5 行命令（`cl_spec_show_bindings 0` + 2 条天空 + 1 条 lifetime + 1 条 deathmsg filter）。
- 用户未提及 `cl_spec_show_bindings 0` 需要开关 → 视为固定命令，常驻 bootstrap。
- ClipSettings 三处定义需同步：
  - `internal/app/clip_settings.go` (Wails 边界 + `GetClipSettings/SaveClipSettings` 公开方法)
  - `internal/config/config.go` (持久化 + 默认值)
  - `frontend/src/shared/types/clips.ts` (前端类型)
- bootstrap 选项类 `bootstrapOptions`（builder.go:119）和上游 `BuildOptions`（builder.go:62）需新增对应字段。
- 选项流转链路：`SettingsPanel.vue` ↔ `GetClipSettings/SaveClipSettings` → `config.Config` → `plugin_generate.go:488` (`BuildOptions`) → `buildBootstrapSequence`。
- 现有 `HideAllUI` (`cl_draw_only_deathnotices 1`，"只显示死亡通知") 与新增 `block_kill_feed`(`mirv_deathmsg filter add block 1`，"屏蔽所有死亡通知") 语义正交，可独立存在。
- POV HUD 默认变更会影响已存在用户：`config.LoadOrCreate` 会反序列化已有 `config.json`，但因为零值就是 false，无法区分"用户主动关"与"老配置缺字段"。
- 受影响测试：`internal/app/produce_pov_test.go:282-283` 显式断言 `default PovHudEnabled should be false`，需要同步翻转。

## Assumptions (temporary)

- `cl_spec_show_bindings 0` 固定加入，不做开关。
- 击杀留存 lifetime 是整数秒（mirv_deathmsg lifetime 接收整数；diff 默认 4）。
- 新增三项 UI 放在设置页面 `clip_title` 区块底部，与 `hide_all_ui` / `pov_hud_enabled` 并列。
- POV HUD 默认值变更只影响 `Default()` 返回值；老 config.json 因已包含 `pov_hud_enabled` 字段而不会被刷新（与现有 `EnableSpecShowXray` 的迁移钩子类似，可加一段"老 config 缺字段则视为默认值"逻辑）。

## Decisions

- **lifetime 范围**：默认 4，范围 1–10，整数（step=1）。
- **POV HUD / SkyBlackout / BlockKillFeed 默认值变更迁移**：只对新用户生效。老 config.json 已有显式字段值的，按读出值保留，不做反向迁移。对应实现上：仅修改 `Default()`，不在 `ApplyDefaults` 中触发"缺字段则上默认"的回填（除非字段全新、JSON 里完全缺）；为避免歧义，新字段在 `ApplyDefaults` 走"零值即视为缺省"的回填路径，因为它们是全新增字段且老 config 一定缺它们 —— 老用户首次升级会获得新字段的新默认（即 SkyBlackout=true / BlockKillFeed=false / KillFeedLifetime=4），但 PovHudEnabled 因为已在老 config 中存在而不被回填。

## Requirements (evolving)

- 后端
  - `config.Config` 新增三个字段（带 json tag + 默认值）。
  - `Default()` 中：`PovHudEnabled=true`、`SkyBlackout=true`、`BlockKillFeed=false`、`KillFeedLifetime=<待定默认>`。
  - `ApplyDefaults` 对 lifetime 范围做钳制。
  - `ClipSettings` (app 层) 三个字段对应透传。
  - `BuildOptions` / `bootstrapOptions` 三个字段对应透传。
  - `buildBootstrapSequence` 按开关条件输出对应 cmd；lifetime 用参数化数值。
- 前端
  - `ClipSettings` TS 类型 + `SettingsPanel.vue` reactive 初始值同步新增三项。
  - clip 区块底部新增 3 个 setting-row：sky_blackout 开关、kill_feed_lifetime 数字输入、block_kill_feed 开关。
  - i18n `zh-CN.json` 三个新 key。
- 测试
  - 调整 `produce_pov_test.go:282` 默认值断言。
  - 给 `builder.go` 增加单测：四种开关排列下 bootstrap 输出包含/不包含对应行。

## Acceptance Criteria (evolving)

- [ ] 设置页打开后能看到三个新控件，默认值符合规范（天空开、屏蔽关、lifetime=默认值）。
- [ ] 切换天空开关后保存再读出能保持；生成的 plugin JSON 中相应包含/不包含 `mirv_sky clouds draw 0` + `r_drawskybox 0` 两条。
- [ ] 切换屏蔽击杀开关后生成的 plugin JSON 中相应包含/不包含 `mirv_deathmsg filter add block 1`。
- [ ] 修改 lifetime 数字后 bootstrap 中 `mirv_deathmsg lifetime <N>` 数值跟随变化。
- [ ] 全新启动（无 config.json）时 PovHudEnabled / SkyBlackout 默认 true，BlockKillFeed 默认 false。
- [ ] `go test ./...` 与 `cd frontend && npm run build` 均通过。

## Definition of Done

- 三处 ClipSettings schema 一致（Go config / Go app / TS）。
- builder.go 单测覆盖新增条件输出。
- produce_pov_test.go 默认值断言更新。
- zh-CN.json 三个新 key 加上（en-US 用户自维护，不动）。
- 不破坏现有 stable Wails 方法签名（仅扩展字段，前端零值兼容）。

## Out of Scope (explicit)

- `cl_spec_show_bindings 0` 不做开关。
- 不调整 en-US.json。
- 不重排现有设置页面其它分组结构。
- 不引入新的 Wails 公开方法。

## Technical Notes

- 受影响文件：
  - `internal/clipsjson/builder.go` (bootstrapOptions / BuildOptions / buildBootstrapSequence)
  - `internal/clipsjson/builder_test.go` (新增 case)
  - `internal/config/config.go` (Config 字段 + Default + ApplyDefaults)
  - `internal/app/clip_settings.go` (ClipSettings + Get/Save 透传 + normalize)
  - `internal/app/plugin_generate.go` (ClipSettings → BuildOptions 桥接)
  - `internal/app/produce_pov_test.go` (默认值断言)
  - `frontend/src/shared/types/clips.ts`
  - `frontend/src/features/settings/components/SettingsPanel.vue`
  - `frontend/src/features/clips/pages/ClipsPage.vue`（默认值兜底）
  - `frontend/src/shared/i18n/zh-CN.json`
- lifetime 钳制：`if v <= 0 { v = 4 }; if v < 1 { v = 1 }; if v > 10 { v = 10 }`。
- 新字段在老 config 中缺失时，按零值进入 `ApplyDefaults`：BlockKillFeed=false（零值=默认），SkyBlackout 零值是 false 但默认要 true，需在 `ApplyDefaults` 通过"检测 JSON 中是否含 sky_blackout 字段"决定是否回填（参考 `EnableSpecShowXray` 在 config.go:110 处的 `strings.Contains(string(data), "enable_spec_show_xray_zero")` 模式）。同理 KillFeedLifetime=0 时回填 4。
