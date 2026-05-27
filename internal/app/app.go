package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"cs2-highlight-tool-v2/internal/appdata"
	"cs2-highlight-tool-v2/internal/envsetup"
	"cs2-highlight-tool-v2/internal/producews"
	"cs2-highlight-tool-v2/internal/release"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx          context.Context
	exeDir       string
	dataDir      string
	migrationErr error
	service      *envsetup.Service
	produceW     *producews.Service

	produceStateMu sync.Mutex
	produceState   produceSessionState
}

func New(wailsConfigData []byte) *App {
	paths := appdata.Resolve(resolveExecutableDir())
	migrationErr := appdata.MigrateLegacyData(paths.ExeDir, paths.DataDir)
	version := release.CurrentAppVersion(wailsConfigData)
	return &App{
		exeDir:       paths.ExeDir,
		dataDir:      paths.DataDir,
		migrationErr: migrationErr,
		service:      envsetup.NewWithDataDir(paths.ExeDir, paths.DataDir, version),
		produceW:     producews.NewDefault(nil),
	}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.produceW.SetEmitter(func(name string, payload any) {
		runtime.EventsEmit(ctx, name, payload)
	})
	if err := a.produceW.Start(); err != nil {
		runtime.LogError(ctx, fmt.Sprintf("start produce websocket server failed: %v", err))
	}
	if a.migrationErr != nil {
		runtime.LogError(ctx, fmt.Sprintf("migrate legacy app data failed: %v", a.migrationErr))
	}
	a.service.Startup(ctx)
}

func (a *App) Shutdown(ctx context.Context) {
	a.stopProduceSessionWorker()
	if err := a.forceRestoreProduceEnvironmentForProduce(); err != nil {
		runtime.LogError(ctx, fmt.Sprintf("restore produce environment failed: %v", err))
	}
	if err := a.produceW.Stop(); err != nil {
		runtime.LogError(ctx, fmt.Sprintf("stop produce websocket server failed: %v", err))
	}
}

func resolveExecutableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		wd, _ := os.Getwd()
		return wd
	}
	return filepath.Dir(exePath)
}

func (a *App) dataRoot() string {
	if a != nil && a.dataDir != "" {
		return a.dataDir
	}
	if a != nil {
		return a.exeDir
	}
	return ""
}

func (a *App) dataPath(elem ...string) string {
	parts := append([]string{a.dataRoot()}, elem...)
	return filepath.Join(parts...)
}

func (a *App) configPath() string {
	return a.dataPath("config.json")
}
