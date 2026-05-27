package envsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cs2-highlight-tool-v2/internal/config"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var detectCS2ExeFromSteamFn = detectCS2ExeFromSteam

func (s *Service) ensureCS2Path() error {
	cfg := s.currentConfig()
	checkStarted := s.logStepStart(componentCS2, "check", "resolve_path", "", 0, map[string]string{
		"cs2_dir": cfg.CS2Dir,
		"cs2_exe": cfg.CS2Exe,
	})
	cs2Exe, err := resolveCS2Exe(cfg.CS2Dir, cfg.CS2Exe)
	if err != nil {
		detectStarted := s.logStepStart(componentCS2, "auto_detect", "detect_steam", "", 0, nil)
		detected, detectErr := detectCS2ExeFromSteamFn()
		if detectErr != nil {
			s.logStepFail(componentCS2, "auto_detect", "detect_steam", "", 0, detectStarted, detectErr, nil)
			s.logStepDone(componentCS2, "check", "resolve_path", "", 0, checkStarted, map[string]string{
				"resolved": "false",
			})
			s.updateStep(componentCS2, func(step *ComponentStatus) {
				step.Status = statusNeedsAction
				step.Error = err.Error()
				step.Path = cfg.CS2Exe
			})
			s.emitLogWithFields("warning", "CS2 路径需要用户处理", logFields{
				Component: componentCS2,
				Stage:     "check",
				Action:    "needs_action",
				Error:     err.Error(),
			})
			return nil
		}
		s.logStepDone(componentCS2, "auto_detect", "detect_steam", "", 0, detectStarted, map[string]string{
			"cs2_exe": detected,
		})
		cs2Exe = detected
	}
	s.logStepDone(componentCS2, "check", "resolve_path", "", 0, checkStarted, map[string]string{
		"resolved": "true",
		"cs2_exe":  cs2Exe,
	})

	persistStarted := s.logStepStart(componentCS2, "persist_config", "write", "", 0, nil)
	updatedCfg, err := s.persistConfig(func(next *config.Config) error {
		next.CS2Exe = cs2Exe
		next.CS2Dir = filepath.Dir(cs2Exe)
		return nil
	})
	if err != nil {
		s.logStepFail(componentCS2, "persist_config", "write", "", 0, persistStarted, err, nil)
		return err
	}
	s.logStepDone(componentCS2, "persist_config", "write", "", 0, persistStarted, map[string]string{
		"config_cs2_exe": updatedCfg.CS2Exe,
	})
	s.updateStep(componentCS2, func(step *ComponentStatus) {
		step.Status = statusReady
		step.Path = updatedCfg.CS2Exe
		step.Error = ""
	})
	s.emitLogWithFields("info", "CS2 路径校验完成", logFields{
		Component: componentCS2,
		Stage:     "ready",
		Action:    "component_ready",
		Meta: map[string]string{
			"path": updatedCfg.CS2Exe,
		},
	})
	return nil
}

func (s *Service) saveCS2Path(selected string) error {
	selected = config.CleanPath(selected)
	saveStarted := s.logStepStart(componentCS2, "save_path", "manual_select", "", 0, map[string]string{
		"selected": selected,
	})
	cs2Exe, err := resolveCS2Exe(selected, selected)
	if err != nil {
		s.logStepFail(componentCS2, "save_path", "manual_select", "", 0, saveStarted, err, nil)
		return err
	}
	_, err = s.persistConfig(func(next *config.Config) error {
		next.CS2Exe = cs2Exe
		next.CS2Dir = filepath.Dir(cs2Exe)
		return nil
	})
	if err != nil {
		s.logStepFail(componentCS2, "save_path", "manual_select", "", 0, saveStarted, err, nil)
		return err
	}
	s.logStepDone(componentCS2, "save_path", "manual_select", "", 0, saveStarted, map[string]string{
		"cs2_exe": cs2Exe,
	})
	return err
}

func resolveCS2Exe(dirOrEmpty, exeOrEmpty string) (string, error) {
	candidates := make([]string, 0, 3)
	if exeOrEmpty != "" {
		exeOrEmpty = config.CleanPath(exeOrEmpty)
		info, err := os.Stat(exeOrEmpty)
		if err == nil && !info.IsDir() {
			candidates = append(candidates, exeOrEmpty)
		}
		if err == nil && info.IsDir() {
			candidates = append(candidates, filepath.Join(exeOrEmpty, "cs2.exe"))
			candidates = append(candidates, filepath.Join(exeOrEmpty, "game", "bin", "win64", "cs2.exe"))
		}
	}
	if dirOrEmpty != "" {
		dirOrEmpty = config.CleanPath(dirOrEmpty)
		candidates = append(candidates, filepath.Join(dirOrEmpty, "cs2.exe"))
		candidates = append(candidates, filepath.Join(dirOrEmpty, "game", "bin", "win64", "cs2.exe"))
	}
	for _, candidate := range candidates {
		if strings.ToLower(filepath.Base(candidate)) == "cs2.exe" {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", fmt.Errorf("请选择包含 cs2.exe 的 CS2 安装目录")
}

func (s *Service) pickManualFile(componentID string) (string, error) {
	options := runtime.OpenDialogOptions{Title: "选择已下载文件"}
	switch componentID {
	case componentHLAE:
		options.Filters = []runtime.FileFilter{{DisplayName: "HLAE ZIP (*.zip)", Pattern: "*.zip"}}
	case componentPlugin:
		options.Filters = []runtime.FileFilter{{DisplayName: "Plugin ZIP (*.zip)", Pattern: "*.zip"}}
	case componentFFmpeg:
		options.Filters = []runtime.FileFilter{{DisplayName: "FFmpeg Archive (*.7z;*.zip)", Pattern: "*.7z;*.zip"}}
	default:
		return "", fmt.Errorf("不支持手动导入: %s", componentID)
	}
	return runtime.OpenFileDialog(s.ctx, options)
}
