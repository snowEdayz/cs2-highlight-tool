package app

import "cs2-highlight-tool-v2/internal/envsetup"

func (a *App) GetStartupState() envsetup.StartupState {
	return a.service.GetStartupState()
}

func (a *App) RunStartupChecks() envsetup.StartupState {
	return a.service.RunStartupChecks()
}

func (a *App) RetryStartupComponent(componentID string) envsetup.StartupState {
	return a.service.RetryStartupComponent(componentID)
}

func (a *App) OpenManualDownload(componentID string) error {
	return a.service.OpenManualDownload(componentID)
}

func (a *App) OpenExternalURL(rawURL string) error {
	return a.service.OpenExternalURL(rawURL)
}

func (a *App) ImportManualDownload(componentID string) envsetup.StartupState {
	return a.service.ImportManualDownload(componentID)
}

func (a *App) PickCS2Path() envsetup.StartupState {
	return a.service.PickCS2Path()
}

func (a *App) EnterMainApp() error {
	return a.service.EnterMainApp()
}

func (a *App) ApplySelfUpdate() envsetup.StartupState {
	return a.service.ApplySelfUpdate()
}

func (a *App) ReinstallStartupComponent(componentID string) (envsetup.StartupState, error) {
	return a.service.ReinstallStartupComponent(componentID)
}

func (a *App) ExportStartupLogs() (string, error) {
	return a.service.ExportStartupLogs()
}
