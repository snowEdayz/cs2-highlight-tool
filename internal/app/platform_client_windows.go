//go:build windows

package app

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func listPIDsByExeName(exeName string) ([]int, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("创建进程快照失败: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if err := windows.Process32First(snapshot, &entry); err != nil {
		if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			return nil, nil
		}
		return nil, fmt.Errorf("读取进程快照失败: %w", err)
	}

	pids := make([]int, 0, 4)
	for {
		name := windows.UTF16ToString(entry.ExeFile[:])
		if strings.EqualFold(name, exeName) {
			pids = append(pids, int(entry.ProcessID))
		}
		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
				break
			}
			return nil, fmt.Errorf("遍历进程快照失败: %w", err)
		}
	}
	return pids, nil
}

func sendWMCloseToPlatformClient(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("无效 pid: %d", pid)
	}
	hwnds, err := enumWindowsByPID(uint32(pid), false)
	if err != nil {
		return err
	}
	for _, hwnd := range hwnds {
		_, _, callErr := procPostMessageW.Call(hwnd, uintptr(wmClose), 0, 0)
		if isWindowsCallError(callErr) {
			return fmt.Errorf("发送 WM_CLOSE 失败: %w", callErr)
		}
	}
	return nil
}
