//go:build windows

package app

import (
	"errors"
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	cs2ProcessName = "cs2.exe"
	wmClose        = 0x0010
)

var (
	user32DLL                    = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows              = user32DLL.NewProc("EnumWindows")
	procGetWindowThreadProcessID = user32DLL.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible          = user32DLL.NewProc("IsWindowVisible")
	procPostMessageW             = user32DLL.NewProc("PostMessageW")
)

type enumWindowContext struct {
	pid         uint32
	visibleOnly bool
	hwnds       []uintptr
}

func listCS2PIDs() ([]int, error) {
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
		if strings.EqualFold(name, cs2ProcessName) {
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

func closeCS2ProcessByPID(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("无效 pid: %d", pid)
	}

	exited, err := waitForProcessExit(pid, 0)
	if err != nil {
		return err
	}
	if exited {
		return nil
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

	exited, err = waitForProcessExit(pid, cs2ProcessCloseGraceTimeout)
	if err != nil {
		return err
	}
	if exited {
		return nil
	}

	if err := terminateProcess(pid); err != nil {
		return err
	}

	exited, err = waitForProcessExit(pid, 2*time.Second)
	if err != nil {
		return err
	}
	if !exited {
		return fmt.Errorf("强制关闭 CS2 超时: pid=%d", pid)
	}
	return nil
}

func enumWindowsByPID(pid uint32, visibleOnly bool) ([]uintptr, error) {
	ctx := &enumWindowContext{
		pid:         pid,
		visibleOnly: visibleOnly,
		hwnds:       make([]uintptr, 0, 4),
	}

	callback := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		context := (*enumWindowContext)(unsafe.Pointer(lparam))

		var windowPID uint32
		_, _, _ = procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&windowPID)))
		if windowPID != context.pid {
			return 1
		}
		if context.visibleOnly {
			visible, _, _ := procIsWindowVisible.Call(hwnd)
			if visible == 0 {
				return 1
			}
		}

		context.hwnds = append(context.hwnds, hwnd)
		return 1
	})

	r1, _, callErr := procEnumWindows.Call(callback, uintptr(unsafe.Pointer(ctx)))
	if r1 == 0 && isWindowsCallError(callErr) {
		return nil, fmt.Errorf("枚举窗口失败: %w", callErr)
	}
	return ctx.hwnds, nil
}

func waitForProcessExit(pid int, timeout time.Duration) (bool, error) {
	handle, err := windows.OpenProcess(windows.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return true, nil
		}
		return false, fmt.Errorf("打开进程失败: %w", err)
	}
	defer windows.CloseHandle(handle)

	waitMs := durationToWaitMilliseconds(timeout)
	result, err := windows.WaitForSingleObject(handle, waitMs)
	if err != nil {
		return false, fmt.Errorf("等待进程退出失败: %w", err)
	}

	switch result {
	case uint32(windows.WAIT_OBJECT_0):
		return true, nil
	case uint32(windows.WAIT_TIMEOUT):
		return false, nil
	default:
		return false, fmt.Errorf("等待进程退出返回异常状态: %d", result)
	}
}

func terminateProcess(pid int) error {
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, uint32(pid))
	if err != nil {
		if errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
			return nil
		}
		return fmt.Errorf("打开进程失败: %w", err)
	}
	defer windows.CloseHandle(handle)

	if err := windows.TerminateProcess(handle, 1); err != nil {
		return fmt.Errorf("TerminateProcess 失败: %w", err)
	}
	return nil
}

func durationToWaitMilliseconds(timeout time.Duration) uint32 {
	if timeout <= 0 {
		return 0
	}
	ms := timeout / time.Millisecond
	if ms <= 0 {
		return 1
	}
	if ms > time.Duration(^uint32(0)-1) {
		return ^uint32(0) - 1
	}
	return uint32(ms)
}

func isWindowsCallError(err error) bool {
	if err == nil {
		return false
	}
	errno, ok := err.(syscall.Errno)
	if !ok {
		return true
	}
	return errno != 0
}
