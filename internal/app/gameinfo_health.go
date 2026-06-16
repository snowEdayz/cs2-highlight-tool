package app

import (
	"fmt"
	"os"

	"cs2-highlight-tool-v2/internal/config"
	"cs2-highlight-tool-v2/internal/producegame"
)

const (
	gameInfoHealthOK          = "ok"
	gameInfoHealthNeedsRepair = "needs_repair"
	gameInfoHealthUnknown     = "unknown"
)

type GameInfoHealth struct {
	Status       string `json:"status"`
	NeedsRepair  bool   `json:"needs_repair"`
	GameInfoPath string `json:"gameinfo_path"`
	Message      string `json:"message"`
	Error        string `json:"error"`
}

func (a *App) GetGameInfoHealth() (*GameInfoHealth, error) {
	return a.readGameInfoHealth()
}

func (a *App) RepairGameInfo() (*GameInfoHealth, error) {
	health, err := a.readGameInfoHealth()
	if err != nil {
		return nil, err
	}
	if !health.NeedsRepair {
		return health, nil
	}
	contentBytes, err := os.ReadFile(health.GameInfoPath)
	if err != nil {
		return nil, fmt.Errorf("读取 gameinfo.gi 失败: %w", err)
	}
	repaired, changed := producegame.RemovePluginSearchPath(string(contentBytes))
	if !changed {
		return &GameInfoHealth{
			Status:       gameInfoHealthOK,
			NeedsRepair:  false,
			GameInfoPath: health.GameInfoPath,
			Message:      "gameinfo.gi 状态正常",
		}, nil
	}
	if err := os.WriteFile(health.GameInfoPath, []byte(repaired), 0644); err != nil {
		return nil, fmt.Errorf("修复 gameinfo.gi 失败: %w", err)
	}
	return &GameInfoHealth{
		Status:       gameInfoHealthOK,
		NeedsRepair:  false,
		GameInfoPath: health.GameInfoPath,
		Message:      "gameinfo.gi 已修复",
	}, nil
}

func (a *App) readGameInfoHealth() (*GameInfoHealth, error) {
	if a == nil || (a.dataDir == "" && a.service == nil) {
		return unknownGameInfoHealth("", fmt.Errorf("工作目录尚未初始化")), nil
	}
	cfg, err := config.LoadOrCreate(a.configPath(), a.dataRoot())
	if err != nil {
		return unknownGameInfoHealth("", fmt.Errorf("读取配置失败: %w", err)), nil
	}
	cs2Exe, err := resolveCS2ExeForLaunch(cfg)
	if err != nil {
		return unknownGameInfoHealth("", err), nil
	}
	gameInfoPath, err := producegame.ResolveGameInfoPath(cs2Exe, config.CleanPath(cfg.CS2Dir))
	if err != nil {
		return unknownGameInfoHealth("", err), nil
	}
	contentBytes, err := os.ReadFile(gameInfoPath)
	if err != nil {
		return unknownGameInfoHealth(gameInfoPath, fmt.Errorf("读取 gameinfo.gi 失败: %w", err)), nil
	}
	if producegame.HasPluginSearchPath(string(contentBytes)) {
		return &GameInfoHealth{
			Status:       gameInfoHealthNeedsRepair,
			NeedsRepair:  true,
			GameInfoPath: gameInfoPath,
			Message:      "检测到 CS2 插件搜索路径残留，可能导致正常启动游戏失败",
		}, nil
	}
	return &GameInfoHealth{
		Status:       gameInfoHealthOK,
		NeedsRepair:  false,
		GameInfoPath: gameInfoPath,
		Message:      "gameinfo.gi 状态正常",
	}, nil
}

func unknownGameInfoHealth(gameInfoPath string, err error) *GameInfoHealth {
	message := "暂未检测到 gameinfo.gi 状态"
	errText := ""
	if err != nil {
		errText = err.Error()
	}
	return &GameInfoHealth{
		Status:       gameInfoHealthUnknown,
		NeedsRepair:  false,
		GameInfoPath: gameInfoPath,
		Message:      message,
		Error:        errText,
	}
}
