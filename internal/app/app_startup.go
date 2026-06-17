package app

import (
	"fmt"

	"cs2-highlight-tool-v2/internal/envsetup"
)

// workspaceNotInitializedErr 工作目录未初始化时的统一错误信息。
func workspaceNotInitializedErr() error {
	return fmt.Errorf("请先完成工作目录初始化")
}

func (a *App) GetStartupState() envsetup.StartupState {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return envsetup.StartupState{Mode: envsetup.ModeWorkspaceInit}
	}
	return svc.GetStartupState()
}

func (a *App) RunStartupChecks() envsetup.StartupState {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return envsetup.StartupState{Mode: envsetup.ModeWorkspaceInit}
	}
	return svc.RunStartupChecks()
}

func (a *App) RetryStartupComponent(componentID string) envsetup.StartupState {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return envsetup.StartupState{Mode: envsetup.ModeWorkspaceInit}
	}
	return svc.RetryStartupComponent(componentID)
}

func (a *App) CancelStartupDownload(componentID string) envsetup.StartupState {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return envsetup.StartupState{Mode: envsetup.ModeWorkspaceInit}
	}
	return svc.CancelStartupDownload(componentID)
}

func (a *App) OpenManualDownload(componentID string) error {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return workspaceNotInitializedErr()
	}
	return svc.OpenManualDownload(componentID)
}

func (a *App) OpenExternalURL(rawURL string) error {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return workspaceNotInitializedErr()
	}
	return svc.OpenExternalURL(rawURL)
}

func (a *App) ImportManualDownload(componentID string) envsetup.StartupState {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return envsetup.StartupState{Mode: envsetup.ModeWorkspaceInit}
	}
	return svc.ImportManualDownload(componentID)
}

func (a *App) PickCS2Path() envsetup.StartupState {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return envsetup.StartupState{Mode: envsetup.ModeWorkspaceInit}
	}
	return svc.PickCS2Path()
}

func (a *App) EnterMainApp() error {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return workspaceNotInitializedErr()
	}
	return svc.EnterMainApp()
}

func (a *App) ReinstallStartupComponent(componentID string) (envsetup.StartupState, error) {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return envsetup.StartupState{Mode: envsetup.ModeWorkspaceInit}, workspaceNotInitializedErr()
	}
	return svc.ReinstallStartupComponent(componentID)
}

func (a *App) ExportStartupLogs() (string, error) {
	a.serviceMu.Lock()
	svc := a.service
	a.serviceMu.Unlock()
	if svc == nil {
		return "", workspaceNotInitializedErr()
	}
	return svc.ExportStartupLogs()
}
