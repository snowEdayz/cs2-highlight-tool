package envsetup

import (
	"fmt"
	"net/url"
	"strings"

	"cs2-highlight-tool-v2/internal/endpoints"
	"cs2-highlight-tool-v2/internal/release"
	"cs2-highlight-tool-v2/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (s *Service) checkSelfUpdate(source DownloadSource) {
	current := s.version
	source = normalizeDownloadSource(string(source))
	s.emitLogWithFields("info", "开始检查软件更新", logFields{
		Component: "self_update",
		Stage:     "check",
		Action:    "start",
		Source:    string(source),
		Meta: map[string]string{
			"current": current,
		},
	})

	apiURL := endpoints.APIURLFor("app", string(source))
	fetchStarted := s.logStepStart("self_update", "fetch_release", "from_unified_snapshot", string(source), 1, map[string]string{
		"api_url": apiURL,
	})
	candidates, err := s.collectReleaseAssetCandidates("self_update", source, release.SelectAppExeAsset)
	if err != nil {
		s.logStepFail("self_update", "fetch_release", "from_unified_snapshot", string(source), 1, fetchStarted, err, map[string]string{
			"api_url": apiURL,
		})
		s.mu.Lock()
		s.state.SelfUpdate = SelfUpdateState{
			Status:  statusFailed,
			Current: current,
			Error:   fmt.Sprintf("软件更新检查失败: %v", err),
			URL:     endpoints.ManualURLFor("self_update", string(source)),
		}
		s.mu.Unlock()
		s.emitLogWithFields("warning", fmt.Sprintf("软件更新检查失败: %v，将继续环境检查", err), logFields{
			Component: "self_update",
			Stage:     "check",
			Action:    "failed",
			Source:    string(source),
			Error:     err.Error(),
		})
		s.emitState()
		return
	}
	info := candidates[0].Info
	asset := candidates[0].Asset
	assetURL := candidates[0].AssetURL
	s.logStepDone("self_update", "fetch_release", "from_unified_snapshot", string(source), 1, fetchStarted, map[string]string{
		"tag":             info.TagName,
		"selected_source": string(candidates[0].Source),
	})

	selectStarted := s.logStepStart("self_update", "select_asset", "pick", string(source), 0, nil)
	latest := info.TagName
	if latest == "" {
		latest = current
	}
	s.logStepDone("self_update", "select_asset", "pick", string(source), 0, selectStarted, map[string]string{
		"asset_name": asset.Name,
		"asset_url":  assetURL,
		"latest":     latest,
	})

	if release.CompareVersions(current, latest) >= 0 {
		s.mu.Lock()
		s.state.SelfUpdate = SelfUpdateState{
			Status:  statusReady,
			Current: current,
			Latest:  latest,
			URL:     infoManualURL("self_update", source, info),
		}
		s.mu.Unlock()
		s.emitLogWithFields("info", "当前已是最新软件版本", logFields{
			Component: "self_update",
			Stage:     "check",
			Action:    "up_to_date",
			Source:    string(source),
			Meta: map[string]string{
				"current": current,
				"latest":  latest,
			},
		})
		s.emitState()
		return
	}
	s.mu.Lock()
	s.state.SelfUpdate = SelfUpdateState{
		Status:    statusNeedsAction,
		Available: true,
		Current:   current,
		Latest:    latest,
		URL:       infoManualURL("self_update", source, info),
		AssetURL:  assetURL,
	}
	s.state.CanEnterMain = false
	s.mu.Unlock()
	s.emitLogWithFields("warning", fmt.Sprintf("发现软件新版本（%s）: %s -> %s", strings.ToUpper(string(source)), current, latest), logFields{
		Component: "self_update",
		Stage:     "check",
		Action:    "update_available",
		Source:    string(source),
		Meta: map[string]string{
			"current": current,
			"latest":  latest,
		},
	})
	s.emitState()
}

func (s *Service) downloadAndApplySelfUpdate(update SelfUpdateState) error {
	started := s.logStepStart("self_update", "download_and_apply", "start", "", 0, map[string]string{
		"latest":    update.Latest,
		"asset_url": update.AssetURL,
	})
	s.mu.Lock()
	s.state.SelfUpdate.Status = statusDownloading
	s.state.SelfUpdate.Error = ""
	s.mu.Unlock()
	s.emitState()

	urls := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	addURL := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return
		}
		if _, ok := seen[raw]; ok {
			return
		}
		seen[raw] = struct{}{}
		urls = append(urls, raw)
	}

	addURL(update.AssetURL)
	candidates, err := s.collectReleaseAssetCandidates("self_update", s.currentSource(), release.SelectAppExeAsset)
	if err == nil {
		for _, candidate := range candidates {
			addURL(candidate.AssetURL)
		}
	}
	githubURL := firstGitHubURL(candidates)
	if githubURL == "" && isGitHubURL(update.AssetURL) {
		githubURL = strings.TrimSpace(update.AssetURL)
	}
	if githubURL != "" {
		addURL(endpoints.BuildGHProxyURL(githubURL))
	}
	if len(urls) == 0 {
		err := fmt.Errorf("没有可用的软件更新下载链接")
		s.logStepFail("self_update", "download_and_apply", "start", "", 0, started, err, nil)
		return err
	}

	var lastErr error
	for idx, rawURL := range urls {
		s.emitLogWithFields("info", "开始尝试下载并应用软件更新", logFields{
			Component: "self_update",
			Stage:     "download_and_apply",
			Action:    "attempt",
			Attempt:   idx + 1,
			Meta: map[string]string{
				"url": rawURL,
			},
		})
		attemptErr := updater.StartApply(s.dataDir, rawURL, update.Latest, func(url, targetPath string) error {
			return s.downloadFile("self_update", url, targetPath)
		})
		if attemptErr == nil {
			lastErr = nil
			break
		}
		lastErr = attemptErr
	}
	if lastErr != nil {
		s.logStepFail("self_update", "download_and_apply", "start", "", 0, started, lastErr, nil)
		return lastErr
	}

	s.mu.Lock()
	s.state.SelfUpdate.Status = statusInstalling
	s.mu.Unlock()
	s.logStepDone("self_update", "download_and_apply", "start", "", 0, started, nil)
	s.emitLogWithFields("info", "软件更新已下载，正在重启并替换主程序", logFields{
		Component: "self_update",
		Stage:     "apply",
		Action:    "restart_replace",
		Meta: map[string]string{
			"latest": update.Latest,
		},
	})
	s.emitState()
	runtime.Quit(s.ctx)
	return nil
}

func firstGitHubURL(candidates []releaseAssetCandidate) string {
	for _, candidate := range candidates {
		raw := strings.TrimSpace(candidate.AssetURL)
		if isGitHubURL(raw) {
			return raw
		}
	}
	return ""
}

func isGitHubURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(u.Hostname()), "github.com")
}
