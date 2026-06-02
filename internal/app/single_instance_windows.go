//go:build windows

package app

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const singleInstanceMutexName = "Local\\CS2HighlightTool"

// EnsureSingleInstance 尝试创建命名互斥体。
// 如果同名 Mutex 已存在，返回非 nil error 表示已有实例运行；
// 否则返回清理函数（调用者应在 main 退出前执行）。
func EnsureSingleInstance() (cleanup func(), err error) {
	namePtr, err := windows.UTF16PtrFromString(singleInstanceMutexName)
	if err != nil {
		return nil, fmt.Errorf("构建互斥体名称失败: %w", err)
	}

	handle, err := windows.CreateMutex(nil, false, namePtr)
	if err != nil {
		if err == windows.ERROR_ALREADY_EXISTS {
			return nil, fmt.Errorf("程序已在运行中")
		}
		return nil, fmt.Errorf("创建互斥体失败: %w", err)
	}

	// 注意：handle 必须保持打开直至进程退出，否则内核会释放该命名 Mutex。
	return func() {
		_ = windows.CloseHandle(handle)
	}, nil
}
