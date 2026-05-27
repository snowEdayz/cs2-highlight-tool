//go:build !windows

package app

import "fmt"

func listCS2PIDs() ([]int, error) {
	return nil, fmt.Errorf("当前系统不支持枚举 cs2.exe 进程")
}

func closeCS2ProcessByPID(pid int) error {
	return fmt.Errorf("当前系统不支持按 PID 关闭 cs2 进程: pid=%d", pid)
}
