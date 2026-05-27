package envsetup

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"cs2-highlight-tool-v2/internal/endpoints"
	"cs2-highlight-tool-v2/internal/release"
)

const (
	urlKindDirect = "url"
	urlKindMirror = "mirror_url"
	urlKindGitHub = "github_url"
)

type releaseAssetCandidate struct {
	Source   DownloadSource
	Info     *release.Info
	Asset    release.Asset
	AssetURL string
	URLKind  string
}

func infoManualURL(componentID string, fallbackSource DownloadSource, info *release.Info) string {
	if info == nil {
		return endpoints.ManualURLFor(componentID, string(fallbackSource))
	}
	return firstNonEmpty(
		info.HTMLURL,
		endpoints.ReleasePageURL(info.Repo, info.Source),
		endpoints.ManualURLFor(componentID, string(fallbackSource)),
	)
}

func (s *Service) collectReleaseAssetCandidates(componentID string, primarySource DownloadSource, selectAsset func(*release.Info) (release.Asset, bool)) ([]releaseAssetCandidate, error) {
	primarySource = normalizeDownloadSource(string(primarySource))
	info, err := s.componentReleaseInfo(primarySource, componentID)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", strings.ToUpper(string(primarySource)), err)
	}

	asset, ok := selectAsset(info)
	if !ok {
		return nil, fmt.Errorf("%s: Release 中未找到可用资产", strings.ToUpper(string(primarySource)))
	}

	countryCode := s.currentCountryCode()
	orderedURLs := orderedAssetURLsByCountry(asset, countryCode)
	candidates := make([]releaseAssetCandidate, 0, len(orderedURLs))
	seenURL := make(map[string]struct{}, len(orderedURLs))
	for _, item := range orderedURLs {
		assetURL := strings.TrimSpace(item.url)
		if assetURL == "" {
			continue
		}
		if _, exists := seenURL[assetURL]; exists {
			continue
		}
		seenURL[assetURL] = struct{}{}
		candidates = append(candidates, releaseAssetCandidate{
			Source:   primarySource,
			Info:     info,
			Asset:    asset,
			AssetURL: assetURL,
			URLKind:  item.kind,
		})
	}

	if len(candidates) == 0 {
		if preferDirectAndMirror(countryCode) {
			return nil, fmt.Errorf("%s: 资产下载链接为空（期望字段: url 或 mirror_url）", strings.ToUpper(string(primarySource)))
		}
		return nil, fmt.Errorf("%s: 资产下载链接为空（期望字段: github_url）", strings.ToUpper(string(primarySource)))
	}
	return candidates, nil
}

type urlCandidate struct {
	kind string
	url  string
}

func orderedAssetURLsByCountry(asset release.Asset, countryCode string) []urlCandidate {
	if preferDirectAndMirror(countryCode) {
		return []urlCandidate{
			{kind: urlKindMirror, url: strings.TrimSpace(asset.MirrorURL)},
			{kind: urlKindDirect, url: firstNonEmpty(strings.TrimSpace(asset.URL), strings.TrimSpace(asset.DownloadURL))},
		}
	}
	return []urlCandidate{{kind: urlKindGitHub, url: strings.TrimSpace(asset.GitHubURL)}}
}

func preferDirectAndMirror(countryCode string) bool {
	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	return countryCode == "" || countryCode == "CN"
}

func (s *Service) downloadAndInstallWithFallback(componentID string, latest string, candidates []releaseAssetCandidate, install func(path string) error) error {
	if len(candidates) == 0 {
		return fmt.Errorf("没有可用下载候选")
	}

	failures := make([]string, 0, len(candidates))
	attempt := 0
	for _, candidate := range candidates {
		attempt++
		s.emitLogWithFields("info", "开始尝试下载组件资产", logFields{
			Component: componentID,
			Stage:     "download_fallback",
			Action:    "attempt",
			Source:    string(candidate.Source),
			Attempt:   attempt,
			Meta: map[string]string{
				"url":      candidate.AssetURL,
				"url_kind": candidate.URLKind,
			},
		})
		targetPath := tempAssetPath(s.dataDir, componentID, latest, candidate.Asset)
		if err := s.downloadFile(componentID, candidate.AssetURL, targetPath); err != nil {
			failures = append(failures, fmt.Sprintf("%d/%s(%s) 下载失败: %v", attempt, strings.ToUpper(string(candidate.Source)), candidate.URLKind, err))
			continue
		}
		if err := install(targetPath); err != nil {
			failures = append(failures, fmt.Sprintf("%d/%s(%s) 安装失败: %v", attempt, strings.ToUpper(string(candidate.Source)), candidate.URLKind, err))
			continue
		}
		return nil
	}

	if len(failures) == 0 {
		return fmt.Errorf("所有下载回退均失败")
	}
	return fmt.Errorf("%s", strings.Join(failures, " | "))
}

func tempAssetPath(dataDir, componentID, latest string, asset release.Asset) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(asset.Name)))
	if ext == "" {
		u, err := url.Parse(strings.TrimSpace(release.AssetDownloadURL(asset)))
		if err == nil {
			ext = strings.ToLower(filepath.Ext(u.Path))
		}
	}
	if ext == "" {
		ext = ".zip"
	}
	return filepath.Join(dataDir, "temp", componentID+"_"+sanitizeFileName(latest)+ext)
}
