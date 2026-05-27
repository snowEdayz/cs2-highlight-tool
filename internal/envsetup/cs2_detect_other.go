//go:build !windows

package envsetup

import "fmt"

func detectCS2ExeFromSteam() (string, error) {
	return "", fmt.Errorf("当前系统不支持通过 Steam 自动探测 CS2 路径")
}
