package app

import (
	"fmt"
	"strings"
	"time"
)

const (
	platformClientGraceTimeout  = 5 * time.Second
	platformClientPollInterval  = 300 * time.Millisecond
)

// PlatformClientConfig describes a gaming platform client that must be closed before recording.
type PlatformClientConfig struct {
	ExeName     string
	DisplayName string
}

// PlatformClientStatus is the current running state of one platform client.
type PlatformClientStatus struct {
	ExeName     string `json:"exe_name"`
	DisplayName string `json:"display_name"`
	Running     bool   `json:"running"`
	PID         int    `json:"pid"`
}

// PlatformClientCloseResult is returned by RequestClosePlatformClient.
type PlatformClientCloseResult struct {
	ExeName string `json:"exe_name"`
	Closed  bool   `json:"closed"`
	Error   string `json:"error,omitempty"`
}

// platformClientConfigs is the authoritative list of platform clients to check.
// Add new entries here to support additional platforms.
var platformClientConfigs = []PlatformClientConfig{
	{ExeName: "完美世界竞技平台.exe", DisplayName: "完美世界竞技平台"},
	{ExeName: "5EClient.exe", DisplayName: "5E 对战平台"},
}

// CheckPlatformClients returns the running state of all known platform clients.
func (a *App) CheckPlatformClients() []PlatformClientStatus {
	result := make([]PlatformClientStatus, 0, len(platformClientConfigs))
	for _, cfg := range platformClientConfigs {
		status := PlatformClientStatus{
			ExeName:     cfg.ExeName,
			DisplayName: cfg.DisplayName,
		}
		pids, err := listPIDsByExeName(cfg.ExeName)
		if err == nil && len(pids) > 0 {
			status.Running = true
			status.PID = pids[0]
		}
		result = append(result, status)
	}
	return result
}

// RequestClosePlatformClient sends WM_CLOSE to the given platform client and waits up to
// platformClientGraceTimeout for it to exit. Returns closed=true if the process is gone.
// No force-kill is performed — the process must exit on its own.
func (a *App) RequestClosePlatformClient(exeName string) PlatformClientCloseResult {
	result := PlatformClientCloseResult{ExeName: exeName}

	var found bool
	for _, cfg := range platformClientConfigs {
		if strings.EqualFold(cfg.ExeName, exeName) {
			found = true
			break
		}
	}
	if !found {
		result.Error = fmt.Sprintf("未知的平台客户端: %s", exeName)
		return result
	}

	pids, err := listPIDsByExeName(exeName)
	if err != nil {
		result.Error = fmt.Sprintf("枚举进程失败: %v", err)
		return result
	}
	if len(pids) == 0 {
		result.Closed = true
		return result
	}

	for _, pid := range pids {
		_ = sendWMCloseToPlatformClient(pid)
	}

	deadline := time.Now().Add(platformClientGraceTimeout)
	for {
		remaining, checkErr := listPIDsByExeName(exeName)
		if checkErr == nil && len(remaining) == 0 {
			result.Closed = true
			return result
		}
		if time.Now().After(deadline) {
			break
		}
		time.Sleep(platformClientPollInterval)
	}

	return result
}
