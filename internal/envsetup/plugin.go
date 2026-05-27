package envsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cs2-highlight-tool-v2/internal/config"
	"cs2-highlight-tool-v2/internal/download"
	"cs2-highlight-tool-v2/internal/endpoints"
	"cs2-highlight-tool-v2/internal/release"
)

const pluginDLLFileName = "server.dll"

func (s *Service) ensurePlugin(source DownloadSource) error {
	source = normalizeDownloadSource(string(source))
	cfg := s.currentConfig()

	checkStarted := s.logStepStart(componentPlugin, "check", "validate_local", string(source), 0, map[string]string{
		"local_dll": cfg.PluginDLL,
	})
	localVersion, localErr := resolveInstalledPluginVersion(cfg.PluginDLL)
	localReady := localErr == nil
	s.logStepDone(componentPlugin, "check", "validate_local", string(source), 0, checkStarted, map[string]string{
		"local_ready": strconv.FormatBool(localReady),
		"local_ver":   localVersion,
	})

	releaseURL := endpoints.APIURLFor(componentPlugin, string(source))
	fetchStarted := s.logStepStart(componentPlugin, "fetch_release", "from_unified_snapshot", string(source), 0, map[string]string{
		"api_url": releaseURL,
	})
	candidates, err := s.collectReleaseAssetCandidates(componentPlugin, source, release.SelectPluginAsset)
	if err != nil {
		s.logStepFail(componentPlugin, "fetch_release", "from_unified_snapshot", string(source), 0, fetchStarted, err, map[string]string{
			"api_url": releaseURL,
		})
		if localReady {
			return fmt.Errorf("获取插件 DLL Release 失败: %v；本地版本可用，但更新获取失败，可能导致后续功能不可用", err)
		}
		return fmt.Errorf("获取插件 DLL Release 失败: %w", err)
	}
	info := candidates[0].Info
	asset := candidates[0].Asset
	assetURL := candidates[0].AssetURL
	s.logStepDone(componentPlugin, "fetch_release", "from_unified_snapshot", string(source), 0, fetchStarted, map[string]string{
		"tag":             info.TagName,
		"selected_source": string(candidates[0].Source),
	})

	selectStarted := s.logStepStart(componentPlugin, "select_asset", "pick", string(source), 0, nil)
	if strings.ToLower(filepath.Ext(asset.Name)) != ".zip" {
		selectErr := fmt.Errorf("插件 Release 资产必须为 .zip: %s", asset.Name)
		s.logStepFail(componentPlugin, "select_asset", "pick", string(source), 0, selectStarted, selectErr, map[string]string{
			"asset_name": asset.Name,
		})
		return selectErr
	}
	s.logStepDone(componentPlugin, "select_asset", "pick", string(source), 0, selectStarted, map[string]string{
		"asset_name": asset.Name,
		"asset_url":  assetURL,
	})

	latest := firstNonEmpty(info.TagName, asset.Name)
	s.updateStep(componentPlugin, func(step *ComponentStatus) {
		step.RemoteVersion = latest
		step.ManualURL = infoManualURL(componentPlugin, source, info)
	})
	if localReady && release.CompareVersions(localVersion, latest) >= 0 {
		s.updateStep(componentPlugin, func(step *ComponentStatus) {
			step.Status = statusReady
			step.LocalVersion = localVersion
			step.Path = cfg.PluginDLL
		})
		s.emitLogWithFields("info", "本地插件已是最新版本", logFields{
			Component: componentPlugin,
			Stage:     "ready",
			Action:    "skip_install",
			Source:    string(source),
			Meta: map[string]string{
				"local_version":  localVersion,
				"remote_version": latest,
			},
		})
		return nil
	}

	s.updateStep(componentPlugin, func(step *ComponentStatus) { step.Status = statusDownloading })
	if err := s.downloadAndInstallWithFallback(componentPlugin, latest, candidates, s.installPluginFromFile); err != nil {
		return fmt.Errorf("下载插件 DLL 失败: %w", err)
	}
	return nil
}

func (s *Service) installPluginFromFile(path string) error {
	s.updateStep(componentPlugin, func(step *ComponentStatus) { step.Status = statusInstalling })
	pluginDir := filepath.Join(s.dataDir, "plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return err
	}
	if strings.ToLower(filepath.Ext(path)) != ".zip" {
		return fmt.Errorf("请选择包含 %s 的 zip 文件", pluginDLLFileName)
	}

	extractDir := filepath.Join(s.dataDir, "temp", "plugin_extract")
	_ = os.RemoveAll(extractDir)
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return err
	}
	defer os.RemoveAll(extractDir)

	extractStarted := s.logStepStart(componentPlugin, "extract", "unzip", string(s.currentSource()), 0, map[string]string{
		"archive_path": path,
		"extract_dir":  extractDir,
	})
	if err := download.Unzip(path, extractDir); err != nil {
		s.logStepFail(componentPlugin, "extract", "unzip", string(s.currentSource()), 0, extractStarted, err, nil)
		return err
	}
	s.logStepDone(componentPlugin, "extract", "unzip", string(s.currentSource()), 0, extractStarted, nil)

	validateStarted := s.logStepStart(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, nil)
	sourceDLL := filepath.Join(extractDir, pluginDLLFileName)
	sourceDLLInfo, err := os.Stat(sourceDLL)
	if err != nil || sourceDLLInfo.IsDir() {
		if err == nil {
			err = fmt.Errorf("插件压缩包中未找到 %s", pluginDLLFileName)
		} else if os.IsNotExist(err) {
			err = fmt.Errorf("插件压缩包中未找到 %s", pluginDLLFileName)
		}
		s.logStepFail(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, err, nil)
		return err
	}
	sourceChangelog := filepath.Join(extractDir, "changelog.xml")
	sourceChangelogInfo, err := os.Stat(sourceChangelog)
	if err != nil || sourceChangelogInfo.IsDir() {
		if err == nil {
			err = fmt.Errorf("插件压缩包中未找到 changelog.xml")
		} else if os.IsNotExist(err) {
			err = fmt.Errorf("插件压缩包中未找到 changelog.xml")
		}
		s.logStepFail(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, err, nil)
		return err
	}

	target := filepath.Join(pluginDir, pluginDLLFileName)
	if err := download.CopyFile(sourceDLL, target); err != nil {
		s.logStepFail(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, err, nil)
		return err
	}
	targetChangelog := filepath.Join(pluginDir, "changelog.xml")
	if err := download.CopyFile(sourceChangelog, targetChangelog); err != nil {
		s.logStepFail(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, err, nil)
		return err
	}
	if err := validatePlugin(target); err != nil {
		s.logStepFail(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, err, nil)
		return err
	}
	installedVersion, err := resolveInstalledPluginVersion(target)
	if err != nil {
		s.logStepFail(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, err, nil)
		return err
	}
	if err := writeComponentVersion(pluginDir, installedVersion, target); err != nil {
		s.logStepFail(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, err, nil)
		return err
	}
	s.logStepDone(componentPlugin, "validate", "verify_archive", string(s.currentSource()), 0, validateStarted, map[string]string{
		"target_dll":       target,
		"target_changelog": targetChangelog,
		"version":          installedVersion,
	})

	persistStarted := s.logStepStart(componentPlugin, "persist_config", "write", string(s.currentSource()), 0, nil)
	cfg, err := s.persistConfig(func(next *config.Config) error {
		next.PluginDLL = target
		return nil
	})
	if err != nil {
		s.logStepFail(componentPlugin, "persist_config", "write", string(s.currentSource()), 0, persistStarted, err, nil)
		return err
	}
	s.logStepDone(componentPlugin, "persist_config", "write", string(s.currentSource()), 0, persistStarted, map[string]string{
		"config_plugin_dll": cfg.PluginDLL,
	})
	s.updateStep(componentPlugin, func(step *ComponentStatus) {
		step.Status = statusReady
		step.LocalVersion = installedVersion
		step.Path = cfg.PluginDLL
		step.Error = ""
	})
	s.emitLogWithFields("info", "插件安装完成", logFields{
		Component: componentPlugin,
		Stage:     "ready",
		Action:    "component_ready",
		Source:    string(s.currentSource()),
		Meta: map[string]string{
			"version": installedVersion,
			"path":    cfg.PluginDLL,
		},
	})
	return nil
}

func resolveInstalledPluginVersion(pluginDLL string) (string, error) {
	if err := validatePlugin(pluginDLL); err != nil {
		return "", err
	}
	changelogPath := filepath.Join(filepath.Dir(pluginDLL), "changelog.xml")
	version, err := readHLAEVersionFromChangelog(changelogPath)
	if err != nil {
		return "", err
	}
	return version, nil
}

func validatePlugin(path string) error {
	if path == "" {
		return fmt.Errorf("插件 DLL 路径为空")
	}
	if !strings.EqualFold(filepath.Base(path), pluginDLLFileName) {
		return fmt.Errorf("插件文件名必须为 %s", pluginDLLFileName)
	}
	if strings.ToLower(filepath.Ext(path)) != ".dll" {
		return fmt.Errorf("插件文件不是 DLL: %s", path)
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("插件 DLL 不存在: %s", path)
	}
	return nil
}
