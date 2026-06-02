package app

import (
	"fmt"
	"strings"

	"cs2-highlight-tool-v2/internal/config"
	"cs2-highlight-tool-v2/internal/envsetup"
	"cs2-highlight-tool-v2/internal/fivee"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *App) GetFiveEPlayerName() string {
	cfg, err := config.LoadOrCreate(a.configPath(), a.dataRoot())
	if err != nil {
		if a != nil && a.ctx != nil {
			wailsruntime.LogError(a.ctx, fmt.Sprintf("load config for 5e player name failed: %v", err))
		}
		return ""
	}
	return strings.TrimSpace(cfg.FiveEPlayerName)
}

func (a *App) ListFiveERecentMatches(playerName string, page int) (*fivee.FiveEMatchListResult, error) {
	playerName = fivee.NormalizePlayerDomainInput(playerName)
	if err := a.saveFiveEPlayerName(playerName); err != nil {
		return nil, err
	}

	result := &fivee.FiveEMatchListResult{
		PlayerName: playerName,
		Matches:    make([]fivee.FiveEMatchItem, 0),
	}
	if playerName == "" {
		return result, nil
	}

	matches, err := fivee.ListRecentMatches(playerName, page)
	if err != nil {
		return nil, err
	}
	result.Matches = matches
	return result, nil
}

func (a *App) saveFiveEPlayerName(playerName string) error {
	configPath := a.configPath()
	cfg, err := config.LoadOrCreate(configPath, a.dataRoot())
	if err != nil {
		return err
	}
	cfg.FiveEPlayerName = strings.TrimSpace(playerName)
	if err := config.Save(configPath, cfg); err != nil {
		return err
	}
	return nil
}

func (a *App) ImportFiveEMatch(matchID string) ([]string, error) {
	downloadMatchID, err := fivee.ExtractMatchID(matchID)
	if err != nil {
		return nil, err
	}
	cacheRoot := a.dataPath("demo", "5e", downloadMatchID)
	progressID := fivee.ProgressComponentID(downloadMatchID)

	stablePath, err := fivee.ImportDemo(downloadMatchID, cacheRoot, func(active bool, percent float64, indeterminate bool) {
		a.emitFiveEDownloadProgress(progressID, active, percent, indeterminate)
	})
	if err != nil {
		return nil, err
	}
	a.cleanupLegacyRawDemoCopy(stablePath)
	return []string{stablePath}, nil
}

func (a *App) emitFiveEDownloadProgress(componentID string, active bool, percent float64, indeterminate bool) {
	if a == nil || a.ctx == nil || strings.TrimSpace(componentID) == "" {
		return
	}
	wailsruntime.EventsEmit(a.ctx, "download_progress", envsetup.ProgressMessage{
		ComponentID:   componentID,
		Active:        active,
		Percent:       percent,
		Indeterminate: indeterminate,
	})
}
