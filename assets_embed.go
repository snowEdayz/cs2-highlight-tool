package main

import "embed"

//go:embed frontend/dist
var assets embed.FS

//go:embed wails.json
var wailsConfigData []byte

const (
	ffmpegDownloadURLGlobal = "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.7z"
	ffmpegDownloadURLCN     = "https://gitee.com/hkslover/ffmpeg_release/releases/download/v8.0.1/ffmpeg-8.0.1-essentials_build.7z"

	statsBaseURL = "https://snowblog.xyz/api/v1"

	updateAPIURLGitHub = "https://api.github.com/repos/hkslover/cs2-highlight-tool/releases/latest"
	updateAPIURLGitee  = "https://gitee.com/api/v5/repos/hkslover/cs2-highlight-tool/releases/latest"

	perfectWorldDemoURLFormat = "https://pwaweblogin.wmpvp.com/csgo/demo/%s"
	fiveEMatchAPIURLFormat    = "https://gate.5eplay.com/crane/http/api/data/match/%s"

	hlaeReleaseAPIURLGitee  = "https://gitee.com/api/v5/repos/hkslover/advancedfx/releases/latest"
	hlaeReleaseAPIURLGitHub = "https://api.github.com/repos/advancedfx/advancedfx/releases/latest"
)
