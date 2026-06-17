package envsetup

import (
	"fmt"
	"strings"

	"cs2-highlight-tool-v2/internal/endpoints"
	"cs2-highlight-tool-v2/internal/release"
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
		URL:       assetURL,
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
